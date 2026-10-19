package ui

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

func BenchmarkUI_Menu_Candidate_A_DynamicRebuild(b *testing.B) {
	application := newTestApp(b)
	gui := NewUI(application)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rebuilding menu tree on each cycle
		menu := BuildAppMenu(application, gui, nil)
		_ = menu.Items()
	}
}

func BenchmarkUI_Menu_Candidate_B_StaticCachedMenu(b *testing.B) {
	application := newTestApp(b)
	gui := NewUI(application)
	menu := BuildAppMenu(application, gui, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cached menu inspection and dispatch
		items := menu.Items()
		_ = items
	}
}

func BenchmarkUI_InWindowPopoverMenu_RenderOpen(b *testing.B) {
	application := newTestApp(b)
	gui := NewUI(application)
	gui.activeMenu = "file" // Menu popup actively rendered in OpenGL

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}

func BenchmarkUI_InWindowTabContextMenu_RenderOpen(b *testing.B) {
	application := newTestApp(b)
	application.NewTab("main.go", "package main\n")
	gui := NewUI(application)
	gui.tabContextMenuOpen = true
	gui.tabContextMenuIdx = 0

	tester := ui.NewTester(gui.Render, 960, 680)

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}
