package app

import (
	"path/filepath"
	"testing"

	"gpad/internal/session"
)

func TestAppTabsAndNavigation(t *testing.T) {
	app := NewApp()
	if len(app.Tabs) != 1 {
		t.Fatalf("expected 1 tab, got %d", len(app.Tabs))
	}

	// Create tab 2
	tab2 := app.NewTab("doc2.go", "package main\n")
	if len(app.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(app.Tabs))
	}
	if app.ActiveTab != 1 {
		t.Fatalf("expected active tab 1, got %d", app.ActiveTab)
	}

	// Switch tab
	app.SwitchTab(0)
	if app.ActiveTab != 0 {
		t.Fatalf("expected active tab 0, got %d", app.ActiveTab)
	}

	// Next tab
	app.NextTab()
	if app.ActiveTab != 1 {
		t.Fatalf("expected active tab 1 after NextTab(), got %d", app.ActiveTab)
	}

	// Prev tab
	app.PrevTab()
	if app.ActiveTab != 0 {
		t.Fatalf("expected active tab 0 after PrevTab(), got %d", app.ActiveTab)
	}

	// Close tab 1
	app.CloseTab(1)
	if len(app.Tabs) != 1 {
		t.Fatalf("expected 1 tab after closing, got %d", len(app.Tabs))
	}
	_ = tab2
}

func TestAppSessionPersistence(t *testing.T) {
	tmpDir := t.TempDir()
	mgr, err := session.NewManager(filepath.Join(tmpDir, "sess"))
	if err != nil {
		t.Fatalf("session.NewManager error: %v", err)
	}

	app1 := &App{
		SessionMgr: mgr,
	}
	app1.NewTab("Note 1", "Draft content 1")
	app1.NewTab("Note 2", "Draft content 2")

	if err := app1.SaveSession(); err != nil {
		t.Fatalf("SaveSession failed: %v", err)
	}

	app2 := &App{
		SessionMgr: mgr,
	}
	if err := app2.RestoreSession(); err != nil {
		t.Fatalf("RestoreSession failed: %v", err)
	}

	if len(app2.Tabs) != 2 {
		t.Fatalf("expected 2 restored tabs, got %d", len(app2.Tabs))
	}
	if app2.Tabs[0].Content() != "Draft content 1" {
		t.Fatalf("expected tab 0 content 'Draft content 1', got %q", app2.Tabs[0].Content())
	}
}
