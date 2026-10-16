package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gpad/internal/search"
	"gpad/internal/session"
)

func TestAppTabsAndNavigation(t *testing.T) {
	app := NewAppWithSessionDir(t.TempDir())
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

func TestAppLargeFileSessionZeroContentInWAL(t *testing.T) {
	tmpDir := t.TempDir()
	sessDir := filepath.Join(tmpDir, "sess")
	mgr1, err := session.NewManager(sessDir)
	if err != nil {
		t.Fatalf("session.NewManager error: %v", err)
	}

	// Create a large 10MB test file on disk
	largeFilePath := filepath.Join(tmpDir, "large_sample.log")
	largeData := bytes.Repeat([]byte("2026-10-16 LOG LINE WITH PADDING FOR PERFORMANCE VERIFICATION.\n"), 160000) // ~10MB
	if err := os.WriteFile(largeFilePath, largeData, 0o644); err != nil {
		t.Fatalf("write large file failed: %v", err)
	}

	app1 := &App{
		SessionMgr:   mgr1,
		SearchEngine: search.NewAdaptiveSearchDispatcher(),
	}
	if err := app1.OpenFile(largeFilePath); err != nil {
		t.Fatalf("OpenFile failed: %v", err)
	}

	// Wait for background ingestion on app1
	deadline1 := time.Now().Add(2 * time.Second)
	for app1.Tabs[0].IsLoading() && time.Now().Before(deadline1) {
		time.Sleep(5 * time.Millisecond)
	}

	// Ensure WAL size is under 2KB (must NEVER serialize 10MB content into session WAL)
	walInfo, err := os.Stat(filepath.Join(sessDir, "session.wal"))
	if err != nil {
		t.Fatalf("stat session.wal failed: %v", err)
	}
	if walInfo.Size() > 2048 {
		t.Fatalf("session.wal bloated! Expected < 2048 bytes, got %d bytes", walInfo.Size())
	}

	_ = mgr1.Close()

	// Reopen app and verify clean zero-copy disk reload
	mgr2, err := session.NewManager(sessDir)
	if err != nil {
		t.Fatalf("mgr2 failed: %v", err)
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

	// Wait for background async reload on app2
	deadline2 := time.Now().Add(2 * time.Second)
	for app2.Tabs[0].IsLoading() && time.Now().Before(deadline2) {
		time.Sleep(5 * time.Millisecond)
	}

	if app2.Tabs[0].Len() != len(largeData) {
		t.Fatalf("expected restored tab length %d, got %d", len(largeData), app2.Tabs[0].Len())
	}
	if app2.Tabs[0].IsDirty() {
		t.Fatalf("clean file loaded from disk should not be dirty")
	}
}

func TestAppDebouncedSearch(t *testing.T) {
	app := &App{
		SearchEngine: search.NewAdaptiveSearchDispatcher(),
	}
	app.NewTab("search_doc.txt", "The quick brown fox jumps over the lazy dog. Another fox is fast.")

	// Rapid typing simulation
	app.TriggerSearch("f")
	app.TriggerSearch("fo")
	app.TriggerSearch("fox")

	if !app.IsSearchingActive() {
		t.Fatalf("expected IsSearching true while debouncing")
	}

	// Wait for debounce and async execution
	time.Sleep(350 * time.Millisecond)

	if app.IsSearchingActive() {
		t.Fatalf("expected IsSearching false after search completes")
	}
	res := app.GetSearchResults()
	if res.Count != 2 {
		t.Fatalf("expected 2 matches for 'fox', got %d", res.Count)
	}
}

func TestAppTabLifecycleAdvanced(t *testing.T) {
	app := NewAppWithSessionDir(t.TempDir())
	defer app.Stop()

	// Initially 1 tab
	app.NewTab("tab1.txt", "content 1")
	app.NewTab("tab2.txt", "content 2")
	app.NewTab("tab3.txt", "content 3")
	app.NewTab("tab4.txt", "content 4")

	if len(app.Tabs) != 5 {
		t.Fatalf("expected 5 tabs, got %d", len(app.Tabs))
	}

	// CloseTabsToRight of index 2 (keeps 0, 1, 2; removes 3, 4)
	app.CloseTabsToRight(2)
	if len(app.Tabs) != 3 {
		t.Fatalf("expected 3 tabs after CloseTabsToRight(2), got %d", len(app.Tabs))
	}

	// CloseOtherTabs keeps only index 1
	app.CloseOtherTabs(1)
	if len(app.Tabs) != 1 {
		t.Fatalf("expected 1 tab after CloseOtherTabs(1), got %d", len(app.Tabs))
	}
	if app.Tabs[0].Title != "tab1.txt" {
		t.Fatalf("expected remaining tab to be 'tab1.txt', got %q", app.Tabs[0].Title)
	}

	// CloseAllTabs resets to 1 fresh untitled tab
	app.CloseAllTabs()
	if len(app.Tabs) != 1 {
		t.Fatalf("expected 1 tab after CloseAllTabs(), got %d", len(app.Tabs))
	}
}

func TestAppSaveActiveFileAs(t *testing.T) {
	app := NewAppWithSessionDir(t.TempDir())
	defer app.Stop()

	doc := app.NewTab("scratch.txt", "Hello SaveAs")
	savePath := filepath.Join(t.TempDir(), "target_saved.txt")

	if err := app.SaveActiveFileAs(savePath); err != nil {
		t.Fatalf("SaveActiveFileAs failed: %v", err)
	}

	if doc.Path != savePath {
		t.Fatalf("expected doc.Path to update to %q, got %q", savePath, doc.Path)
	}
	if doc.IsDirty() {
		t.Fatalf("expected doc to not be dirty after SaveActiveFileAs")
	}

	readBytes, err := os.ReadFile(savePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	if string(readBytes) != "Hello SaveAs" {
		t.Fatalf("expected content %q, got %q", "Hello SaveAs", string(readBytes))
	}
}

