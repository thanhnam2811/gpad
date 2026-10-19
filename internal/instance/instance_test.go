package instance

import (
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestSingleInstance_Forward(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "test_gpad.sock")

	var openedMu sync.Mutex
	opened := make([]string, 0)
	focused := false

	l, err := StartListener(sockPath, func(path string) {
		openedMu.Lock()
		opened = append(opened, path)
		openedMu.Unlock()
	}, func() {
		focused = true
	})
	if err != nil {
		t.Fatalf("StartListener failed: %v", err)
	}
	defer l.Close()

	files := []string{"test1.txt", "test2.go"}
	ok := TryForward(sockPath, files)
	if !ok {
		t.Fatalf("expected TryForward to succeed")
	}

	time.Sleep(50 * time.Millisecond)

	openedMu.Lock()
	defer openedMu.Unlock()
	if len(opened) != 2 {
		t.Fatalf("expected 2 opened files, got %d: %v", len(opened), opened)
	}
	if !focused {
		t.Fatalf("expected focused to be true")
	}
}

func TestSingleInstance_NoListener(t *testing.T) {
	tempDir := t.TempDir()
	sockPath := filepath.Join(tempDir, "non_existent.sock")

	ok := TryForward(sockPath, []string{"test.txt"})
	if ok {
		t.Fatalf("expected TryForward to fail when no listener running")
	}
}
