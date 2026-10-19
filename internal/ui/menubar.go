package ui

import (
	"fmt"

	"gpad/internal/keybind"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

type menuDef struct {
	id      string
	label   string
	width   float32
	content func()
}

// renderMenuBar renders the ultra-compact 22px in-window top menu bar.
func (u *UI) renderMenuBar(c *ui.Context, t *ui.Theme) {
	menus := []menuDef{
		{
			id:    "file",
			label: "File",
			width: 220,
			content: func() {
				renderMenuItem(c, t, "new-tab", "New Tab", u.keyRegistry.ShortcutString(keybind.ActionFileNew), false, func() {
					u.app.NewTab("", "")
					u.activeMenu = ""
				})
				renderMenuItem(c, t, "open-file", "Open File...", u.keyRegistry.ShortcutString(keybind.ActionFileOpen), false, func() {
					u.activeMenu = ""
					paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
						Title: "Open File - Gopad++",
					})
					if err == nil && len(paths) > 0 {
						_ = u.app.OpenFile(paths[0])
					}
				})
				renderMenuItem(c, t, "save", "Save", u.keyRegistry.ShortcutString(keybind.ActionFileSave), false, func() {
					u.activeMenu = ""
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
				renderMenuItem(c, t, "save-as", "Save As...", u.keyRegistry.ShortcutString(keybind.ActionFileSaveAs), false, func() {
					u.activeMenu = ""
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
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "close-tab", "Close Tab", u.keyRegistry.ShortcutString(keybind.ActionFileCloseTab), false, func() {
					u.activeMenu = ""
					u.RequestCloseTab(u.app.ActiveTab)
				})
				renderMenuItem(c, t, "close-other-tabs", "Close Other Tabs", "", false, func() {
					u.app.CloseOtherTabs(u.app.ActiveTab)
					u.activeMenu = ""
				})
				renderMenuItem(c, t, "close-tabs-right", "Close Tabs to the Right", "", false, func() {
					u.app.CloseTabsToRight(u.app.ActiveTab)
					u.activeMenu = ""
				})
				renderMenuItem(c, t, "close-all-tabs", "Close All Tabs", "", false, func() {
					u.app.CloseAllTabs()
					u.activeMenu = ""
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "exit", "Exit", u.keyRegistry.ShortcutString(keybind.ActionFileExit), false, func() {
					u.activeMenu = ""
					mygo.App.Quit()
				})
			},
		},
		{
			id:    "edit",
			label: "Edit",
			width: 210,
			content: func() {
				renderMenuItem(c, t, "undo", "Undo", u.keyRegistry.ShortcutString(keybind.ActionEditUndo), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.Undo)
					}
				})
				renderMenuItem(c, t, "redo", "Redo", u.keyRegistry.ShortcutString(keybind.ActionEditRedo), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.Redo)
					}
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "cut", "Cut", u.keyRegistry.ShortcutString(keybind.ActionEditCut), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.Cut)
					}
				})
				renderMenuItem(c, t, "copy", "Copy", u.keyRegistry.ShortcutString(keybind.ActionEditCopy), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.Copy)
					}
				})
				renderMenuItem(c, t, "paste", "Paste", u.keyRegistry.ShortcutString(keybind.ActionEditPaste), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.Paste)
					}
				})
				renderMenuItem(c, t, "select-all", "Select All", u.keyRegistry.ShortcutString(keybind.ActionEditSelectAll), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.SelectAll)
					}
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "duplicate-line", "Duplicate Line", u.keyRegistry.ShortcutString(keybind.ActionEditDupLine), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.DuplicateLinesDown)
					}
				})
				renderMenuItem(c, t, "delete-line", "Delete Line", u.keyRegistry.ShortcutString(keybind.ActionEditDelLine), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.DeleteLine)
					}
				})
				renderMenuItem(c, t, "toggle-comment", "Toggle Line Comment", u.keyRegistry.ShortcutString(keybind.ActionEditToggleComment), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.Run(editor.ToggleComment)
					}
				})
			},
		},
		{
			id:    "search",
			label: "Search",
			width: 220,
			content: func() {
				renderMenuItem(c, t, "find", "Find...", u.keyRegistry.ShortcutString(keybind.ActionSearchFind), false, func() {
					u.activeMenu = ""
					u.OpenFind()
				})
				renderMenuItem(c, t, "replace", "Replace...", u.keyRegistry.ShortcutString(keybind.ActionSearchReplace), false, func() {
					u.activeMenu = ""
					u.OpenFindReplace()
				})
				renderMenuItem(c, t, "find-next", "Find Next", u.keyRegistry.ShortcutString(keybind.ActionSearchFindNext), false, func() {
					u.activeMenu = ""
					u.FindNext()
				})
				renderMenuItem(c, t, "find-prev", "Find Previous", u.keyRegistry.ShortcutString(keybind.ActionSearchFindPrev), false, func() {
					u.activeMenu = ""
					u.FindPrevious()
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "goto-line", "Go to Line...", u.keyRegistry.ShortcutString(keybind.ActionSearchGoToLine), false, func() {
					u.activeMenu = ""
					u.GoToLine()
				})
				renderMenuItem(c, t, "select-matches", "Select All Matches", u.keyRegistry.ShortcutString(keybind.ActionSearchSelectMatches), false, func() {
					u.activeMenu = ""
					u.SelectAllMatches()
				})
			},
		},
		{
			id:    "view",
			label: "View",
			width: 200,
			content: func() {
				renderMenuItem(c, t, "word-wrap", "Toggle Word Wrap", u.keyRegistry.ShortcutString(keybind.ActionViewToggleWrap), false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						doc.State.SoftWrap = !doc.State.SoftWrap
					}
				})
				renderMenuItem(c, t, "line-numbers", "Toggle Line Numbers", "", false, func() {
					u.activeMenu = ""
					if doc := u.app.ActiveDocument(); doc != nil && doc.State != nil {
						if doc.State.LineNumbers == editor.LineNumbersOff {
							doc.State.LineNumbers = editor.LineNumbersOn
						} else {
							doc.State.LineNumbers = editor.LineNumbersOff
						}
					}
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "fullscreen", "Toggle Full Screen", u.keyRegistry.ShortcutString(keybind.ActionViewFullScreen), false, func() {
					u.activeMenu = ""
				})
			},
		},
		{
			id:    "doc",
			label: "Document",
			width: 220,
			content: func() {
				doc := u.app.ActiveDocument()
				lineEnding := "Line Ending: LF"
				encoding := "Encoding: UTF-8"
				if doc != nil {
					lineEnding = fmt.Sprintf("Line Ending: %s", doc.LineEndingDesc())
					encoding = fmt.Sprintf("Encoding: %s", doc.Encoding)
				}
				renderMenuItem(c, t, "conv-lf", "Convert to LF (Unix)", "", false, func() {
					u.activeMenu = ""
					if doc != nil {
						doc.CRLF = false
					}
				})
				renderMenuItem(c, t, "conv-crlf", "Convert to CRLF (Windows)", "", false, func() {
					u.activeMenu = ""
					if doc != nil {
						doc.CRLF = true
					}
				})
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "info-enc", encoding, "", true, nil)
				renderMenuItem(c, t, "info-le", lineEnding, "", true, nil)
				renderMenuSeparator(c, t)
				renderMenuItem(c, t, "copy-doc-path", "Copy Document Path", "", false, func() {
					u.activeMenu = ""
					u.app.CopyFilePath(u.app.ActiveTab)
				})
			},
		},
		{
			id:    "help",
			label: "Help",
			width: 190,
			content: func() {
				renderMenuItem(c, t, "about", "About Gopad++", "", false, func() {
					u.activeMenu = ""
					_, _ = mygo.Dialog.Message(mygo.MessageOptions{
						Title:   "About Gopad++",
						Message: "Gopad++ (gpad) v0.1.0-alpha",
						Detail:  "A high-density native text editor for Linux (GNOME 46) written in Go.\nOpenGL GPU Rendering • Strictly Zero Webview • Notepad++ Ergonomics",
						Type:    mygo.MessageInfo,
					})
				})
			},
		},
	}

	var activeBtn ui.Element
	var activeContent func()
	var activeWidth float32

	ui.Row(c.Key("in-window-menubar")).FillWidth().Height(22).Background(t.Surface).Padding(0, 4).Gap(1).AlignItems(ui.Center).Children(func() {
		for _, m := range menus {
			menuItem := m
			btn := ui.ButtonBase(c.Key("menu-btn-"+menuItem.id)).Height(20).Padding(1, 7).Radius(3).FocusRing(false).AlignItems(ui.Center)

			isCurrent := u.activeMenu == menuItem.id

			if btn.Clicked() {
				if isCurrent {
					u.activeMenu = ""
				} else {
					u.activeMenu = menuItem.id
				}
			} else if u.activeMenu != "" && !isCurrent && btn.Hovered() {
				u.activeMenu = menuItem.id
			}

			isNowActive := u.activeMenu == menuItem.id
			if isNowActive || btn.Hovered() {
				btn.Background(t.SurfaceHover)
			}

			btn.Children(func() {
				ui.Text(c, menuItem.label).FontSize(11).TextColor(t.Text)
			})

			if isNowActive {
				activeBtn = btn
				activeContent = menuItem.content
				activeWidth = menuItem.width
			}
		}
	})

	// Render EXACTLY ONE PopoverBase attached to the active button (0 overlap possible)
	if u.activeMenu != "" && activeContent != nil {
		isOpen := true
		ui.PopoverBase(c, activeBtn, &isOpen, func(panel ui.Element) {
			panel.Background(t.Surface).Border(1, t.Border).Radius(4).Padding(3, 3).Width(activeWidth)
			ui.Column(c).FillWidth().Gap(1).Children(func() {
				activeContent()
			})
		})

		if !isOpen {
			u.activeMenu = ""
		}
	}
}
