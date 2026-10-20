package ui

import (
	"errors"
	"testing"

	"github.com/egoist/mygo/ui"
	"gpad/internal/app"
	"gpad/internal/buffer"
)

func TestFailedAsyncLoadRendersTerminalError(t *testing.T) {
	doc := &buffer.Document{Title: "missing.txt", Loading: true, ReadyCh: make(chan struct{})}
	application := &app.App{Tabs: []*buffer.Document{doc}}
	gui := NewUI(application)
	tester := ui.NewTester(gui.Render, 960, 680)
	tester.Frame()
	if !tester.HasText("Please wait") {
		t.Fatalf("missing loading state: %v", tester.Texts())
	}
	doc.SetLoadError(errors.New("synthetic read failure"))
	tester.Frame()
	if !tester.HasText("Unable to load file: synthetic read failure") {
		t.Fatalf("missing terminal error: %v", tester.Texts())
	}
	if tester.HasText("Please wait") {
		t.Fatal("failed load still shows loading")
	}
}
