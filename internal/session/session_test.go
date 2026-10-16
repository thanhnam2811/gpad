package session

import (
	"testing"
)

func TestSessionSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state := SessionState{
		ActiveIndex: 1,
		Tabs: []TabSession{
			{
				ID:         "tab-1",
				Title:      "Untitled 1",
				Path:       "",
				Content:    "Draft text for tab 1",
				CursorLine: 1,
				CursorCol:  5,
				IsDirty:    true,
			},
			{
				ID:         "tab-2",
				Title:      "main.go",
				Path:       "/tmp/main.go",
				Content:    "package main\n\nfunc main() {}\n",
				CursorLine: 3,
				CursorCol:  1,
				IsDirty:    false,
			},
		},
	}

	if err := mgr.SaveSnapshot(state); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	loaded, err := mgr.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if loaded.ActiveIndex != 1 {
		t.Fatalf("expected ActiveIndex 1, got %d", loaded.ActiveIndex)
	}
	if len(loaded.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(loaded.Tabs))
	}
	if loaded.Tabs[0].Content != "Draft text for tab 1" {
		t.Fatalf("tab 0 content mismatch: %q", loaded.Tabs[0].Content)
	}
}
