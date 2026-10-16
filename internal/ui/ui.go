package ui

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
	"gpad/internal/app"
)

// UI represents the native UI renderer.
type UI struct {
	app      *app.App
	isDark   bool
	themeSet bool
}

// NewUI initializes the UI renderer.
func NewUI(a *app.App) *UI {
	u := &UI{
		app:    a,
		isDark: detectGnomeDarkMode(),
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
	// Sync theme
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
	if c.Shortcut(ui.Cmd, ui.KeyS) {
		_ = u.app.SaveActiveFile()
	}
	if c.Shortcut(ui.Cmd, ui.KeyF) {
		u.app.SearchOpen = !u.app.SearchOpen
	}
	if u.app.SearchOpen && c.Shortcut(0, ui.KeyEscape) {
		u.app.SearchOpen = false
	}

	// Main container - 0px margins, occupies full window
	ui.Column(c).Fill().Padding(0).Gap(0).Children(func() {
		// 1. Compact Tab Bar (Height: 28px)
		u.renderTabBar(c, t)

		// 2. Quick Find / Replace Bar (if active)
		if u.app.SearchOpen {
			u.renderSearchBar(c, t)
		}

		// 3. Main Code Editor Viewport (Fill remainder, 0px margins)
		u.renderEditor(c)

		// 4. Micro Status Bar (Height: 20px)
		u.renderStatusBar(c, t)
	})
}

func (u *UI) renderTabBar(c *ui.Context, t *ui.Theme) {
	tabBarBg := t.Surface
	activeTabBg := t.Background

	ui.Row(c).FillWidth().Height(28).Background(tabBarBg).Padding(2, 4).Gap(2).AlignItems(ui.Center).Children(func() {
		for i, doc := range u.app.Tabs {
			idx := i
			isActive := idx == u.app.ActiveTab
			bg := tabBarBg
			if isActive {
				bg = activeTabBg
			}

			// Individual compact tab item
			tabRow := ui.Row(c.Key(fmt.Sprintf("tab-%d", idx))).Height(24).Background(bg).Padding(2, 6).Gap(4).AlignItems(ui.Center)
			tabRow.Children(func() {
				// Dirty indicator dot
				if doc.IsDirty() {
					ui.Text(c, "●").FontSize(10).TextColor(t.Warning)
				}

				// Tab title button
				title := doc.Title
				if title == "" {
					title = "Untitled"
				}
				btn := ui.Button(c.Key(fmt.Sprintf("tab-title-%d", idx)), title).Padding(0, 4)
				if btn.Clicked() {
					u.app.SwitchTab(idx)
				}

				// Close tab button
				closeBtn := ui.Button(c.Key(fmt.Sprintf("tab-close-%d", idx)), "×").Padding(0, 2)
				if closeBtn.Clicked() {
					u.app.CloseTab(idx)
				}
			})
		}

		// New Tab Button "+"
		newTabBtn := ui.Button(c.Key("new-tab-btn"), "+").Width(24).Height(24).Padding(0, 0)
		if newTabBtn.Clicked() {
			u.app.NewTab("", "")
		}
	})
}

func (u *UI) renderSearchBar(c *ui.Context, t *ui.Theme) {
	ui.Row(c).FillWidth().Height(28).Background(t.SurfaceHover).Padding(2, 8).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.Text(c, "Find:").FontSize(12).TextColor(t.TextMuted)
		if ui.TextInput(c.Key("search-input"), &u.app.SearchQuery).Width(240).Changed() {
			u.app.PerformSearch(u.app.SearchQuery)
		}

		countText := fmt.Sprintf("%d matches (%d µs)", u.app.SearchResults.Count, u.app.SearchResults.Duration.Microseconds())
		ui.Text(c, countText).FontSize(12).TextColor(t.Accent)

		closeBtn := ui.Button(c.Key("search-close-btn"), "✕").Padding(0, 4)
		if closeBtn.Clicked() {
			u.app.SearchOpen = false
		}
	})
}

func (u *UI) renderEditor(c *ui.Context) {
	doc := u.app.ActiveDocument()
	if doc == nil || doc.State == nil {
		ui.Column(c).Fill().Center().Children(func() {
			ui.Text(c, "No document open")
		})
		return
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
