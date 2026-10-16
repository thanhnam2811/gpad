package ui

import (
	"testing"

	"gpad/internal/app"
)

func BenchmarkUI_Menu_Candidate_A_DynamicRebuild(b *testing.B) {
	application := app.NewApp()
	gui := NewUI(application)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rebuilding menu tree on each cycle
		menu := BuildAppMenu(application, gui, nil)
		_ = menu.Items()
	}
}

func BenchmarkUI_Menu_Candidate_B_StaticCachedMenu(b *testing.B) {
	application := app.NewApp()
	gui := NewUI(application)
	menu := BuildAppMenu(application, gui, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cached menu inspection and dispatch
		items := menu.Items()
		_ = items
	}
}
