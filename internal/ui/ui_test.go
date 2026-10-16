package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
	"gpad/internal/app"
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
	doc.Buffer.Edit(struct{ Start, End int; Text string }{Start: 0, End: 0, Text: "// "})
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
