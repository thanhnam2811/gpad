package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"gpad/internal/app"
	"gpad/internal/buffer"
	"gpad/internal/search"
)

// UI represents the native UI renderer.
type UI struct {
	app              *app.App
	isDark           bool
	themeSet         bool
	largeFindOpen    bool
	largeFindQuery   string
	largeMatchIndex  int
}

// NewUI initializes the UI renderer.
func NewUI(a *app.App) *UI {
	u := &UI{
		app:    a,
		isDark: detectGnomeDarkMode(),
	}

	// Register native close confirmation handler
	a.OnConfirmClose = func(doc *buffer.Document) app.CloseAction {
		title := doc.Title
		if title == "" {
			title = "Untitled"
		}
		res, err := mygo.Dialog.Message(mygo.MessageOptions{
			Title:         "Gopad++",
			Message:       fmt.Sprintf("Do you want to save changes to \"%s\"?", title),
			Detail:        "Your changes will be lost if you don't save them.",
			Type:          mygo.MessageWarning,
			Buttons:       []string{"Save", "Don't Save", "Cancel"},
			DefaultButton: 0,
			CancelButton:  2,
		})
		if err != nil {
			return app.CloseActionCancel
		}
		switch res.Button {
		case 0:
			if doc.Path == "" {
				path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
					Title:       "Save As - Gopad++",
					DefaultPath: doc.Title + ".txt",
				})
				if err != nil || path == "" {
					return app.CloseActionCancel
				}
				_ = a.SaveActiveFileAs(path)
				return app.CloseActionDiscard // already saved to disk
			}
			return app.CloseActionSave
		case 1:
			return app.CloseActionDiscard
		default:
			return app.CloseActionCancel
		}
	}

	return u
}

// detectGnomeDarkMode queries GNOME 46 color-scheme setting.
func detectGnomeDarkMode() bool {
	out, err := exec.Command("gsettings", "get", "org.gnome.desktop.interface", "color-scheme").Output()
	if err == nil {
		str := strings.TrimSpace(string(out))
		if strings.Contains(str, "prefer-dark") {
			return true
		}
	}
	return true // Default dark for GNOME modern desktop
}

// Render builds the frame on each render cycle.
func (u *UI) Render(c *ui.Context) {
	// Theme synchronization with GNOME 46
	t := c.Theme()
	if u.isDark && !t.Dark {
		c.SetTheme(ui.DarkTheme())
		t = c.Theme()
	}

	// Global keyboard shortcuts
	if c.Shortcut(ui.Cmd, ui.KeyN) {
		u.app.NewTab("", "")
	}
	if c.Shortcut(ui.Cmd, ui.KeyW) {
		u.app.CloseTab(u.app.ActiveTab)
	}
	if c.Shortcut(ui.Cmd, ui.KeyTab) {
		u.app.NextTab()
	}
	if c.Shortcut(ui.Cmd|ui.Shift, ui.KeyTab) {
		u.app.PrevTab()
	}
	for i := 0; i < 9; i++ {
		if c.Shortcut(ui.Cmd, ui.Key(int(ui.Key1)+i)) {
			if i < len(u.app.Tabs) {
				u.app.SwitchTab(i)
			}
		}
	}
	if c.Shortcut(ui.Cmd, ui.KeyS) {
		_ = u.app.SaveActiveFile()
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		u.OpenFind()
	}
	if c.Shortcut(ui.Cmd, ui.KeyH) {
		u.OpenFindReplace()
	}
	if c.Shortcut(0, ui.KeyF3) {
		u.FindNext()
	}
	if c.Shortcut(ui.Shift, ui.KeyF3) {
		u.FindPrevious()
	}
	if c.Shortcut(ui.Cmd, ui.KeyG) {
		u.GoToLine()
	}
	if c.Shortcut(ui.Alt, ui.KeyEnter) || c.Shortcut(ui.Cmd|ui.Shift, ui.KeyL) {
		u.SelectAllMatches()
	}

	// Main container - 0px margins, occupies full window
	ui.Column(c).Fill().Padding(0).Gap(0).Children(func() {
		// 1. In-Window High-Density Top Menu Bar (Height: 22px)
		u.renderMenuBar(c, t)

		// 2. SVG Vector Micro Toolbar (Height: 24px)
		u.renderToolbar(c, t)

		// 3. Compact Tab Bar (Height: 28px)
		u.renderTabBar(c, t)

		// 4. Main Code Editor Viewport (Fill remainder, 0px margins)
		// Built-in native find, replace, and goToLine automatically float docked at top-right
		u.renderEditor(c)

		// 5. Micro Status Bar (Height: 20px)
		u.renderStatusBar(c, t)
	})
}

