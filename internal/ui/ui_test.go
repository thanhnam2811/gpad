package ui

import (
	"testing"

	"gpad/internal/app"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func TestUIHeadlessComponents(t *testing.T) {
	application := app.NewApp()
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
	application := app.NewApp()
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
	application := app.NewApp()
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
