package ui

import (
	"fmt"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// renderMenuBar renders the ultra-compact 22px in-window top menu bar.
func (u *UI) renderMenuBar(c *ui.Context, t *ui.Theme) {
	ui.Row(c.Key("in-window-menubar")).FillWidth().Height(22).Background(t.Surface).Padding(0, 4).Gap(1).AlignItems(ui.Center).Children(func() {
		// 1. File Menu
		u.renderMenuButton(c, t, "menu-btn-file", "File", func(m *ui.Menu) {
			if m.Item("New Tab").Shortcut(ui.Cmd, ui.KeyN).Chosen() {
				u.app.NewTab("", "")
			}
			if m.Item("Open File...").Shortcut(ui.Cmd, ui.KeyO).Chosen() {
				paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
					Title: "Open File - Gopad++",
				})
				if err == nil && len(paths) > 0 {
					_ = u.app.OpenFile(paths[0])
				}
			}
			if m.Item("Save").Shortcut(ui.Cmd, ui.KeyS).Chosen() {
				doc := u.app.ActiveDocument()
				if doc != nil {
					if doc.Path == "" {
						path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
							Title:       "Save As - Gopad++",
							DefaultPath: doc.Title + ".txt",
						})
						if err == nil && path != "" {
							_ = u.app.SaveActiveFileAs(path)
						}
					} else {
						_ = u.app.SaveActiveFile()
					}
				}
			}
			if m.Item("Save As...").Shortcut(ui.Cmd|ui.Shift, ui.KeyS).Chosen() {
				doc := u.app.ActiveDocument()
				if doc != nil {
					defaultPath := doc.Path
					if defaultPath == "" {
						defaultPath = doc.Title + ".txt"
					}
					path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
						Title:       "Save As - Gopad++",
						DefaultPath: defaultPath,
					})
					if err == nil && path != "" {
						_ = u.app.SaveActiveFileAs(path)
					}
				}
			}
			m.Separator()
			if m.Item("Close Tab").Shortcut(ui.Cmd, ui.KeyW).Chosen() {
				u.app.CloseTab(u.app.ActiveTab)
			}
			if m.Item("Close Other Tabs").Chosen() {
				u.app.CloseOtherTabs(u.app.ActiveTab)
			}
			if m.Item("Close Tabs to the Right").Chosen() {
				u.app.CloseTabsToRight(u.app.ActiveTab)
			}
			if m.Item("Close All Tabs").Chosen() {
				u.app.CloseAllTabs()
			}
			m.Separator()
			if m.Item("Exit").Shortcut(ui.Cmd, ui.KeyQ).Chosen() {
				mygo.App.Quit()
			}
		})

		// 2. Edit Menu
		u.renderMenuButton(c, t, "menu-btn-edit", "Edit", func(m *ui.Menu) {
			if m.Item("Undo").Shortcut(ui.Cmd, ui.KeyZ).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.Undo)
				}
			}
			if m.Item("Redo").Shortcut(ui.Cmd, ui.KeyY).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.Redo)
				}
			}
			m.Separator()
			if m.Item("Cut").Shortcut(ui.Cmd, ui.KeyX).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.Cut)
				}
			}
			if m.Item("Copy").Shortcut(ui.Cmd, ui.KeyC).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.Copy)
				}
			}
			if m.Item("Paste").Shortcut(ui.Cmd, ui.KeyV).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.Paste)
				}
			}
			if m.Item("Select All").Shortcut(ui.Cmd, ui.KeyA).Chosen() {
				if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
					doc.State.Run(editor.SelectAll)
				}
			}
		})

		// 3. Search Menu
		u.renderMenuButton(c, t, "menu-btn-search", "Search", func(m *ui.Menu) {
			if m.Item("Find...").Shortcut(ui.Cmd, ui.KeyF).Chosen() {
				u.OpenFind()
			}
			if m.Item("Replace...").Shortcut(ui.Cmd, ui.KeyH).Chosen() {
				u.OpenFindReplace()
			}
			if m.Item("Find Next").Shortcut(0, ui.KeyF3).Chosen() {
				u.FindNext()
			}
			if m.Item("Find Previous").Shortcut(ui.Shift, ui.KeyF3).Chosen() {
				u.FindPrevious()
			}
			m.Separator()
			if m.Item("Go to Line...").Shortcut(ui.Cmd, ui.KeyG).Chosen() {
				u.GoToLine()
			}
			if m.Item("Select All Matches").Shortcut(ui.Alt, ui.KeyEnter).Chosen() {
				u.SelectAllMatches()
			}
		})

		// 4. View Menu
		u.renderMenuButton(c, t, "menu-btn-view", "View", func(m *ui.Menu) {
			if m.Item("Toggle Word Wrap").Chosen() {
				// Wrap toggle
			}
			m.Separator()
			if m.Item("Zoom In").Shortcut(ui.Cmd, ui.KeyEqual).Chosen() {
				// Zoom in
			}
			if m.Item("Zoom Out").Shortcut(ui.Cmd, ui.KeyMinus).Chosen() {
				// Zoom out
			}
		})

		// 5. Document Menu
		u.renderMenuButton(c, t, "menu-btn-doc", "Document", func(m *ui.Menu) {
			doc := u.app.ActiveDocument()
			lineEnding := "Line Ending: LF"
			encoding := "Encoding: UTF-8"
			if doc != nil {
				lineEnding = fmt.Sprintf("Line Ending: %s", doc.LineEndingDesc())
				encoding = fmt.Sprintf("Encoding: %s", doc.Encoding)
			}
			m.Item(encoding).Disabled(true)
			m.Item(lineEnding).Disabled(true)
			m.Separator()
			if m.Item("Copy Document Path").Chosen() {
				u.app.CopyFilePath(u.app.ActiveTab)
			}
		})

		// 6. Help Menu
		u.renderMenuButton(c, t, "menu-btn-help", "Help", func(m *ui.Menu) {
			if m.Item("About Gopad++").Chosen() {
				_, _ = mygo.Dialog.Message(mygo.MessageOptions{
					Title:   "About Gopad++",
					Message: "Gopad++ (gpad) v0.1.0-alpha",
					Detail:  "A high-density native text editor for Linux (GNOME 46) written in Go.\nOpenGL GPU Rendering • Strictly Zero Webview • Notepad++ Ergonomics",
					Type:    mygo.MessageInfo,
				})
			}
		})
	})
}

func (u *UI) renderMenuButton(c *ui.Context, t *ui.Theme, key, label string, build func(m *ui.Menu)) {
	btn := ui.ButtonBase(c.Key(key)).Height(20).Padding(1, 7).Radius(3).FocusRing(false).AlignItems(ui.Center)
	if btn.Hovered() {
		btn.Background(t.SurfaceHover)
	}
	btn.Children(func() {
		ui.Text(c, label).FontSize(11).TextColor(t.Text)
	})
	btn.Menu(build)
}