func (u *UI) renderTabBar(c *ui.Context, t *ui.Theme) {
	tabBarBg := t.Surface
	activeTabBg := t.Background

	ui.Row(c).FillWidth().Height(28).Background(tabBarBg).Padding(2, 4).Gap(3).AlignItems(ui.Center).Children(func() {
		for i, doc := range u.app.Tabs {
			idx := i
			isActive := idx == u.app.ActiveTab
			bg := tabBarBg
			if isActive {
				bg = activeTabBg
			}

			// Individual cohesive tab pill
			tabPill := ui.ButtonBase(c.Key(fmt.Sprintf("tab-pill-%d", idx))).Height(24).Radius(4).Padding(0, 6).FocusRing(false).AlignItems(ui.Center)
			isHovered := tabPill.Hovered()
			if !isActive && isHovered {
				tabPill.Background(t.SurfaceHover)
			} else {
				tabPill.Background(bg)
			}

			// Tab Context Menu (Right Click)
			tabPill.ContextMenu(func(m *ui.Menu) {
				if m.Item("Close Tab").Chosen() {
					u.app.CloseTab(idx)
				}
				if m.Item("Close Other Tabs").Chosen() {
					u.app.CloseOtherTabs(idx)
				}
				if m.Item("Close Tabs to the Right").Chosen() {
					u.app.CloseTabsToRight(idx)
				}
				m.Separator()
				if m.Item("Copy Full Path").Chosen() {
					u.app.CopyFilePath(idx)
				}
			})

			// Content of tab pill
			var closedClicked bool
			tabPill.Children(func() {
				ui.Row(c).AlignItems(ui.Center).Gap(6).Children(func() {
					title := doc.Title
					if title == "" {
						title = "Untitled"
					}
					if doc.IsLoading() {
						title = title + " (loading...)"
					}

					textElem := ui.Text(c, title).FontSize(11).SingleLine()
					if isActive {
						textElem.TextColor(t.Text).FontWeight(600)
					} else {
						textElem.TextColor(t.TextMuted)
					}

					// Indicator slot with hover duality:
					// Hovered: close button '✕'
					// Not hovered & Dirty: warning dot '●'
					// Not hovered & Clean: placeholder spacer
					if isHovered {
						closeBtn := ui.ButtonBase(c.Key(fmt.Sprintf("tab-close-%d", idx))).Size(14, 14).Radius(3).Center().FocusRing(false)
						if closeBtn.Hovered() {
							closeBtn.Background(t.SurfaceHover)
						}
						closeBtn.Children(func() {
							ui.Text(c, "✕").FontSize(9).TextColor(t.Text)
						})
						if closeBtn.Clicked() {
							closedClicked = true
							u.app.CloseTab(idx)
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
				u.app.SwitchTab(idx)
			}
		}

		// New Tab Button "+"
		newTabBtn := ui.ButtonBase(c.Key("new-tab-btn")).Size(24, 24).Radius(4).Center().FocusRing(false)
		if newTabBtn.Hovered() {
			newTabBtn.Background(t.SurfaceHover)
		}
		newTabBtn.Children(func() {
			ui.Text(c, "+").FontSize(14).TextColor(t.TextMuted)
		})
		if newTabBtn.Clicked() {
			u.app.NewTab("", "")
		}

		// Empty space: double-clicking opens a new tab
		emptySpace := ui.Row(c.Key("tab-empty-space")).FillWidth().Height(24)
		if emptySpace.DoubleClicked() {
			u.app.NewTab("", "")
		}
	})
}

// OpenFind opens the find bar in the active editor.
// For large files (>= 5MB) or while loading, uses Zero-Lock Large-File Search to prevent UI freeze.
func (u *UI) OpenFind() {
	doc := u.app.ActiveDocument()
	if doc == nil {
		return
	}
	if doc.Len() >= 5*1024*1024 || doc.IsLoading() {
		u.largeFindOpen = true
		return
	}
	if doc.State != nil {
		doc.State.Run(editor.Find)
	}
}

// OpenFindReplace opens the find & replace bar in the active editor.
func (u *UI) OpenFindReplace() {
	doc := u.app.ActiveDocument()
	if doc == nil {
		return
	}
	if doc.Len() >= 5*1024*1024 || doc.IsLoading() {
		u.largeFindOpen = true
		return
	}
	if doc.State != nil {
		doc.State.Run(editor.FindReplace)
	}
}

// FindNext jumps to the next match.
func (u *UI) FindNext() {
	doc := u.app.ActiveDocument()
	if doc == nil {
		return
	}
	if u.largeFindOpen {
		res := u.app.GetSearchResults()
		if len(res.Matches) > 0 {
			u.largeMatchIndex = (u.largeMatchIndex + 1) % len(res.Matches)
			u.jumpToMatch(doc, res.Matches[u.largeMatchIndex])
		}
		return
	}
	if doc.State != nil {
		doc.State.Run(editor.FindNext)
	}
}

// FindPrevious jumps to the previous match.
func (u *UI) FindPrevious() {
	doc := u.app.ActiveDocument()
	if doc == nil {
		return
	}
	if u.largeFindOpen {
		res := u.app.GetSearchResults()
		if len(res.Matches) > 0 {
			u.largeMatchIndex = (u.largeMatchIndex - 1 + len(res.Matches)) % len(res.Matches)
			u.jumpToMatch(doc, res.Matches[u.largeMatchIndex])
		}
		return
	}
	if doc.State != nil {
		doc.State.Run(editor.FindPrevious)
	}
}

// GoToLine opens the go-to-line bar in the active editor.
func (u *UI) GoToLine() {
	if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
		doc.State.Run(editor.GoToLine)
	}
}

// SelectAllMatches selects all occurrences of the query as multi-carets.
func (u *UI) SelectAllMatches() {
	if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
		doc.State.Run(editor.SelectAllMatches)
	}
}

func (u *UI) jumpToMatch(doc *buffer.Document, m search.Match) {
	if doc == nil || doc.State == nil {
		return
	}
	if m.Line > 0 {
		doc.State.GoTo(editor.Pos{Line: m.Line - 1, Col: max(0, m.Col - 1)})
		doc.State.Select(m.Offset, m.Offset+m.Length)
	}
}

func (u *UI) renderEditor(c *ui.Context) {
	doc := u.app.ActiveDocument()
	if doc == nil {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "No document open")
		})
		return
	}

	if doc.IsLoading() {
		ui.Column(c).Fill().Center().Children(func() {
			sizeMB := float64(doc.LoadingSize) / (1024 * 1024)
			loadingText := fmt.Sprintf("Loading \"%s\" (%.1f MB)... Please wait", doc.Title, sizeMB)
			ui.Text(c, loadingText).FontSize(13).TextColor(c.Theme().TextMuted)
		})
		return
	}

	if doc.State == nil {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "No document open")
		})
		return
	}

	t := c.Theme()

	// If large file search bar is active, render docked bar above editor
	if u.largeFindOpen {
		ui.Row(c.Key("large-find-docked")).FillWidth().Height(28).Background(t.Surface).Border(1, t.Border).Padding(2, 8).Gap(6).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "Find:").FontSize(11).TextColor(t.TextMuted)

			input := ui.TextInput(c.Key("large-find-input"), &u.largeFindQuery).Width(180)
			if input.Changed() {
				u.app.TriggerSearch(u.largeFindQuery)
				u.largeMatchIndex = 0
			}

			res := u.app.GetSearchResults()
			countStr := "0/0"
			if res.Count > 0 {
				countStr = fmt.Sprintf("%d/%d", u.largeMatchIndex+1, res.Count)
			}
			ui.Text(c, countStr).FontSize(11).TextColor(t.TextMuted)

			prevBtn := ui.ButtonBase(c.Key("large-find-prev")).Size(20, 20).Radius(3).Center()
			if prevBtn.Hovered() {
				prevBtn.Background(t.SurfaceHover)
			}
			prevBtn.Children(func() { ui.Text(c, "▲").FontSize(8) })
			if prevBtn.Clicked() && len(res.Matches) > 0 {
				u.largeMatchIndex = (u.largeMatchIndex - 1 + len(res.Matches)) % len(res.Matches)
				u.jumpToMatch(doc, res.Matches[u.largeMatchIndex])
			}

			nextBtn := ui.ButtonBase(c.Key("large-find-next")).Size(20, 20).Radius(3).Center()
			if nextBtn.Hovered() {
				nextBtn.Background(t.SurfaceHover)
			}
			nextBtn.Children(func() { ui.Text(c, "▼").FontSize(8) })
			if nextBtn.Clicked() && len(res.Matches) > 0 {
				u.largeMatchIndex = (u.largeMatchIndex + 1) % len(res.Matches)
				u.jumpToMatch(doc, res.Matches[u.largeMatchIndex])
			}

			closeBtn := ui.ButtonBase(c.Key("large-find-close")).Size(20, 20).Radius(3).Center()
			if closeBtn.Hovered() {
				closeBtn.Background(t.SurfaceHover)
			}
			closeBtn.Children(func() { ui.Text(c, "✕").FontSize(9) })
			if closeBtn.Clicked() {
				u.largeFindOpen = false
			}
		})
	}

	// Main code editor occupying entire central viewport
	editor.View(c, doc.State).Fill()
}

func (u *UI) renderStatusBar(c *ui.Context, t *ui.Theme) {
	doc := u.app.ActiveDocument()
	line, col := 1, 1
	enc := "UTF-8"
	le := "LF"
	sizeStr := "0 B"

	if doc != nil {
		line, col = doc.CaretPosition()
		enc = doc.Encoding
		le = doc.LineEndingDesc()
		docLen := doc.Len()
		if docLen < 1024 {
			sizeStr = fmt.Sprintf("%d B", docLen)
		} else if docLen < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(docLen)/1024.0)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(docLen)/(1024.0*1024.0))
		}
	}

	latUs := u.app.LastOpLatency.Microseconds()
	statusText := fmt.Sprintf("Ln %d, Col %d  |  %s  |  %s  |  %s  |  Latency: %d µs", line, col, enc, le, sizeStr, latUs)

	ui.Row(c).FillWidth().Height(20).Background(t.Surface).Padding(1, 8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, statusText).FontSize(11).TextColor(t.TextMuted)
	})
}
