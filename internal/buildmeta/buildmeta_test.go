package buildmeta

import (
	"encoding/binary"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func TestRequiredIconSizes(t *testing.T) {
	t.Parallel()

	want := []int{16, 24, 32, 48, 256}
	if !slices.Equal(RequiredIconSizes, want) {
		t.Fatalf("RequiredIconSizes = %v, want %v", RequiredIconSizes, want)
	}
}

func TestReadVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "version.go")
	if err := os.WriteFile(path, []byte("package baremark\n\nconst Version = \"1.2.3\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := ReadVersion(path)
	if err != nil {
		t.Fatalf("ReadVersion() error = %v", err)
	}
	if got.String() != "1.2.3" {
		t.Fatalf("ReadVersion() = %q, want %q", got.String(), "1.2.3")
	}
	if got.Major != 1 || got.Minor != 2 || got.Patch != 3 {
		t.Fatalf("ReadVersion() parts = %d.%d.%d", got.Major, got.Minor, got.Patch)
	}
}

func TestReadVersionRejectsInvalidSemanticVersion(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "version.go")
	if err := os.WriteFile(path, []byte("package baremark\n\nconst Version = \"1.2\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := ReadVersion(path); err == nil {
		t.Fatal("ReadVersion() error = nil, want invalid version error")
	}
}

func TestValidateProjectAssets(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeVersionFile(t, root)
	writeLogo(t, root)
	for _, path := range CanonicalIconPaths {
		writeICO(t, filepath.Join(root, filepath.FromSlash(path)), RequiredIconSizes)
	}

	meta, err := ValidateProject(root)
	if err != nil {
		t.Fatalf("ValidateProject() error = %v", err)
	}
	if meta.Version.String() != "0.1.0" {
		t.Fatalf("ValidateProject() version = %q, want %q", meta.Version.String(), "0.1.0")
	}
}

func TestValidateProjectAssetsRejectsIncompleteICO(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeVersionFile(t, root)
	writeLogo(t, root)
	for i, path := range CanonicalIconPaths {
		sizes := RequiredIconSizes
		if i == 0 {
			sizes = sizes[:len(sizes)-1]
		}
		writeICO(t, filepath.Join(root, filepath.FromSlash(path)), sizes)
	}

	if _, err := ValidateProject(root); err == nil {
		t.Fatal("ValidateProject() error = nil, want incomplete ICO error")
	}
}

func writeVersionFile(t *testing.T, root string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "version.go"), []byte("package baremark\n\nconst Version = \"0.1.0\"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeLogo(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(CanonicalLogoPath))
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 2, 1))); err != nil {
		_ = f.Close()
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
}

func writeICO(t *testing.T, path string, sizes []int) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}

	data := make([]byte, 6+16*len(sizes))
	binary.LittleEndian.PutUint16(data[2:4], 1)
	binary.LittleEndian.PutUint16(data[4:6], uint16(len(sizes)))
	for i, size := range sizes {
		offset := 6 + i*16
		if size < 256 {
			data[offset] = byte(size)
			data[offset+1] = byte(size)
		}
		data[offset+2] = 0
		data[offset+3] = 0
		binary.LittleEndian.PutUint16(data[offset+4:offset+6], 1)
		binary.LittleEndian.PutUint16(data[offset+6:offset+8], 32)
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
}
