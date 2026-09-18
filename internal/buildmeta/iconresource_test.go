package buildmeta

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReadICOPayloads(t *testing.T) {
	t.Parallel()

	payloads := [][]byte{{1, 2, 3}, {4, 5, 6, 7}}
	data := make([]byte, 6+16*len(payloads))
	binary.LittleEndian.PutUint16(data[2:4], 1)
	binary.LittleEndian.PutUint16(data[4:6], uint16(len(payloads)))
	nextOffset := len(data)
	for i, payload := range payloads {
		offset := 6 + i*16
		data[offset] = byte(16 + i*16)
		data[offset+1] = data[offset]
		binary.LittleEndian.PutUint32(data[offset+8:offset+12], uint32(len(payload)))
		binary.LittleEndian.PutUint32(data[offset+12:offset+16], uint32(nextOffset))
		data = append(data, payload...)
		nextOffset += len(payload)
	}
	path := filepath.Join(t.TempDir(), "icon.ico")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := readICOPayloads(path)
	if err != nil {
		t.Fatalf("readICOPayloads() error = %v", err)
	}
	if err := compareIconPayloads(got, payloads); err != nil {
		t.Fatalf("compareIconPayloads() error = %v", err)
	}
}

func TestCompareIconPayloadsRejectsDifferentImage(t *testing.T) {
	t.Parallel()

	err := compareIconPayloads([][]byte{{1, 2}}, [][]byte{{1, 3}})
	if err == nil || !strings.Contains(err.Error(), "image entry 0 differs") {
		t.Fatalf("compareIconPayloads() error = %v, want image mismatch", err)
	}
}
