package main

import (
	"flag"
	"fmt"
	"gpad/internal/buildinfo"
	"io"
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
	files, done, err := parseCommandLine(os.Args[1:], os.Stdout)
	if err != nil {
		log.Print(err)
		os.Exit(2)
	}
	if done {
		return
	}
	sockPath := instance.SocketPath()

	// 1. If another instance is running, forward the files and exit 0 immediately
	if len(files) > 0 {
		if instance.TryForward(sockPath, files) {
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
			if instance.TryForward(sockPath, files) {
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
	if len(files) > 0 {
		for _, arg := range files {
			if err := application.OpenFile(arg); err != nil {
				log.Printf("Open file: %v", err)
			}
		}
	}

	gui := gpadui.NewUI(application)

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

// parseCommandLine handles informational modes before opening sessions or IPC.
func parseCommandLine(args []string, out io.Writer) ([]string, bool, error) {
	flags := flag.NewFlagSet("gpad", flag.ContinueOnError)
	flags.SetOutput(out)
	version := flags.Bool("version", false, "print build version and exit")
	install := flags.Bool("install-desktop", false, "install a per-user desktop launcher and icons, then exit")
	flags.Usage = func() {
		fmt.Fprintln(out, "Usage: gpad [--version | --help | --install-desktop] [--] [files...]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return nil, true, nil
		}
		return nil, true, err
	}
	if *version {
		fmt.Fprintln(out, buildinfo.String())
		return nil, true, nil
	}
	if *install {
		executable, err := os.Executable()
		if err == nil {
			err = gpadui.EnsureDesktopEntry(executable)
		}
		return nil, true, err
	}
	return flags.Args(), false, nil
}
