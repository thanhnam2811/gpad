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

	l, err := StartListener(sockPath, func(path, token string) {
		openedMu.Lock()
		opened = append(opened, path)
		openedMu.Unlock()
	}, func(token string) {
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

// TestSingleInstance_ForwardsActivationToken is the Wayland focus fix: the
// running instance has to know the activation token of the process that asked
// for the file, or it cannot bring its window forward.
func TestSingleInstance_ForwardsActivationToken(t *testing.T) {
	for _, tc := range []struct {
		name, activationToken, startupID, want string
	}{
		{"shell launch", "gnome-shell/Gopad++/2545_TIME9628335", "", "gnome-shell/Gopad++/2545_TIME9628335"},
		{"glib launch", "", "4b6b6535-47bb_TIME0", "4b6b6535-47bb_TIME0"},
		{"launched by hand", "", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("XDG_ACTIVATION_TOKEN", tc.activationToken)
			t.Setenv("DESKTOP_STARTUP_ID", tc.startupID)
			sockPath := filepath.Join(t.TempDir(), "test_gpad.sock")

			var mu sync.Mutex
			got := make([]string, 0)
			l, err := StartListener(sockPath, func(path, token string) {
				mu.Lock()
				got = append(got, "open:"+token)
				mu.Unlock()
			}, func(token string) {
				mu.Lock()
				got = append(got, "focus:"+token)
				mu.Unlock()
			})
			if err != nil {
				t.Fatalf("StartListener failed: %v", err)
			}
			defer l.Close()

			if !TryForward(sockPath, []string{"test.txt"}) {
				t.Fatalf("expected TryForward to succeed")
			}
			time.Sleep(50 * time.Millisecond)

			mu.Lock()
			defer mu.Unlock()
			want := []string{"open:" + tc.want, "focus:" + tc.want}
			if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
				t.Fatalf("got %v, want %v", got, want)
			}
		})
	}
}
