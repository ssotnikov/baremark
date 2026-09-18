package buildmeta

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const (
	resourceTypeIcon      = 3
	resourceTypeGroupIcon = 14
	resourceIDAppMain     = 1
	resourceIDAppLight    = 101
	resourceIDFileLight   = 201
	resourceIDFileDark    = 202
	resourceIDApplication = 32512
)

// ValidatePEIconResources verifies the fixed icon group IDs and exact source
// image payloads, including the primary Shell and IDI_APPLICATION dark icons.
func ValidatePEIconResources(executablePath, iconsRoot string) error {
	file, err := pe.Open(executablePath)
	if err != nil {
		return fmt.Errorf("open PE executable %q: %w", executablePath, err)
	}
	defer file.Close()

	root, err := peResourceRoot(file)
	if err != nil {
		return fmt.Errorf("read PE resources from %q: %w", executablePath, err)
	}
	groupDirectory, err := root.childDirectory(0, resourceTypeGroupIcon)
	if err != nil {
		return fmt.Errorf("read icon groups from %q: %w", executablePath, err)
	}
	groupIDs, err := root.numericDirectoryIDs(groupDirectory)
	if err != nil || len(groupIDs) == 0 {
		return fmt.Errorf("read icon groups from %q: no numeric icon groups", executablePath)
	}
	slices.Sort(groupIDs)
	wantIDs := []uint32{resourceIDAppMain, resourceIDAppLight, resourceIDFileLight, resourceIDFileDark, resourceIDApplication}
	if !slices.Equal(groupIDs, wantIDs) {
		return fmt.Errorf("icon group IDs in %q are %v, want %v", executablePath, groupIDs, wantIDs)
	}

	for _, group := range []struct {
		name string
		id   uint32
		path string
	}{
		{name: "primary Shell", id: resourceIDAppMain, path: filepath.Join(iconsRoot, "app", "BareMark-app-dark.ico")},
		{name: "app light", id: resourceIDAppLight, path: filepath.Join(iconsRoot, "app", "BareMark-app-light.ico")},
		{name: "file light", id: resourceIDFileLight, path: filepath.Join(iconsRoot, "file", "BareMark-file-light.ico")},
		{name: "file dark", id: resourceIDFileDark, path: filepath.Join(iconsRoot, "file", "BareMark-file-dark.ico")},
		{name: "IDI_APPLICATION", id: resourceIDApplication, path: filepath.Join(iconsRoot, "app", "BareMark-app-dark.ico")},
	} {
		expected, err := readICOPayloads(group.path)
		if err != nil {
			return err
		}
		actual, err := root.iconGroupPayloads(groupDirectory, group.id)
		if err != nil {
			return fmt.Errorf("read %s icon group %d from %q: %w", group.name, group.id, executablePath, err)
		}
		if err := compareIconPayloads(actual, expected); err != nil {
			return fmt.Errorf("%s icon group %d in %q does not match %q: %w", group.name, group.id, executablePath, group.path, err)
		}
	}
	return nil
}

type peResources struct {
	file *pe.File
	data []byte
}

type peResourceEntry struct {
	id        uint32
	numeric   bool
	directory bool
	offset    uint32
}

func peResourceRoot(file *pe.File) (*peResources, error) {
	optional, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || len(optional.DataDirectory) <= 2 {
		return nil, fmt.Errorf("missing PE32+ resource directory")
	}
	directory := optional.DataDirectory[2]
	if directory.VirtualAddress == 0 || directory.Size == 0 {
		return nil, fmt.Errorf("empty PE resource directory")
	}
	data, err := peDataAtRVA(file, directory.VirtualAddress, directory.Size)
	if err != nil {
		return nil, err
	}
	return &peResources{file: file, data: data}, nil
}

func (resources *peResources) entries(directoryOffset uint32) ([]peResourceEntry, error) {
	if uint64(directoryOffset)+16 > uint64(len(resources.data)) {
		return nil, fmt.Errorf("truncated resource directory at offset 0x%x", directoryOffset)
	}
	base := int(directoryOffset)
	named := int(binary.LittleEndian.Uint16(resources.data[base+12 : base+14]))
	numeric := int(binary.LittleEndian.Uint16(resources.data[base+14 : base+16]))
	count := named + numeric
	if count > (len(resources.data)-base-16)/8 {
		return nil, fmt.Errorf("truncated resource directory entries at offset 0x%x", directoryOffset)
	}

	entries := make([]peResourceEntry, 0, count)
	for i := 0; i < count; i++ {
		offset := base + 16 + i*8
		name := binary.LittleEndian.Uint32(resources.data[offset : offset+4])
		target := binary.LittleEndian.Uint32(resources.data[offset+4 : offset+8])
		entries = append(entries, peResourceEntry{
			id:        name & 0x7fffffff,
			numeric:   name&0x80000000 == 0,
			directory: target&0x80000000 != 0,
			offset:    target & 0x7fffffff,
		})
	}
	return entries, nil
}

