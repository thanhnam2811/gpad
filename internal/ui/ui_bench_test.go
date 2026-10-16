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

func BenchmarkUI_TabBar_Candidate_A_DisconnectedButtons(b *testing.B) {
	application := app.NewApp()
	for i := 1; i <= 8; i++ {
		application.NewTab(fmt.Sprintf("doc_%d.go", i), "package main\n")
	}

	renderA := func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).FillWidth().Height(28).Background(t.Surface).Padding(2, 4).Gap(2).AlignItems(ui.Center).Children(func() {
			for i, doc := range application.Tabs {
				idx := i
				isActive := idx == application.ActiveTab
				bg := t.Surface
				if isActive {
					bg = t.Background
				}
				tabRow := ui.Row(c.Key(fmt.Sprintf("tab-a-%d", idx))).Height(24).Background(bg).Padding(2, 6).Gap(4).AlignItems(ui.Center)
				tabRow.Children(func() {
					if doc.IsDirty() {
						ui.Text(c, "●").FontSize(10).TextColor(t.Warning)
					}
					btn := ui.Button(c.Key(fmt.Sprintf("tab-a-title-%d", idx)), doc.Title).Padding(0, 4)
					if btn.Clicked() {
						application.SwitchTab(idx)
					}
					closeBtn := ui.Button(c.Key(fmt.Sprintf("tab-a-close-%d", idx)), "×").Padding(0, 2)
					if closeBtn.Clicked() {
						application.CloseTab(idx)
					}
				})
			}
		})
	}

	tester := ui.NewTester(renderA, 960, 680)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}

func BenchmarkUI_TabBar_Candidate_B_CohesiveSinglePill(b *testing.B) {
	application := app.NewApp()
	for i := 1; i <= 8; i++ {
		application.NewTab(fmt.Sprintf("doc_%d.go", i), "package main\n")
	}

	renderB := func(c *ui.Context) {
		t := c.Theme()
		ui.Row(c).FillWidth().Height(28).Background(t.Surface).Padding(2, 4).Gap(3).AlignItems(ui.Center).Children(func() {
			for i, doc := range application.Tabs {
				idx := i
				isActive := idx == application.ActiveTab
				bg := t.Surface
				if isActive {
					bg = t.Background
				}

				tabPill := ui.ButtonBase(c.Key(fmt.Sprintf("tab-b-%d", idx))).Height(24).Radius(4).Padding(0, 6).FocusRing(false).AlignItems(ui.Center)
				if !isActive && tabPill.Hovered() {
					tabPill.Background(t.SurfaceHover)
				} else {
					tabPill.Background(bg)
				}

				tabPill.ContextMenu(func(m *ui.Menu) {
					if m.Item("Close Tab").Chosen() {
						application.CloseTab(idx)
					}
					if m.Item("Close Other Tabs").Chosen() {
						application.CloseOtherTabs(idx)
					}
					if m.Item("Close Tabs to the Right").Chosen() {
						application.CloseTabsToRight(idx)
					}
					m.Separator()
					if m.Item("Copy Full Path").Chosen() {
						application.CopyFilePath(idx)
					}
				})

				var closedClicked bool
				tabPill.Children(func() {
					ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
						textElem := ui.Text(c, doc.Title).FontSize(11).SingleLine()
						if isActive {
							textElem.TextColor(t.Text).FontWeight(600)
						} else {
							textElem.TextColor(t.TextMuted)
						}

						if tabPill.Hovered() {
							closeBtn := ui.ButtonBase(c.Key(fmt.Sprintf("tab-b-close-%d", idx))).Size(14, 14).Radius(3).Center().FocusRing(false)
							if closeBtn.Hovered() {
								closeBtn.Background(t.SurfaceHover)
							}
							closeBtn.Children(func() {
								ui.Text(c, "✕").FontSize(9).TextColor(t.Text)
							})
							if closeBtn.Clicked() {
								closedClicked = true
								application.CloseTab(idx)
							}
						} else if doc.IsDirty() {
							ui.Row(c).Size(14, 14).Center().Children(func() {
								ui.Text(c, "●").FontSize(8).TextColor(t.Warning)
							})
						} else {
							ui.Box(c).Size(14, 14)
						}
					})
				})

				if !closedClicked && tabPill.Clicked() {
					application.SwitchTab(idx)
				}
			}
		})
	}

	tester := ui.NewTester(renderB, 960, 680)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tester.Frame()
	}
}

