package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
	"gpad/internal/app"
)

func BenchmarkUI_RenderSteadyFrame(b *testing.B) {
	application := app.NewApp()
	application.NewTab("main.go", "package main\n\nfunc main() {\n\tprintln(\"Hello Gopad++\")\n}\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}

func BenchmarkUI_RenderTypingFrame(b *testing.B) {
	application := app.NewApp()
	doc := application.NewTab("typing.txt", "Initial line 1\nInitial line 2\n")
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate typing edit
		doc.Buffer.Edit(struct{ Start, End int; Text string }{Start: 0, End: 0, Text: "a"})
		tester.Frame()
	}
}

func BenchmarkUI_RenderSwitchTabFrame(b *testing.B) {
	application := app.NewApp()
	for i := 1; i <= 5; i++ {
		application.NewTab(fmt.Sprintf("tab_%d.go", i), fmt.Sprintf("package p%d\n// line 2\n// line 3\n", i))
	}
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		application.SwitchTab(i % 5)
		tester.Frame()
	}
}

func BenchmarkUI_RenderLargeDocumentFrame(b *testing.B) {
	application := app.NewApp()
	var codeBuilder strings.Builder
	for i := 1; i <= 10000; i++ {
		codeBuilder.WriteString(fmt.Sprintf("func fn%d() { println(%d) }\n", i, i))
	}
	application.NewTab("large.go", codeBuilder.String())
	gui := NewUI(application)

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}
