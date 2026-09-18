package buildmeta

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WindowsArtifactKind identifies the Windows binary container being validated.
type WindowsArtifactKind string

const (
	WindowsArtifactSYSO WindowsArtifactKind = "syso"
	WindowsArtifactEXE  WindowsArtifactKind = "exe"
)

const (
	machineAMD64          = 0x8664
	machineARM64          = 0xaa64
	coffHeaderSize        = 20
	coffSectionHeaderSize = 40
	pe32PlusMagic         = 0x020b
	imageFileExecutable   = 0x0002
)

// ValidateWindowsArtifact checks that a generated resource object or executable
// has the expected 64-bit Windows architecture and contains a resource section.
func ValidateWindowsArtifact(path string, kind WindowsArtifactKind, architecture string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s artifact %q: %w", kind, path, err)
	}

	wantMachine, err := machineForArchitecture(architecture)
	if err != nil {
		return err
	}

	coffOffset, err := artifactCOFFOffset(data, kind)
	if err != nil {
		return fmt.Errorf("validate %s artifact %q: %w", kind, path, err)
	}
	if len(data)-coffOffset < coffHeaderSize {
		return fmt.Errorf("validate %s artifact %q: truncated COFF header", kind, path)
	}

	machine := binary.LittleEndian.Uint16(data[coffOffset : coffOffset+2])
	if machine != wantMachine {
		return fmt.Errorf("validate %s artifact %q: machine 0x%04x, want 0x%04x for %s", kind, path, machine, wantMachine, architecture)
	}

	sectionCount := int(binary.LittleEndian.Uint16(data[coffOffset+2 : coffOffset+4]))
	optionalHeaderSize := int(binary.LittleEndian.Uint16(data[coffOffset+16 : coffOffset+18]))
	characteristics := binary.LittleEndian.Uint16(data[coffOffset+18 : coffOffset+20])
	if err := validateArtifactHeader(data, kind, coffOffset, optionalHeaderSize, characteristics); err != nil {
		return fmt.Errorf("validate %s artifact %q: %w", kind, path, err)
	}

	sectionOffset := coffOffset + coffHeaderSize + optionalHeaderSize
	if sectionCount == 0 || sectionOffset > len(data) || sectionCount > (len(data)-sectionOffset)/coffSectionHeaderSize {
		return fmt.Errorf("validate %s artifact %q: truncated or empty COFF section table", kind, path)
	}
	for i := 0; i < sectionCount; i++ {
		offset := sectionOffset + i*coffSectionHeaderSize
		name := strings.TrimRight(string(data[offset:offset+8]), "\x00")
		if strings.HasPrefix(name, ".rsrc") {
			return nil
		}
	}
	return fmt.Errorf("validate %s artifact %q: resource section not found", kind, path)
}

// PublishWindowsArtifact validates a staged artifact and replaces the stable
// artifact with a same-directory rename only after validation succeeds.
func PublishWindowsArtifact(temporaryPath, outputPath string, kind WindowsArtifactKind, architecture, iconsRoot string) error {
	if filepath.Clean(filepath.Dir(temporaryPath)) != filepath.Clean(filepath.Dir(outputPath)) {
		return fmt.Errorf("temporary and stable artifacts must be in the same directory")
	}
	if err := ValidateWindowsArtifact(temporaryPath, kind, architecture); err != nil {
		return err
	}
	if iconsRoot != "" {
		if kind != WindowsArtifactEXE {
			return fmt.Errorf("icon resource validation is supported only for EXE artifacts")
		}
		if err := ValidatePEIconResources(temporaryPath, iconsRoot); err != nil {
			return err
		}
	}
	if err := os.Rename(temporaryPath, outputPath); err != nil {
		return fmt.Errorf("publish artifact %q as %q: %w", temporaryPath, outputPath, err)
	}
	return nil
}

func machineForArchitecture(architecture string) (uint16, error) {
	switch architecture {
	case "amd64":
		return machineAMD64, nil
	case "arm64":
		return machineARM64, nil
	default:
		return 0, fmt.Errorf("unsupported Windows architecture %q", architecture)
	}
}

func artifactCOFFOffset(data []byte, kind WindowsArtifactKind) (int, error) {
	switch kind {
	case WindowsArtifactSYSO:
		if len(data) >= 2 && string(data[:2]) == "MZ" {
			return 0, fmt.Errorf("expected a COFF resource object, found a PE image")
		}
		return 0, nil
	case WindowsArtifactEXE:
		if len(data) < 0x40 || string(data[:2]) != "MZ" {
			return 0, fmt.Errorf("missing DOS executable header")
		}
		peOffset := int(binary.LittleEndian.Uint32(data[0x3c:0x40]))
		if peOffset < 0 || peOffset > len(data)-4 || string(data[peOffset:peOffset+4]) != "PE\x00\x00" {
			return 0, fmt.Errorf("missing PE signature")
		}
		return peOffset + 4, nil
	default:
		return 0, fmt.Errorf("unsupported Windows artifact kind %q", kind)
	}
}

func validateArtifactHeader(data []byte, kind WindowsArtifactKind, coffOffset, optionalHeaderSize int, characteristics uint16) error {
	switch kind {
	case WindowsArtifactSYSO:
		if optionalHeaderSize != 0 {
			return fmt.Errorf("COFF resource object unexpectedly has an optional header")
		}
	case WindowsArtifactEXE:
		if characteristics&imageFileExecutable == 0 {
			return fmt.Errorf("PE image is not marked executable")
		}
		optionalHeaderOffset := coffOffset + coffHeaderSize
		if optionalHeaderSize < 2 || optionalHeaderOffset > len(data)-2 {
			return fmt.Errorf("missing PE optional header")
		}
		magic := binary.LittleEndian.Uint16(data[optionalHeaderOffset : optionalHeaderOffset+2])
		if magic != pe32PlusMagic {
			return fmt.Errorf("optional header magic 0x%04x, want PE32+ 0x%04x", magic, pe32PlusMagic)
		}
	}
	return nil
}
