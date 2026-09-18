package buildmeta

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateWindowsArtifactAcceptsMatchingArchitecture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		kind    WindowsArtifactKind
		arch    string
		machine uint16
		data    func(uint16, bool) []byte
	}{
		{name: "amd64 syso", kind: WindowsArtifactSYSO, arch: "amd64", machine: 0x8664, data: syntheticSYSO},
		{name: "arm64 syso", kind: WindowsArtifactSYSO, arch: "arm64", machine: 0xaa64, data: syntheticSYSO},
		{name: "amd64 exe", kind: WindowsArtifactEXE, arch: "amd64", machine: 0x8664, data: syntheticEXE},
		{name: "arm64 exe", kind: WindowsArtifactEXE, arch: "arm64", machine: 0xaa64, data: syntheticEXE},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "artifact")
			if err := os.WriteFile(path, test.data(test.machine, true), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := ValidateWindowsArtifact(path, test.kind, test.arch); err != nil {
				t.Fatalf("ValidateWindowsArtifact() error = %v", err)
			}
		})
	}
}

func TestValidateWindowsArtifactRejectsWrongArchitecture(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		kind WindowsArtifactKind
		data func(uint16, bool) []byte
	}{
		{name: "syso", kind: WindowsArtifactSYSO, data: syntheticSYSO},
		{name: "exe", kind: WindowsArtifactEXE, data: syntheticEXE},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "artifact")
			if err := os.WriteFile(path, test.data(0xaa64, true), 0o600); err != nil {
				t.Fatal(err)
			}
			err := ValidateWindowsArtifact(path, test.kind, "amd64")
			if err == nil || !strings.Contains(err.Error(), "machine 0xaa64, want 0x8664") {
				t.Fatalf("ValidateWindowsArtifact() error = %v, want architecture mismatch", err)
			}
		})
	}
}

func TestValidateWindowsArtifactRejectsMissingResources(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name string
		kind WindowsArtifactKind
		data func(uint16, bool) []byte
	}{
		{name: "syso", kind: WindowsArtifactSYSO, data: syntheticSYSO},
		{name: "exe", kind: WindowsArtifactEXE, data: syntheticEXE},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(t.TempDir(), "artifact")
			if err := os.WriteFile(path, test.data(0x8664, false), 0o600); err != nil {
				t.Fatal(err)
			}
			err := ValidateWindowsArtifact(path, test.kind, "amd64")
			if err == nil || !strings.Contains(err.Error(), "resource section") {
				t.Fatalf("ValidateWindowsArtifact() error = %v, want missing resource section", err)
			}
		})
	}
}

func TestPublishWindowsArtifactReplacesStableFileAfterValidation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	temporaryPath := filepath.Join(dir, "temporary.exe")
	outputPath := filepath.Join(dir, "stable.exe")
	want := syntheticEXE(0x8664, true)
	if err := os.WriteFile(temporaryPath, want, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, []byte("old artifact"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := PublishWindowsArtifact(temporaryPath, outputPath, WindowsArtifactEXE, "amd64", ""); err != nil {
		t.Fatalf("PublishWindowsArtifact() error = %v", err)
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatal("published artifact does not match validated temporary artifact")
	}
	if _, err := os.Stat(temporaryPath); !os.IsNotExist(err) {
		t.Fatalf("temporary artifact still exists after publication: %v", err)
	}
}

func TestPublishWindowsArtifactPreservesStableFileOnValidationFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	temporaryPath := filepath.Join(dir, "temporary.exe")
	outputPath := filepath.Join(dir, "stable.exe")
	old := []byte("last known good artifact")
	if err := os.WriteFile(temporaryPath, syntheticEXE(0xaa64, true), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, old, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := PublishWindowsArtifact(temporaryPath, outputPath, WindowsArtifactEXE, "amd64", ""); err == nil {
		t.Fatal("PublishWindowsArtifact() error = nil, want architecture mismatch")
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(old) {
		t.Fatalf("stable artifact changed after failed validation: got %q, want %q", got, old)
	}
}

func TestPublishWindowsArtifactPreservesStableFileOnPrimaryIconFailure(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	temporaryPath := filepath.Join(dir, "temporary.exe")
	outputPath := filepath.Join(dir, "stable.exe")
	old := []byte("last known good artifact")
	if err := os.WriteFile(temporaryPath, syntheticEXE(0x8664, true), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outputPath, old, 0o600); err != nil {
		t.Fatal(err)
	}

	err := PublishWindowsArtifact(temporaryPath, outputPath, WindowsArtifactEXE, "amd64", filepath.Join(dir, "missing.ico"))
	if err == nil {
		t.Fatal("PublishWindowsArtifact() error = nil, want primary icon validation error")
	}
	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != string(old) {
		t.Fatalf("stable artifact changed after primary icon validation failed: got %q, want %q", got, old)
	}
}

func syntheticSYSO(machine uint16, withResources bool) []byte {
	data := make([]byte, 20+40)
	binary.LittleEndian.PutUint16(data[0:2], machine)
	binary.LittleEndian.PutUint16(data[2:4], 1)
	sectionName := ".data"
	if withResources {
		sectionName = ".rsrc$01"
	}
	copy(data[20:28], sectionName)
	return data
}

func syntheticEXE(machine uint16, withResources bool) []byte {
	const (
		peOffset           = 0x80
		optionalHeaderSize = 0xf0
	)
	coffOffset := peOffset + 4
	sectionOffset := coffOffset + 20 + optionalHeaderSize
	data := make([]byte, sectionOffset+40)
	copy(data[0:2], "MZ")
	binary.LittleEndian.PutUint32(data[0x3c:0x40], peOffset)
	copy(data[peOffset:peOffset+4], "PE\x00\x00")
	binary.LittleEndian.PutUint16(data[coffOffset:coffOffset+2], machine)
	binary.LittleEndian.PutUint16(data[coffOffset+2:coffOffset+4], 1)
	binary.LittleEndian.PutUint16(data[coffOffset+16:coffOffset+18], optionalHeaderSize)
	binary.LittleEndian.PutUint16(data[coffOffset+18:coffOffset+20], 0x0002)
	binary.LittleEndian.PutUint16(data[coffOffset+20:coffOffset+22], 0x020b)
	sectionName := ".data"
	if withResources {
		sectionName = ".rsrc"
	}
	copy(data[sectionOffset:sectionOffset+8], sectionName)
	return data
}
