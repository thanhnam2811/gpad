package fileio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileIOLoadAndSave(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.txt")
	testData := []byte("Gopad++ High Speed File I/O")

	if err := SaveFile(testPath, testData); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	payload, err := LoadFile(testPath)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	defer payload.Close()

	if !bytes.Equal(payload.Data, testData) {
		t.Fatalf("expected data %q, got %q", testData, payload.Data)
	}

	// Empty file test
	emptyPath := filepath.Join(tmpDir, "empty.txt")
	if err := os.WriteFile(emptyPath, []byte{}, 0o644); err != nil {
		t.Fatalf("failed to create empty file: %v", err)
	}
	emptyPayload, err := LoadFile(emptyPath)
	if err != nil {
		t.Fatalf("LoadFile empty failed: %v", err)
	}
	defer emptyPayload.Close()
	if len(emptyPayload.Data) != 0 {
		t.Fatalf("expected 0 bytes, got %d", len(emptyPayload.Data))
	}
}
