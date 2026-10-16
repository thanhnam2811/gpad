package app

import (
	"path/filepath"
	"testing"

	"gpad/internal/session"
)

func TestAppTabsAndNavigation(t *testing.T) {
	app := NewApp()
	defer app.Stop()

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

func TestAppSessionPersistenceAndRecovery(t *testing.T) {
	tmpDir := t.TempDir()
	sessDir := filepath.Join(tmpDir, "sess")
	mgr1, err := session.NewManager(sessDir)
	if err != nil {
		t.Fatalf("session.NewManager error: %v", err)
	}

	app1 := &App{
		SessionMgr: mgr1,
	}
	app1.NewTab("Untitled 1", "Initial draft note")

	// User types in Untitled 1
	app1.Tabs[0].Buffer.Edit(struct{ Start, End int; Text string }{Start: 0, End: 0, Text: "Important: "})

	// AutoSave detects typing
	app1.CheckAutoSave()

	// Simulate app quit
	if err := mgr1.Close(); err != nil {
		t.Fatalf("mgr1.Close failed: %v", err)
	}

	// Simulate reopening Gopad++
	mgr2, err := session.NewManager(sessDir)
	if err != nil {
		t.Fatalf("mgr2 NewManager failed: %v", err)
	}
	defer mgr2.Close()

	app2 := &App{
		SessionMgr: mgr2,
	}
	if err := app2.RestoreSession(); err != nil {
		t.Fatalf("RestoreSession failed: %v", err)
	}

	if len(app2.Tabs) != 1 {
		t.Fatalf("expected 1 restored tab, got %d", len(app2.Tabs))
	}
	expectedContent := "Important: Initial draft note"
	if app2.Tabs[0].Content() != expectedContent {
		t.Fatalf("expected tab content %q, got %q", expectedContent, app2.Tabs[0].Content())
	}
	if !app2.Tabs[0].IsDirty() {
		t.Fatalf("restored draft should be marked dirty")
	}
}
