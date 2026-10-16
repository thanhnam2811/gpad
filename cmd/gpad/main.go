package main

import (
	"log"
	"os"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"gpad/internal/app"
	gpadui "gpad/internal/ui"
)

func main() {
	application := app.NewApp()

	// Ensure WAL draft session is persisted and resources closed on quit
	mygo.App.OnQuit(func() {
		application.Stop()
	})

	// Handle initial file argument if passed: gpad <filename>
	if len(os.Args) > 1 {
		_ = application.OpenFile(os.Args[1])
	}

	gui := gpadui.NewUI(application)

	mygo.App.WhenReady(func() {
		mygo.NewWindow(mygo.WindowOptions{
			Title:   "Gopad++",
			Width:   960,
			Height:  680,
			Content: ui.View(gui.Render),
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
