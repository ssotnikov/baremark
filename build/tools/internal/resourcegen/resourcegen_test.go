package resourcegen

import (
	"debug/pe"
	"encoding/binary"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGenerateFixedIconGroups(t *testing.T) {
	root := filepath.Clean(filepath.Join("..", "..", "..", ".."))
	for _, test := range []struct {
		arch    string
		machine uint16
	}{
		{arch: "amd64", machine: pe.IMAGE_FILE_MACHINE_AMD64},
		{arch: "arm64", machine: pe.IMAGE_FILE_MACHINE_ARM64},
	} {
		t.Run(test.arch, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "resource.syso")
			if err := Generate(root, test.arch, "0.1.0", "baremark-test.exe", output); err != nil {
				t.Fatal(err)
			}
			file, err := pe.Open(output)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			if file.Machine != test.machine {
				t.Fatalf("machine = 0x%x, want 0x%x", file.Machine, test.machine)
			}
			var resourceData []byte
			for _, section := range file.Sections {
				if section.Name == ".rsrc" || section.Name == ".rsrc$01" {
					resourceData, err = section.Data()
					if err != nil {
						t.Fatal(err)
					}
					break
				}
			}
			if resourceData == nil {
				t.Fatal("COFF resource section missing")
			}
			groupIDs, err := coffIconGroupIDs(resourceData)
			if err != nil {
				t.Fatal(err)
			}
			want := []uint32{GroupAppMain, GroupAppLight, GroupFileLight, GroupFileDark, GroupApplication}
			if !reflect.DeepEqual(groupIDs, want) {
				t.Fatalf("icon group IDs = %v, want %v", groupIDs, want)
			}
		})
	}
}

func coffIconGroupIDs(data []byte) ([]uint32, error) {
	if len(data) < 16 {
		return nil, errTruncatedResources
	}
	rootCount := int(binary.LittleEndian.Uint16(data[14:16]))
	for i := 0; i < rootCount; i++ {
		offset := 16 + i*8
		if offset+8 > len(data) {
			return nil, errTruncatedResources
		}
		if binary.LittleEndian.Uint32(data[offset:offset+4]) != 14 {
			continue
		}
		directory := int(binary.LittleEndian.Uint32(data[offset+4:offset+8]) & 0x7fffffff)
		if directory+16 > len(data) {
			return nil, errTruncatedResources
		}
		count := int(binary.LittleEndian.Uint16(data[directory+14 : directory+16]))
		ids := make([]uint32, count)
		for j := range ids {
			entry := directory + 16 + j*8
			if entry+8 > len(data) {
				return nil, errTruncatedResources
			}
			ids[j] = binary.LittleEndian.Uint32(data[entry : entry+4])
		}
		return ids, nil
	}
	return nil, errGroupMissing
}

var (
	errTruncatedResources = errors.New("truncated COFF resources")
	errGroupMissing       = errors.New("icon group resource missing")
)
