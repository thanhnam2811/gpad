package ui

import (
	"testing"

	"gpad/internal/app"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// newTestApp builds an App with a throwaway session directory. newTestApp(t)
// restores and auto-saves the developer's real session, so using it here would
// both write test tabs into it and make every number below depend on whatever
// that developer happened to have open.
func newTestApp(tb testing.TB) *app.App {
	tb.Helper()
	return app.NewAppWithSessionDir(tb.TempDir())
}

func TestUIHeadlessComponents(t *testing.T) {
	application := newTestApp(t)
	doc := application.NewTab("hello.go", "package main\n\nfunc main() {}\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	// Verify tab title rendered
	box, found := tester.Find("hello.go")
	if !found || box.W == 0 {
		t.Fatalf("expected tab title 'hello.go' rendered on screen")
	}

	// Verify status bar content
	if !tester.HasText("UTF-8") {
		t.Fatalf("expected 'UTF-8' in status bar, texts: %v", tester.Texts())
	}

	// Test dirty dot
	doc.Buffer.Edit(struct {
		Start, End int
		Text       string
	}{Start: 0, End: 0, Text: "// "})
	tester.Frame()
	if !tester.HasText("●") {
		t.Fatalf("expected dirty dot '●' when document is dirty, texts: %v", tester.Texts())
	}

	// Test native editor find
	gui.OpenFind()
	doc.State.Find("main")
	tester.Frame()

	count, _ := doc.State.FindMatches()
	if count != 2 {
		t.Fatalf("expected 2 matches for 'main', got %d", count)
	}

	// Test FindNext
	gui.FindNext()
	tester.Frame()

	// Test GoToLine
	gui.GoToLine()
	tester.Frame()
}

func TestMenuHoverNoOverlap(t *testing.T) {
	application := newTestApp(t)
	application.NewTab("main.go", "package main\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	// 1. Click "File" menu
	if err := tester.Click("File"); err != nil {
		t.Fatalf("Click('File') failed: %v", err)
	}

	if !tester.HasText("New Tab") {
		t.Fatalf("expected 'New Tab' visible in File menu, texts: %v", tester.Texts())
	}

	// 2. Move hover over "Edit" menu button
	editBox, found := tester.Find("Edit")
	if !found {
		t.Fatalf("could not find 'Edit' button")
	}
	tester.Move(editBox.X+editBox.W/2, editBox.Y+editBox.H/2)

	// In the exact frame where Edit is hovered:
	// "Undo" MUST be visible, and "New Tab" MUST NOT be visible!
	// If both are visible simultaneously, they are overlapping!
	hasNewTab := tester.HasText("New Tab")
	hasUndo := tester.HasText("Undo")

	if hasNewTab && hasUndo {
		t.Fatalf("BUG REPRODUCED: Both File menu ('New Tab') and Edit menu ('Undo') are simultaneously open and overlapping!")
	}
	if !hasUndo {
		t.Fatalf("expected 'Undo' visible when hovering Edit menu, texts: %v", tester.Texts())
	}
}

func TestEditorContextMenuMultipleRightClicks(t *testing.T) {
	application := newTestApp(t)
	application.NewTab("main.go", "package main\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	mockItems := []*mygo.MenuItemSnapshot{
		{ID: 1, Label: "Cut", Visible: true, Enabled: true},
		{ID: 2, Label: "Copy", Visible: true, Enabled: true},
		{ID: 3, Label: "Paste", Visible: true, Enabled: true},
	}

	// 1. First right click at (100, 100)
	gui.ShowEditorContextMenu(mockItems, 100, 100, func(int) {})
	tester.Frame()

	if !tester.HasText("Cut") {
		t.Fatalf("expected context menu item 'Cut' visible after first right click")
	}
	if gui.editorContextMenu.X != 100 || gui.editorContextMenu.Y != 100 {
		t.Fatalf("expected position (100, 100), got (%f, %f)", gui.editorContextMenu.X, gui.editorContextMenu.Y)
	}

	// 2. Move pointer in content file and right click again at (400, 400)
	tester.Move(400, 400)
	tester.RightClickAt(400, 400)
	tester.Frame()

	// The old menu at (100, 100) MUST close, and if a new menu opened it must be at (400, 400)
	if gui.editorContextMenu.Open && (gui.editorContextMenu.X == 100 && gui.editorContextMenu.Y == 100) {
		t.Fatalf("BUG REPRODUCED: Old context menu at (100, 100) remained open and was not replaced by new menu at (400, 400)!")
	}
}

func TestAboutInWindowModal(t *testing.T) {
	application := newTestApp(t)
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	// 1. Open Help menu
	if err := tester.Click("Help"); err != nil {
		t.Fatalf(`Click("Help") failed: %v`, err)
	}

	// 2. Click About Gopad++ item
	if err := tester.Click("About Gopad++"); err != nil {
		t.Fatalf(`Click("About Gopad++") failed: %v`, err)
	}

	// 3. Modal must be open in-window (OpenGL GPU, 0 GTK dialog)
	if !gui.aboutOpen {
		t.Fatalf("expected gui.aboutOpen to be true")
	}

	tester.Frame()

	// 4. Verify in-window modal rendered texts
	if !tester.HasText("v0.1.0-alpha • High-Density Native Text Editor") {
		t.Fatalf("expected version info in About modal, texts: %v", tester.Texts())
	}
	if !tester.HasText("Pure Go & mygo/ui (Purego GPU OpenGL)") {
		t.Fatalf("expected engine info in About modal, texts: %v", tester.Texts())
	}

	// 5. Click OK button inside modal to close
	if err := tester.Click("OK"); err != nil {
		t.Fatalf(`Click("OK") failed: %v`, err)
	}

	if gui.aboutOpen {
		t.Fatalf("expected gui.aboutOpen to be false after clicking OK")
	}
}

func TestToggleWordWrap(t *testing.T) {
	application := newTestApp(t)
	doc := application.NewTab("wrap.txt", "This is a very long line that will wrap when soft wrap is enabled.")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	// Initial state: Wrap: Off
	if doc.State.IsSoftWrap() {
		t.Fatalf("expected wrap off initially")
	}
	if !tester.HasText("Wrap: Off") {
		t.Fatalf("expected 'Wrap: Off' in status bar, texts: %v", tester.Texts())
	}

	// Toggle wrap via gui.ToggleWordWrap()
	gui.ToggleWordWrap()
	tester.Frame()

	if !doc.State.IsSoftWrap() {
		t.Fatalf("expected wrap on after toggle")
	}
	if !tester.HasText("Wrap: On") {
		t.Fatalf("expected 'Wrap: On' in status bar, texts: %v", tester.Texts())
	}

	// Toggle wrap off
	gui.ToggleWordWrap()
	tester.Frame()

	if doc.State.IsSoftWrap() {
		t.Fatalf("expected wrap off after second toggle")
	}
	if !tester.HasText("Wrap: Off") {
		t.Fatalf("expected 'Wrap: Off' in status bar, texts: %v", tester.Texts())
	}
}

func TestZoom(t *testing.T) {
	application := newTestApp(t)
	doc := application.NewTab("zoom.txt", "package main\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()

	// Initial state: 100% zoom
	if !tester.HasText("100%") {
		t.Fatalf("expected '100%%' in status bar, texts: %v", tester.Texts())
	}

	// Zoom in
	gui.ZoomIn()
	tester.Frame()

	if gui.fontSize != 14 {
		t.Fatalf("expected fontSize 14, got %f", gui.fontSize)
	}
	if doc.State.Font.Size != 14 {
		t.Fatalf("expected doc.State.Font.Size 14, got %f", doc.State.Font.Size)
	}
	if !tester.HasText("108%") {
		t.Fatalf("expected '108%%' in status bar, texts: %v", tester.Texts())
	}

	// Zoom out
	gui.ZoomOut()
	tester.Frame()

	if gui.fontSize != 13 {
		t.Fatalf("expected fontSize 13, got %f", gui.fontSize)
	}
	if !tester.HasText("100%") {
		t.Fatalf("expected '100%%' in status bar, texts: %v", tester.Texts())
	}

	// Zoom in multiple times and then reset
	gui.ZoomIn()
	gui.ZoomIn()
	gui.ZoomIn()
	tester.Frame()
	if gui.fontSize != 16 {
		t.Fatalf("expected fontSize 16, got %f", gui.fontSize)
	}

	gui.ZoomReset()
	tester.Frame()
	if gui.fontSize != 13 {
		t.Fatalf("expected fontSize 13 after reset, got %f", gui.fontSize)
	}
	if !tester.HasText("100%") {
		t.Fatalf("expected '100%%' in status bar after reset, texts: %v", tester.Texts())
	}
}
