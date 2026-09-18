package resourcegen

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"reflect"

	"github.com/akavel/rsrc/binutil"
	"github.com/akavel/rsrc/coff"
	"github.com/akavel/rsrc/ico"
	goversioninfo "github.com/josephspurrier/goversioninfo"
)

const (
	GroupAppMain     = 1
	GroupAppLight    = 101
	GroupFileLight   = 201
	GroupFileDark    = 202
	GroupApplication = 32512
)

type iconSpec struct {
	groupID      uint16
	firstImageID uint16
	path         string
}

// Generate creates a COFF resource object for one Windows architecture.
func Generate(root, arch, version, originalName, output string) error {
	if arch != "amd64" && arch != "arm64" {
		return fmt.Errorf("unsupported architecture %q", arch)
	}
	versionJSON, err := os.ReadFile(filepath.Join(root, "build", "windows", "versioninfo.json"))
	if err != nil {
		return fmt.Errorf("read version info: %w", err)
	}
	var vi goversioninfo.VersionInfo
	if err := vi.ParseJSON(versionJSON); err != nil {
		return fmt.Errorf("parse version info: %w", err)
	}
	fileVersion, err := goversioninfo.NewFileVersion(version)
	if err != nil {
		return fmt.Errorf("parse version %q: %w", version, err)
	}
	vi.StringFileInfo.FileVersion = version
	vi.StringFileInfo.ProductVersion = version
	vi.StringFileInfo.OriginalFilename = originalName
	vi.FixedFileInfo.FileVersion = fileVersion
	vi.FixedFileInfo.ProductVersion = fileVersion
	vi.Build()
	vi.Walk()

	manifest, err := os.ReadFile(filepath.Join(root, "build", "windows", "baremark.manifest"))
	if err != nil {
		return fmt.Errorf("read manifest: %w", err)
	}
	resource := coff.NewRSRC()
	if err := resource.Arch(arch); err != nil {
		return err
	}
	resource.AddResource(16, 1, bytes.NewReader(vi.Buffer.Bytes()))
	resource.AddResource(coff.RT_MANIFEST, 1, bytes.NewReader(manifest))

	icons := []iconSpec{
		{GroupAppMain, 1001, filepath.Join(root, "assets", "icons", "app", "BareMark-app-dark.ico")},
		{GroupAppLight, 1101, filepath.Join(root, "assets", "icons", "app", "BareMark-app-light.ico")},
		{GroupFileLight, 1201, filepath.Join(root, "assets", "icons", "file", "BareMark-file-light.ico")},
		{GroupFileDark, 1301, filepath.Join(root, "assets", "icons", "file", "BareMark-file-dark.ico")},
	}
	var mainGroup []byte
	for _, spec := range icons {
		group, err := addIconImages(resource, spec)
		if err != nil {
			return err
		}
		resource.AddResource(coff.RT_GROUP_ICON, spec.groupID, bytes.NewReader(group))
		if spec.groupID == GroupAppMain {
			mainGroup = group
		}
	}
	resource.AddResource(coff.RT_GROUP_ICON, GroupApplication, bytes.NewReader(mainGroup))
	resource.Freeze()

	file, err := os.Create(output)
	if err != nil {
		return fmt.Errorf("create SYSO %q: %w", output, err)
	}
	writer := binutil.Writer{W: file}
	binutil.Walk(resource, func(v reflect.Value, _ string) error {
		if binutil.Plain(v.Kind()) {
			writer.WriteLE(v.Interface())
			return nil
		}
		if sized, ok := v.Interface().(binutil.SizedReader); ok {
			writer.WriteFromSized(sized)
			return binutil.WALK_SKIP
		}
		return nil
	})
	writeErr := writer.Err
	if err := file.Close(); writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		_ = os.Remove(output)
		return fmt.Errorf("write SYSO %q: %w", output, writeErr)
	}
	return nil
}

func addIconImages(resource *coff.Coff, spec iconSpec) ([]byte, error) {
	data, err := os.ReadFile(spec.path)
	if err != nil {
		return nil, fmt.Errorf("read icon %q: %w", spec.path, err)
	}
	entries, err := ico.DecodeHeaders(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode icon %q: %w", spec.path, err)
	}
	if len(entries) == 0 || len(entries) > 0xffff-int(spec.firstImageID) {
		return nil, fmt.Errorf("invalid image count in %q", spec.path)
	}
	var group bytes.Buffer
	if err := binary.Write(&group, binary.LittleEndian, ico.ICONDIR{Type: 1, Count: uint16(len(entries))}); err != nil {
		return nil, err
	}
	for index, entry := range entries {
		start := uint64(entry.ImageOffset)
		end := start + uint64(entry.BytesInRes)
		if end > uint64(len(data)) {
			return nil, fmt.Errorf("image %d exceeds icon %q", index, spec.path)
		}
		imageID := spec.firstImageID + uint16(index)
		resource.AddResource(coff.RT_ICON, imageID, bytes.NewReader(data[start:end]))
		if err := binary.Write(&group, binary.LittleEndian, entry.IconDirEntryCommon); err != nil {
			return nil, err
		}
		if err := binary.Write(&group, binary.LittleEndian, imageID); err != nil {
			return nil, err
		}
	}
	return group.Bytes(), nil
}
