package fileio

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestFileIOLoadAndSaveMmap(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test_mmap.txt")
	testData := []byte("Gopad++ High Speed File I/O with syscall.Mmap zero-copy")

	if err := SaveFile(testPath, testData); err != nil {
		t.Fatalf("SaveFile failed: %v", err)
	}

	payload, err := LoadFile(testPath)
	if err != nil {
		t.Fatalf("LoadFile failed: %v", err)
	}
	if !payload.IsMmap() {
		t.Fatalf("expected payload to use mmap")
	}
	if !bytes.Equal(payload.Data, testData) {
		t.Fatalf("expected %q, got %q", testData, payload.Data)
	}

	// Test double close idempotency
	if err := payload.Close(); err != nil {
		t.Fatalf("first Close failed: %v", err)
	}
	if err := payload.Close(); err != nil {
		t.Fatalf("second Close failed: %v", err)
	}

	// Test Empty File
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
	if emptyPayload.IsMmap() {
		t.Fatalf("empty file should not be mmapped")
	}

	// Test directory rejection
	_, dirErr := LoadFile(tmpDir)
	if dirErr == nil {
		t.Fatalf("expected error when loading directory, got nil")
	}

	// Test non-existent file
	_, notExistErr := LoadFile(filepath.Join(tmpDir, "does_not_exist.txt"))
	if notExistErr == nil {
		t.Fatalf("expected error when loading non-existent file, got nil")
	}
}

func TestSavePreservesModeAndSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	link := filepath.Join(dir, "link")
	if err := os.WriteFile(target, []byte("old"), 0640); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("target", link); err != nil {
		t.Fatal(err)
	}
	if err := SaveFile(link, []byte("new")); err != nil {
		t.Fatal(err)
	}
	if st, err := os.Lstat(link); err != nil || st.Mode()&os.ModeSymlink == 0 {
		t.Fatal("symlink replaced", err)
	}
	st, _ := os.Stat(target)
	if st.Mode().Perm() != 0640 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
	b, _ := os.ReadFile(target)
	if string(b) != "new" {
		t.Fatalf("target %q", b)
	}
}