func (resources *peResources) childDirectory(parentOffset, id uint32) (uint32, error) {
	entries, err := resources.entries(parentOffset)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if entry.numeric && entry.id == id {
			if !entry.directory {
				return 0, fmt.Errorf("resource ID %d is not a directory", id)
			}
			return entry.offset, nil
		}
	}
	return 0, fmt.Errorf("resource ID %d not found", id)
}

func (resources *peResources) numericDirectoryIDs(directoryOffset uint32) ([]uint32, error) {
	entries, err := resources.entries(directoryOffset)
	if err != nil {
		return nil, err
	}
	var ids []uint32
	for _, entry := range entries {
		if entry.numeric && entry.directory {
			ids = append(ids, entry.id)
		}
	}
	return ids, nil
}

func (resources *peResources) leafData(directoryOffset uint32) ([]byte, error) {
	entries, err := resources.entries(directoryOffset)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.directory {
			continue
		}
		if uint64(entry.offset)+16 > uint64(len(resources.data)) {
			return nil, fmt.Errorf("truncated resource data entry at offset 0x%x", entry.offset)
		}
		base := int(entry.offset)
		rva := binary.LittleEndian.Uint32(resources.data[base : base+4])
		size := binary.LittleEndian.Uint32(resources.data[base+4 : base+8])
		return peDataAtRVA(resources.file, rva, size)
	}
	return nil, fmt.Errorf("resource data entry not found")
}

func (resources *peResources) iconGroupPayloads(groupDirectory, groupID uint32) ([][]byte, error) {
	groupLanguageDirectory, err := resources.childDirectory(groupDirectory, groupID)
	if err != nil {
		return nil, err
	}
	groupData, err := resources.leafData(groupLanguageDirectory)
	if err != nil {
		return nil, err
	}
	if len(groupData) < 6 {
		return nil, fmt.Errorf("truncated icon group header")
	}
	count := int(binary.LittleEndian.Uint16(groupData[4:6]))
	if count == 0 || count > (len(groupData)-6)/14 {
		return nil, fmt.Errorf("truncated or empty icon group directory")
	}

	iconDirectory, err := resources.childDirectory(0, resourceTypeIcon)
	if err != nil {
		return nil, err
	}
	payloads := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		offset := 6 + i*14
		iconID := uint32(binary.LittleEndian.Uint16(groupData[offset+12 : offset+14]))
		languageDirectory, err := resources.childDirectory(iconDirectory, iconID)
		if err != nil {
			return nil, err
		}
		payload, err := resources.leafData(languageDirectory)
		if err != nil {
			return nil, err
		}
		payloads = append(payloads, payload)
	}
	return payloads, nil
}

func peDataAtRVA(file *pe.File, rva, size uint32) ([]byte, error) {
	end := uint64(rva) + uint64(size)
	for _, section := range file.Sections {
		sectionStart := uint64(section.VirtualAddress)
		sectionSize := uint64(section.VirtualSize)
		if uint64(section.Size) > sectionSize {
			sectionSize = uint64(section.Size)
		}
		if uint64(rva) < sectionStart || end > sectionStart+sectionSize {
			continue
		}
		data, err := section.Data()
		if err != nil {
			return nil, fmt.Errorf("read PE section %q: %w", section.Name, err)
		}
		offset := uint64(rva) - sectionStart
		if offset+uint64(size) > uint64(len(data)) {
			return nil, fmt.Errorf("RVA 0x%x exceeds raw data in PE section %q", rva, section.Name)
		}
		return data[offset : offset+uint64(size)], nil
	}
	return nil, fmt.Errorf("RVA 0x%x with size %d is outside PE sections", rva, size)
}

func readICOPayloads(path string) ([][]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read canonical icon %q: %w", path, err)
	}
	if len(data) < 6 || binary.LittleEndian.Uint16(data[0:2]) != 0 || binary.LittleEndian.Uint16(data[2:4]) != 1 {
		return nil, fmt.Errorf("canonical icon %q has an invalid ICO header", path)
	}
	count := int(binary.LittleEndian.Uint16(data[4:6]))
	if count == 0 || count > (len(data)-6)/16 {
		return nil, fmt.Errorf("canonical icon %q has a truncated or empty ICO directory", path)
	}
	payloads := make([][]byte, 0, count)
	for i := 0; i < count; i++ {
		directoryOffset := 6 + i*16
		size := uint64(binary.LittleEndian.Uint32(data[directoryOffset+8 : directoryOffset+12]))
		offset := uint64(binary.LittleEndian.Uint32(data[directoryOffset+12 : directoryOffset+16]))
		if offset+size > uint64(len(data)) {
			return nil, fmt.Errorf("canonical icon %q has an out-of-range image entry", path)
		}
		payloads = append(payloads, data[offset:offset+size])
	}
	return payloads, nil
}

func compareIconPayloads(actual, expected [][]byte) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("contains %d images, want %d", len(actual), len(expected))
	}
	for i := range expected {
		if !bytes.Equal(actual[i], expected[i]) {
			return fmt.Errorf("image entry %d differs", i)
		}
	}
	return nil
}
