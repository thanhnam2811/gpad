package main

import (
	"log"
	"os"

	"gpad/internal/app"
	"gpad/internal/instance"
	gpadui "gpad/internal/ui"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

func main() {
	sockPath := instance.SocketPath()

	// 1. If another instance is running, forward the files and exit 0 immediately
	if len(os.Args) > 1 {
		if instance.TryForward(sockPath, os.Args[1:]) {
			os.Exit(0)
		}
	} else {
		// If launched without arguments but an instance is running, focus it and exit
		if instance.TryForward(sockPath, nil) {
			os.Exit(0)
		}
	}

	application := app.NewApp()

	var mainWin *mygo.Window

	// raiseWindow brings the window to the front and focuses it. The token is
	// the activation token of the process that asked for it: without one,
	// Wayland only marks the window as demanding attention instead of
	// raising it over the window the user launched us from.
	raiseWindow := func(token string) {
		if mainWin != nil {
			mainWin.Restore()
			mainWin.SetAlwaysOnTop(true)
			mainWin.FocusActivated(token)
			mainWin.SetAlwaysOnTop(false)
		}
	}

	// 2. Start single-instance listener to accept files from subsequent instances into new tabs
	listener, err := instance.StartListener(sockPath, func(path, token string) {
		_ = application.OpenFile(path)
		raiseWindow(token)
	}, func(token string) {
		raiseWindow(token)
	})
	if err == nil {
		defer listener.Close()
	}

	// Ensure WAL draft session is persisted and resources closed on quit
	mygo.App.OnQuit(func() {
		if listener != nil {
			_ = listener.Close()
		}
		application.Stop()
	})

	// Handle initial file arguments if passed: gpad <file1> <file2>...
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			_ = application.OpenFile(arg)
		}
	}

	gui := gpadui.NewUI(application)

	// Ensure GNOME 46 desktop entry and icon exist
	if binPath, err := os.Executable(); err == nil {
		_ = gpadui.EnsureDesktopEntry(binPath)
	}

	mygo.App.WhenReady(func() {
		mainWin = mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gopad++",
			Width:   960,
			Height:  680,
			Content: ui.View(gui.Render),
		})

		// Custom application icon on window
		_ = mainWin.SetIcon(gpadui.GetAppIconPNG())

		// Take the focus with the token of our own launch, when the
		// launcher gave us one: the window was not started by the user
		// pressing in it, and Wayland asks for the token to focus it.
		if token := instance.ActivationToken(); token != "" {
			mainWin.FocusActivated(token)
		}

		// Intercept context menus to render 100% in-window using OpenGL GPU
		mainWin.SetPopupMenuHandler(gui.ShowEditorContextMenu)
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
