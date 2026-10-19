package ui

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// renderToolbar renders the ultra-compact 24px micro toolbar with GPU vector SVG icons.
func (u *UI) renderToolbar(c *ui.Context, t *ui.Theme) {
	ui.Row(c.Key("micro-toolbar")).FillWidth().Height(24).Background(t.Surface).Padding(1, 6).Gap(2).AlignItems(ui.Center).Children(func() {
		// Group 1: File Actions
		u.renderToolBtn(c, t, "tb-new", IconNew, "New Tab (Ctrl+N)", func() {
			u.app.NewTab("", "")
		})
		u.renderToolBtn(c, t, "tb-open", IconOpen, "Open File... (Ctrl+O)", func() {
			paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
				Title: "Open File - Gopad++",
			})
			if err == nil && len(paths) > 0 {
				_ = u.app.OpenFile(paths[0])
			}
		})
		u.renderToolBtn(c, t, "tb-save", IconSave, "Save (Ctrl+S)", func() {
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
		})
		u.renderToolBtn(c, t, "tb-saveall", IconSaveAll, "Save All", func() {
			for _, d := range u.app.Tabs {
				if d != nil && d.IsDirty() && d.Path != "" {
					_ = u.app.SaveActiveFile()
				}
			}
		})
		u.renderToolBtn(c, t, "tb-close", IconClose, "Close Tab (Ctrl+W)", func() {
			u.RequestCloseTab(u.app.ActiveTab)
		})

		u.renderToolbarSep(c, t)

		// Group 2: Edit Actions
		u.renderToolBtn(c, t, "tb-cut", IconCut, "Cut (Ctrl+X)", func() {
			if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
				doc.State.Run(editor.Cut)
			}
		})
		u.renderToolBtn(c, t, "tb-copy", IconCopy, "Copy (Ctrl+C)", func() {
			if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
				doc.State.Run(editor.Copy)
			}
		})
		u.renderToolBtn(c, t, "tb-paste", IconPaste, "Paste (Ctrl+V)", func() {
			if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
				doc.State.Run(editor.Paste)
			}
		})
		u.renderToolBtn(c, t, "tb-undo", IconUndo, "Undo (Ctrl+Z)", func() {
			if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
				doc.State.Run(editor.Undo)
			}
		})
		u.renderToolBtn(c, t, "tb-redo", IconRedo, "Redo (Ctrl+Y)", func() {
			if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
				doc.State.Run(editor.Redo)
			}
		})

		u.renderToolbarSep(c, t)

		// Group 3: Search Actions
		u.renderToolBtn(c, t, "tb-find", IconFind, "Find... (Ctrl+F)", func() {
			u.OpenFind()
		})
		u.renderToolBtn(c, t, "tb-replace", IconReplace, "Replace... (Ctrl+H)", func() {
			u.OpenFindReplace()
		})
		u.renderToolBtn(c, t, "tb-goto", IconGoToLine, "Go to Line... (Ctrl+G)", func() {
			u.GoToLine()
		})

		u.renderToolbarSep(c, t)

		// Group 4: View Actions
		wrapActive := false
		if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
			wrapActive = doc.State.IsSoftWrap()
		}
		wrapBtn := ui.ButtonBase(c.Key("tb-wrap")).Size(22, 22).Radius(3).Center().FocusRing(false)
		wrapBtn.Tooltip("Toggle Word Wrap (Alt+Z)")
		if wrapActive {
			wrapBtn.Background(t.SurfacePressed)
		} else if wrapBtn.Hovered() {
			wrapBtn.Background(t.SurfaceHover)
		}
		wrapBtn.Children(func() {
			color := t.TextMuted
			if wrapActive {
				color = t.Accent
			}
			ui.Icon(c, IconWordWrap).Size(15, 15).TextColor(color)
		})
		if wrapBtn.Clicked() {
			u.ToggleWordWrap()
		}

		u.renderToolBtn(c, t, "tb-zoomin", IconZoomIn, "Zoom In (Ctrl++)", func() {
			u.ZoomIn()
		})
		u.renderToolBtn(c, t, "tb-zoomout", IconZoomOut, "Zoom Out (Ctrl+-)", func() {
			u.ZoomOut()
		})
	})
}

func (u *UI) renderToolBtn(c *ui.Context, t *ui.Theme, key string, icon *ui.SVG, tip string, action func()) {
	btn := ui.ButtonBase(c.Key(key)).Size(22, 22).Radius(3).Center().FocusRing(false)
	btn.Tooltip(tip)
	if btn.Hovered() {
		btn.Background(t.SurfaceHover)
	}
	btn.Children(func() {
		ui.Icon(c, icon).Size(15, 15).TextColor(t.Text)
	})
	if btn.Clicked() {
		action()
	}
}

func (u *UI) renderToolbarSep(c *ui.Context, t *ui.Theme) {
	ui.Box(c).Width(1).Height(14).Background(t.Border).Margin(0, 2)
}
