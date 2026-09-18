package buildmeta

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
)

const (
	resourceTypeVersion  = 16
	resourceTypeManifest = 24
	fixedFileInfoSize    = 52
	fixedFileInfoMagic   = 0xFEEF04BD
)

// ValidatePEResourceMetadata checks the embedded manifest and version data
// against the canonical build inputs before an executable is published.
func ValidatePEResourceMetadata(executablePath, manifestPath, version, originalName string) error {
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("read canonical manifest %q: %w", manifestPath, err)
	}
	if !bytes.Contains(manifest, []byte("PerMonitorV2")) {
		return fmt.Errorf("canonical manifest %q does not declare PerMonitorV2", manifestPath)
	}
	file, err := pe.Open(executablePath)
	if err != nil {
		return fmt.Errorf("open PE executable %q: %w", executablePath, err)
	}
	defer file.Close()
	resources, err := peResourceRoot(file)
	if err != nil {
		return fmt.Errorf("read PE resources from %q: %w", executablePath, err)
	}
	embeddedManifest, err := resourceData(resources, resourceTypeManifest, 1)
	if err != nil {
		return fmt.Errorf("read embedded manifest from %q: %w", executablePath, err)
	}
	if !bytes.Equal(embeddedManifest, manifest) {
		return fmt.Errorf("embedded manifest in %q differs from %q", executablePath, manifestPath)
	}
	versionData, err := resourceData(resources, resourceTypeVersion, 1)
	if err != nil {
		return fmt.Errorf("read embedded version info from %q: %w", executablePath, err)
	}
	if err := validateVersionResource(versionData, version, originalName); err != nil {
		return fmt.Errorf("validate embedded version info in %q: %w", executablePath, err)
	}
	return nil
}

func resourceData(resources *peResources, resourceType, id uint32) ([]byte, error) {
	typeDirectory, err := resources.childDirectory(0, resourceType)
	if err != nil {
		return nil, err
	}
	languageDirectory, err := resources.childDirectory(typeDirectory, id)
	if err != nil {
		return nil, err
	}
	return resources.leafData(languageDirectory)
}

type versionBlock struct {
	key      string
	value    []byte
	children []byte
}

func parseVersionBlock(data []byte) (versionBlock, int, error) {
	if len(data) < 8 {
		return versionBlock{}, 0, fmt.Errorf("truncated version block")
	}
	length := int(binary.LittleEndian.Uint16(data[0:2]))
	valueLength := int(binary.LittleEndian.Uint16(data[2:4]))
	valueType := binary.LittleEndian.Uint16(data[4:6])
	if length < 8 || length > len(data) {
		return versionBlock{}, 0, fmt.Errorf("invalid version block length %d", length)
	}
	data = data[:length]
	keyEnd := 6
	for keyEnd+1 < len(data) && binary.LittleEndian.Uint16(data[keyEnd:keyEnd+2]) != 0 {
		keyEnd += 2
	}
	if keyEnd+1 >= len(data) {
		return versionBlock{}, 0, fmt.Errorf("unterminated version block key")
	}
	key := decodeUTF16(data[6:keyEnd])
	valueStart := alignFour(keyEnd + 2)
	if valueType == 1 {
		valueLength *= 2
	}
	if valueStart > len(data) || valueLength > len(data)-valueStart {
		return versionBlock{}, 0, fmt.Errorf("truncated value for version block %q", key)
	}
	childrenStart := alignFour(valueStart + valueLength)
	if childrenStart > len(data) {
		childrenStart = len(data)
	}
	return versionBlock{key: key, value: data[valueStart : valueStart+valueLength], children: data[childrenStart:]}, length, nil
}

func alignFour(value int) int { return (value + 3) &^ 3 }

func decodeUTF16(data []byte) string {
	units := make([]uint16, len(data)/2)
	for i := range units {
		units[i] = binary.LittleEndian.Uint16(data[2*i : 2*i+2])
	}
	return string(utf16.Decode(units))
}

func versionChildren(data []byte) ([]versionBlock, error) {
	var children []versionBlock
	for len(data) >= 2 && binary.LittleEndian.Uint16(data[:2]) != 0 {
		child, length, err := parseVersionBlock(data)
		if err != nil {
			return nil, err
		}
		children = append(children, child)
		next := alignFour(length)
		if next > len(data) {
			return nil, fmt.Errorf("truncated version block padding")
		}
		data = data[next:]
	}
	return children, nil
}

func validateVersionResource(data []byte, version, originalName string) error {
	root, _, err := parseVersionBlock(data)
	if err != nil {
		return err
	}
	if root.key != "VS_VERSION_INFO" || len(root.value) != fixedFileInfoSize {
		return fmt.Errorf("missing VS_VERSION_INFO fixed file info")
	}
	if binary.LittleEndian.Uint32(root.value[0:4]) != fixedFileInfoMagic {
		return fmt.Errorf("invalid fixed file info signature")
	}
	matches := semanticVersion.FindStringSubmatch(version)
	if matches == nil {
		return fmt.Errorf("unsupported expected version %q", version)
	}
	var numbers [3]uint64
	for i := range numbers {
		numbers[i], err = strconv.ParseUint(matches[i+1], 10, 16)
		if err != nil {
			return fmt.Errorf("invalid expected version %q: %w", version, err)
		}
	}
	wantMS := uint32(numbers[0]<<16 | numbers[1])
	wantLS := uint32(numbers[2] << 16)
	for _, offset := range []int{8, 16} {
		if binary.LittleEndian.Uint32(root.value[offset:offset+4]) != wantMS || binary.LittleEndian.Uint32(root.value[offset+4:offset+8]) != wantLS {
			return fmt.Errorf("fixed file/product version does not match %q", version)
		}
	}
	rootChildren, err := versionChildren(root.children)
	if err != nil {
		return err
	}
	for _, child := range rootChildren {
		if child.key != "StringFileInfo" {
			continue
		}
		tables, err := versionChildren(child.children)
		if err != nil {
			return err
		}
		for _, table := range tables {
			stringsInTable, err := versionChildren(table.children)
			if err != nil {
				return err
			}
			values := make(map[string]string, len(stringsInTable))
			for _, entry := range stringsInTable {
				values[entry.key] = strings.TrimRight(decodeUTF16(entry.value), "\x00")
			}
			if values["FileVersion"] == version && values["ProductVersion"] == version && values["OriginalFilename"] == originalName {
				return nil
			}
		}
	}
	return fmt.Errorf("version strings do not match version %q and original filename %q", version, originalName)
}
