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

	// 2. Start single-instance listener to accept files from subsequent instances into new tabs
	listener, err := instance.StartListener(sockPath, func(path string) {
		_ = application.OpenFile(path)
	}, func() {
		// Window focus event
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
		w := mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gopad++",
			Width:   960,
			Height:  680,
			Content: ui.View(gui.Render),
		})

		// Custom application icon on window
		_ = w.SetIcon(gpadui.GetAppIconPNG())
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
