package buildmeta

import (
	"encoding/binary"
	"strings"
	"testing"
	"unicode/utf16"
)

func TestValidateVersionResource(t *testing.T) {
	t.Parallel()
	data := syntheticVersionResource("0.1.0", "baremark-0.1.0-amd64.exe")
	if err := validateVersionResource(data, "0.1.0", "baremark-0.1.0-amd64.exe"); err != nil {
		t.Fatalf("validateVersionResource() error = %v", err)
	}
	for _, test := range []struct {
		name         string
		data         []byte
		version      string
		originalName string
		wantError    string
	}{
		{name: "wrong filename", data: data, version: "0.1.0", originalName: "wrong.exe", wantError: "version strings"},
		{name: "wrong version", data: data, version: "0.2.0", originalName: "baremark-0.1.0-amd64.exe", wantError: "fixed file/product version"},
		{name: "truncated", data: data[:4], version: "0.1.0", originalName: "baremark-0.1.0-amd64.exe", wantError: "truncated"},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateVersionResource(test.data, test.version, test.originalName)
			if err == nil || !strings.Contains(err.Error(), test.wantError) {
				t.Fatalf("validateVersionResource() error = %v, want %q", err, test.wantError)
			}
		})
	}
}

func TestValidateVersionResourcePreRelease(t *testing.T) {
	t.Parallel()
	version := "0.1.0-beta.1"
	data := syntheticVersionResource(version, "baremark-0.1.0-beta.1-amd64.exe")
	if err := validateVersionResource(data, version, "baremark-0.1.0-beta.1-amd64.exe"); err != nil {
		t.Fatalf("validateVersionResource() error = %v", err)
	}
}

func syntheticVersionResource(version, originalName string) []byte {
	fixed := make([]byte, fixedFileInfoSize)
	binary.LittleEndian.PutUint32(fixed[0:4], fixedFileInfoMagic)
	binary.LittleEndian.PutUint32(fixed[8:12], 1)
	binary.LittleEndian.PutUint32(fixed[12:16], 0)
	binary.LittleEndian.PutUint32(fixed[16:20], 1)
	binary.LittleEndian.PutUint32(fixed[20:24], 0)
	table := versionTestBlock("040904B0", nil, 1,
		versionTestBlock("FileVersion", utf16Bytes(version+"\x00"), 1),
		versionTestBlock("ProductVersion", utf16Bytes(version+"\x00"), 1),
		versionTestBlock("OriginalFilename", utf16Bytes(originalName+"\x00"), 1),
	)
	return versionTestBlock("VS_VERSION_INFO", fixed, 0, versionTestBlock("StringFileInfo", nil, 1, table))
}

func versionTestBlock(key string, value []byte, valueType uint16, children ...[]byte) []byte {
	data := make([]byte, 6)
	data = append(data, utf16Bytes(key+"\x00")...)
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	data = append(data, value...)
	for len(data)%4 != 0 {
		data = append(data, 0)
	}
	for _, child := range children {
		data = append(data, child...)
	}
	binary.LittleEndian.PutUint16(data[0:2], uint16(len(data)))
	valueLength := len(value)
	if valueType == 1 {
		valueLength /= 2
	}
	binary.LittleEndian.PutUint16(data[2:4], uint16(valueLength))
	binary.LittleEndian.PutUint16(data[4:6], valueType)
	return data
}

func utf16Bytes(value string) []byte {
	units := utf16.Encode([]rune(value))
	data := make([]byte, len(units)*2)
	for i, unit := range units {
		binary.LittleEndian.PutUint16(data[2*i:2*i+2], unit)
	}
	return data
}
