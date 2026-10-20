package main

import (
	"log"
	"os"
	"sync"
	"time"

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

	// Acquire instance ownership before opening the persistent session. Requests
	// received during startup are applied only by the UI thread.
	var requests requestQueue
	listener, err := instance.StartListener(sockPath, func(path, token string) {
		requests.add(ipcRequest{path: path, token: token})
	}, func(token string) {
		requests.add(ipcRequest{token: token})
	})
	if err != nil {
		// The winning process may still be binding its socket.
		for attempt := 0; attempt < 10; attempt++ {
			if instance.TryForward(sockPath, os.Args[1:]) {
				return
			}
			time.Sleep(20 * time.Millisecond)
		}
		log.Printf("Cannot acquire application instance: %v", err)
		return
	}
	defer listener.Close()
	application := app.NewApp()
	flushRequests := func() {
		for _, request := range requests.take() {
			if request.path != "" {
				if err := application.OpenFile(request.path); err != nil {
					log.Printf("Open file during shutdown: %v", err)
				}
			}
		}
	}
	var shutdownOnce sync.Once
	shutdown := func() {
		shutdownOnce.Do(func() {
			_ = listener.StopAccepting()
			flushRequests()
			application.Stop()
			_ = listener.Close()
		})
	}
	defer shutdown()

	var mainWin *mygo.Window
	raiseWindow := func(token string) {
		if mainWin != nil {
			mainWin.Restore()
			mainWin.SetAlwaysOnTop(true)
			mainWin.FocusActivated(token)
			mainWin.SetAlwaysOnTop(false)
		}
	}

	// Ensure WAL draft session is persisted and resources closed on quit
	mygo.App.OnQuit(shutdown)

	// Handle initial file arguments if passed: gpad <file1> <file2>...
	if len(os.Args) > 1 {
		for _, arg := range os.Args[1:] {
			if err := application.OpenFile(arg); err != nil {
				log.Printf("Open file: %v", err)
			}
		}
	}

	gui := gpadui.NewUI(application)

	// Ensure GNOME 46 desktop entry and icon exist
	if binPath, err := os.Executable(); err == nil {
		_ = gpadui.EnsureDesktopEntry(binPath)
	}

	mygo.App.WhenReady(func() {
		mainWin = mygo.NewWindow(mygo.WindowOptions{
			Title:  "Gopad++",
			Width:  960,
			Height: 680,
			Content: ui.View(func(c *ui.Context) {
				for _, request := range requests.take() {
					if request.path != "" {
						if err := application.OpenFile(request.path); err != nil {
							log.Printf("Open file: %v", err)
						}
					}
					raiseWindow(request.token)
				}
				gui.Render(c)
			}),
		})

		requests.setWake(mainWin.Invalidate)

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
		log.Print(err)
	}
}

// requestQueue decouples socket callbacks from UI lifetime and shutdown. Taking
// the queue transfers ownership without allocating on idle render frames.
type ipcRequest struct{ path, token string }
type requestQueue struct {
	mu      sync.Mutex
	pending []ipcRequest
	wake    func()
}

func (q *requestQueue) add(r ipcRequest) {
	q.mu.Lock()
	q.pending = append(q.pending, r)
	wake := q.wake
	q.mu.Unlock()
	if wake != nil {
		wake()
	}
}
func (q *requestQueue) take() []ipcRequest {
	q.mu.Lock()
	pending := q.pending
	q.pending = nil
	q.mu.Unlock()
	return pending
}

// setWake publishes the thread-safe native invalidator after window creation.
func (q *requestQueue) setWake(wake func()) {
	q.mu.Lock()
	q.wake = wake
	pending := len(q.pending) > 0
	q.mu.Unlock()
	if pending {
		wake()
	}
}
