package instance

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// SocketPath returns the standard Unix domain socket path for gpad.
func SocketPath() string {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	if runtimeDir != "" {
		return filepath.Join(runtimeDir, "gpad.sock")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), fmt.Sprintf("gpad-%d.sock", os.Getuid()))
	}
	return filepath.Join(home, ".config", "gpad", "gpad.sock")
}

// ActivationToken returns the activation token of the launch of this process,
// when its launcher gave it one. A window needs it to take the focus on
// Wayland; GLib desktop applications (a file manager, a terminal) provide it
// as XDG_ACTIVATION_TOKEN, and as DESKTOP_STARTUP_ID to the applications they
// launch.
func ActivationToken() string {
	if token := os.Getenv("XDG_ACTIVATION_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("DESKTOP_STARTUP_ID")
}

// TryForward attempts to connect to an existing running gpad instance and forward file paths.
// Returns true if successfully delivered to an active instance, false otherwise.
func TryForward(socketPath string, filePaths []string) bool {
	conn, err := net.DialTimeout("unix", socketPath, 200*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()

	// Set deadline for IPC handshake
	_ = conn.SetDeadline(time.Now().Add(1 * time.Second))

	// The token first: it is what lets the running instance raise its window
	// over the one the user launched us from.
	if token := ActivationToken(); token != "" {
		if _, err := fmt.Fprintf(conn, "TOKEN %s\n", token); err != nil {
			return false
		}
	}

	for _, p := range filePaths {
		abs, err := filepath.Abs(p)
		if err == nil {
			p = abs
		}
		_, err = fmt.Fprintf(conn, "OPEN %s\n", strings.TrimSpace(p))
		if err != nil {
			return false
		}
	}
	_, err = fmt.Fprintln(conn, "FOCUS")
	if err != nil {
		return false
	}

	// Read acknowledgment
	reader := bufio.NewReader(conn)
	ack, err := reader.ReadString('\n')
	if err != nil || strings.TrimSpace(ack) != "OK" {
		return false
	}
	return true
}

// Listener represents the Single-Instance Unix domain socket listener.
type Listener struct {
	path       string
	ln         net.Listener
	onOpenFile func(path, token string)
	onFocus    func(token string)
	quit       chan struct{}
	wg         sync.WaitGroup
}

// StartListener starts listening for file open requests from subsequent gpad processes.
// Both callbacks receive the activation token of the requesting process, empty
// when it has none, for raising the window with FocusActivated.
func StartListener(socketPath string, onOpenFile func(path, token string), onFocus func(token string)) (*Listener, error) {
	// Ensure directory exists
	dir := filepath.Dir(socketPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	// Remove stale socket file if it exists
	_ = os.Remove(socketPath)

	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		return nil, err
	}

	l := &Listener{
		path:       socketPath,
		ln:         ln,
		onOpenFile: onOpenFile,
		onFocus:    onFocus,
		quit:       make(chan struct{}),
	}

	l.wg.Add(1)
	go l.serve()

	return l, nil
}

func (l *Listener) serve() {
	defer l.wg.Done()
	for {
		conn, err := l.ln.Accept()
		if err != nil {
			select {
			case <-l.quit:
				return
			default:
				time.Sleep(5 * time.Millisecond)
				continue
			}
		}

		go l.handleConn(conn)
	}
}

func (l *Listener) handleConn(conn net.Conn) {
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(2 * time.Second))

	reader := bufio.NewReader(conn)
	var token string
	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(line)
		if line != "" {
			switch {
			case strings.HasPrefix(line, "TOKEN "):
				token = strings.TrimSpace(strings.TrimPrefix(line, "TOKEN "))
			case strings.HasPrefix(line, "OPEN "):
				targetFile := strings.TrimSpace(strings.TrimPrefix(line, "OPEN "))
				if targetFile != "" && l.onOpenFile != nil {
					l.onOpenFile(targetFile, token)
				}
			case line == "FOCUS":
				if l.onFocus != nil {
					l.onFocus(token)
				}
				// Acknowledge receipt
				_, _ = fmt.Fprintln(conn, "OK")
				return
			}
		}

		if err != nil {
			break
		}
	}
}

// Close gracefully stops the listener and removes the socket file.
func (l *Listener) Close() error {
	close(l.quit)
	err := l.ln.Close()
	l.wg.Wait()
	_ = os.Remove(l.path)
	return err
}
