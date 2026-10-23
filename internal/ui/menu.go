package ui

import (
	"fmt"
	"gpad/internal/buildinfo"
	"runtime"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui/editor"
	"gpad/internal/app"
)

// BuildAppMenu constructs the professional native top menu bar hierarchy for Linux GNOME 46.
func BuildAppMenu(application *app.App, gui *UI, win *mygo.Window) *mygo.Menu {
	return mygo.NewMenu([]*mygo.MenuItem{
		// 1. File Menu
		{
			Label: "File",
			Submenu: []*mygo.MenuItem{
				{
					Label:       "New Tab",
					Accelerator: "CmdOrCtrl+N",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						application.NewTab("", "")
					},
				},
				{
					Label:       "Open File...",
					Accelerator: "CmdOrCtrl+O",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
							Title:  "Open File - Gopad++",
							Parent: w,
						})
						if err == nil && len(paths) > 0 {
							_ = application.OpenFile(paths[0])
						}
					},
				},
				{
					Label:       "Save",
					Accelerator: "CmdOrCtrl+S",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						doc := application.ActiveDocument()
						if doc != nil {
							if doc.Path == "" {
								// Untitled draft: trigger Save As dialog
								path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
									Title:       "Save As - Gopad++",
									DefaultPath: doc.Title + ".txt",
									Parent:      w,
								})
								if err == nil && path != "" {
									_ = application.SaveActiveFileAs(path)
								}
							} else {
								_ = application.SaveActiveFile()
							}
						}
					},
				},
				{
					Label:       "Save As...",
					Accelerator: "CmdOrCtrl+Shift+S",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						doc := application.ActiveDocument()
						if doc != nil {
							defaultPath := doc.Path
							if defaultPath == "" {
								defaultPath = doc.Title + ".txt"
							}
							path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{
								Title:       "Save As - Gopad++",
								DefaultPath: defaultPath,
								Parent:      w,
							})
							if err == nil && path != "" {
								_ = application.SaveActiveFileAs(path)
							}
						}
					},
				},
				mygo.Separator(),
				{
					Label:       "Close Tab",
					Accelerator: "CmdOrCtrl+W",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						application.CloseTab(application.ActiveTab)
					},
				},
				{
					Label: "Close Other Tabs",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						application.CloseOtherTabs(application.ActiveTab)
					},
				},
				{
					Label: "Close Tabs to the Right",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						application.CloseTabsToRight(application.ActiveTab)
					},
				},
				{
					Label: "Close All Tabs",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						application.CloseAllTabs()
					},
				},
				mygo.Separator(),
				{
					Label:       "Exit",
					Accelerator: "CmdOrCtrl+Q",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						mygo.App.Quit()
					},
				},
			},
		},

		// 2. Edit Menu
		{
			Label: "Edit",
			Submenu: []*mygo.MenuItem{
				{
					Label:       "Undo",
					Accelerator: "CmdOrCtrl+Z",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.Undo)
						}
					},
				},
				{
					Label:       "Redo",
					Accelerator: "CmdOrCtrl+Y",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.Redo)
						}
					},
				},
				mygo.Separator(),
				{
					Label:       "Cut",
					Accelerator: "CmdOrCtrl+X",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.Cut)
						}
					},
				},
				{
					Label:       "Copy",
					Accelerator: "CmdOrCtrl+C",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.Copy)
						}
					},
				},
				{
					Label:       "Paste",
					Accelerator: "CmdOrCtrl+V",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.Paste)
						}
					},
				},
				{
					Label:       "Select All",
					Accelerator: "CmdOrCtrl+A",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.SelectAll)
						}
					},
				},
				mygo.Separator(),
				{
					Label:       "Duplicate Line",
					Accelerator: "CmdOrCtrl+D",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.DuplicateLinesDown)
						}
					},
				},
				{
					Label:       "Delete Line",
					Accelerator: "CmdOrCtrl+Shift+K",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.DeleteLine)
						}
					},
				},
				{
					Label:       "Toggle Line Comment",
					Accelerator: "CmdOrCtrl+/",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							doc.State.Run(editor.ToggleComment)
						}
					},
				},
			},
		},

		// 3. Search Menu
		{
			Label: "Search",
			Submenu: []*mygo.MenuItem{
				{
					Label:       "Find...",
					Accelerator: "CmdOrCtrl+F",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.OpenFind()
					},
				},
				{
					Label:       "Replace...",
					Accelerator: "CmdOrCtrl+H",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.OpenFindReplace()
					},
				},
				{
					Label:       "Find Next",
					Accelerator: "F3",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.FindNext()
					},
				},
				{
					Label:       "Find Previous",
					Accelerator: "Shift+F3",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.FindPrevious()
					},
				},
				{
					Label:       "Select All Matches",
					Accelerator: "Alt+Enter",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.SelectAllMatches()
					},
				},
				mygo.Separator(),
				{
					Label:       "Go to Line...",
					Accelerator: "CmdOrCtrl+G",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.GoToLine()
					},
				},
			},
		},

		// 4. View Menu
		{
			Label: "View",
			Submenu: []*mygo.MenuItem{
				{
					Label:       "Toggle Soft Wrap",
					Accelerator: "Alt+Z",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.ToggleWordWrap()
					},
				},
				{
					Label: "Toggle Line Numbers",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil && !doc.IsLoading() && doc.State != nil {
							if doc.State.LineNumbers == editor.LineNumbersOff {
								doc.State.LineNumbers = editor.LineNumbersOn
							} else {
								doc.State.LineNumbers = editor.LineNumbersOff
							}
						}
					},
				},
				mygo.Separator(),
				{
					Label:       "Zoom In",
					Accelerator: "CmdOrCtrl+=",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.ZoomIn()
					},
				},
				{
					Label:       "Zoom Out",
					Accelerator: "CmdOrCtrl+-",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.ZoomOut()
					},
				},
				{
					Label:       "Reset Zoom",
					Accelerator: "CmdOrCtrl+0",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						gui.ZoomReset()
					},
				},
				mygo.Separator(),
				{
					Label:       "Toggle Full Screen",
					Accelerator: "F11",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if w != nil {
							w.SetFullScreen(!w.IsFullScreen())
						}
					},
				},
			},
		},

		// 5. Document Menu
		{
			Label: "Document",
			Submenu: []*mygo.MenuItem{
				{
					Label: "Convert to LF (Unix)",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil {
							doc.CRLF = false
						}
					},
				},
				{
					Label: "Convert to CRLF (Windows)",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						if doc := application.ActiveDocument(); doc != nil {
							doc.CRLF = true
						}
					},
				},
				mygo.Separator(),
				{
					Label:    "Encoding: UTF-8",
					Type:     mygo.MenuItemCheckbox,
					Checked:  true,
					Disabled: true,
				},
			},
		},

		// 6. Help Menu
		{
			Label: "Help",
			Submenu: []*mygo.MenuItem{
				{
					Label: "About Gopad++",
					Click: func(item *mygo.MenuItem, w *mygo.Window) {
						_, _ = mygo.Dialog.Message(mygo.MessageOptions{
							Title:   "About Gopad++",
							Message: buildinfo.String(),
							Detail:  fmt.Sprintf("Engine: Pure Go & mygo/ui (Purego GPU OpenGL)\nHost: %s/%s\nZero WebView • Zero Electron • 120Hz Target", runtime.GOOS, runtime.GOARCH),
							Parent:  w,
						})
					},
				},
			},
		},
	})
}
