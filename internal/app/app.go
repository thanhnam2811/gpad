package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"gpad/internal/buffer"
	"gpad/internal/fileio"
	"gpad/internal/search"
	"gpad/internal/session"
)

// App manages global application state, tabs, and coordination across subsystems.
type App struct {
	mu             sync.Mutex
	Tabs           []*buffer.Document
	ActiveTab      int
	SearchOpen     bool
	SearchQuery    string
	SearchResults  search.Result
	LastOpLatency  time.Duration
	StatusMessage  string
	SessionMgr     *session.Manager
	SearchEngine   search.Engine
	tabCounter     int
}

// NewApp initializes a new application state with one default Untitled tab.
func NewApp() *App {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	sessionDir := filepath.Join(homeDir, ".config", "gpad", "session")
	sessMgr, _ := session.NewManager(sessionDir)

	a := &App{
		Tabs:         make([]*buffer.Document, 0, 8),
		ActiveTab:    0,
		SessionMgr:   sessMgr,
		SearchEngine: search.NewAdaptiveSearchDispatcher(),
		tabCounter:   0,
	}

	// Try restoring session; if none, open default Untitled-1
	if err := a.RestoreSession(); err != nil || len(a.Tabs) == 0 {
		a.NewTab("", "")
	}

	return a
}

// NewTab creates a new tab. If title is empty, it generates "Untitled N".
func (a *App) NewTab(title, content string) *buffer.Document {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.tabCounter++
	if title == "" {
		title = fmt.Sprintf("Untitled %d", a.tabCounter)
	}
	id := fmt.Sprintf("doc-%d-%d", time.Now().UnixNano(), a.tabCounter)
	doc := buffer.NewDocument(id, title, content, "")

	a.Tabs = append(a.Tabs, doc)
	a.ActiveTab = len(a.Tabs) - 1
	return doc
}

// CloseTab closes tab at index. Keeps at least one tab open.
func (a *App) CloseTab(index int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if index < 0 || index >= len(a.Tabs) {
		return
	}

	a.Tabs = append(a.Tabs[:index], a.Tabs[index+1:]...)
	if len(a.Tabs) == 0 {
		a.tabCounter++
		doc := buffer.NewDocument(fmt.Sprintf("doc-%d", a.tabCounter), fmt.Sprintf("Untitled %d", a.tabCounter), "", "")
		a.Tabs = append(a.Tabs, doc)
		a.ActiveTab = 0
		return
	}

	if a.ActiveTab >= len(a.Tabs) {
		a.ActiveTab = len(a.Tabs) - 1
	}
}

// SwitchTab changes active tab index.
func (a *App) SwitchTab(index int) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if index >= 0 && index < len(a.Tabs) {
		a.ActiveTab = index
	}
}

// NextTab cycles to next tab (Ctrl+Tab).
func (a *App) NextTab() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab + 1) % len(a.Tabs)
	}
}

// PrevTab cycles to previous tab.
func (a *App) PrevTab() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab - 1 + len(a.Tabs)) % len(a.Tabs)
	}
}

// ActiveDocument returns the currently selected document.
func (a *App) ActiveDocument() *buffer.Document {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.Tabs) == 0 {
		return nil
	}
	if a.ActiveTab < 0 || a.ActiveTab >= len(a.Tabs) {
		a.ActiveTab = 0
	}
	return a.Tabs[a.ActiveTab]
}

// OpenFile loads a file into a new or active clean tab.
func (a *App) OpenFile(path string) error {
	start := time.Now()
	payload, err := fileio.LoadFile(path)
	if err != nil {
		return err
	}
	defer payload.Close()

	title := filepath.Base(path)
	doc := buffer.NewDocument(path, title, string(payload.Data), path)

	a.mu.Lock()
	defer a.mu.Unlock()

	// If current tab is clean Untitled, replace it
	if len(a.Tabs) == 1 && a.Tabs[0].Path == "" && !a.Tabs[0].IsDirty() && a.Tabs[0].Len() == 0 {
		a.Tabs[0] = doc
		a.ActiveTab = 0
	} else {
		a.Tabs = append(a.Tabs, doc)
		a.ActiveTab = len(a.Tabs) - 1
	}

	a.LastOpLatency = time.Since(start)
	return nil
}

// SaveActiveFile saves the currently active document to disk.
func (a *App) SaveActiveFile() error {
	doc := a.ActiveDocument()
	if doc == nil || doc.Path == "" {
		return fmt.Errorf("no target file path specified")
	}

	start := time.Now()
	content := doc.Content()
	if err := fileio.SaveFile(doc.Path, []byte(content)); err != nil {
		return err
	}
	doc.MarkSaved()

	a.mu.Lock()
	a.LastOpLatency = time.Since(start)
	a.mu.Unlock()
	return nil
}

// SaveSession writes all tabs and drafts to session storage.
func (a *App) SaveSession() error {
	if a.SessionMgr == nil {
		return nil
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	state := session.SessionState{
		ActiveIndex: a.ActiveTab,
		Tabs:        make([]session.TabSession, len(a.Tabs)),
	}

	for i, doc := range a.Tabs {
		line, col := doc.CaretPosition()
		state.Tabs[i] = session.TabSession{
			ID:         doc.ID,
			Title:      doc.Title,
			Path:       doc.Path,
			Content:    doc.Content(),
			CursorLine: line,
			CursorCol:  col,
			IsDirty:    doc.IsDirty(),
		}
	}

	return a.SessionMgr.SaveSnapshot(state)
}

// RestoreSession loads prior tabs and state from session storage.
func (a *App) RestoreSession() error {
	if a.SessionMgr == nil {
		return nil
	}

	state, err := a.SessionMgr.LoadSnapshot()
	if err != nil || state == nil || len(state.Tabs) == 0 {
		return err
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	a.Tabs = make([]*buffer.Document, 0, len(state.Tabs))
	for _, t := range state.Tabs {
		doc := buffer.NewDocument(t.ID, t.Title, t.Content, t.Path)
		if !t.IsDirty {
			doc.MarkSaved()
		}
		a.Tabs = append(a.Tabs, doc)
	}

	if state.ActiveIndex >= 0 && state.ActiveIndex < len(a.Tabs) {
		a.ActiveTab = state.ActiveIndex
	} else {
		a.ActiveTab = 0
	}
	return nil
}

// PerformSearch executes search on current document buffer.
func (a *App) PerformSearch(query string) {
	doc := a.ActiveDocument()
	if doc == nil || len(query) == 0 {
		a.SearchResults = search.Result{}
		return
	}

	content := doc.Content()
	res := a.SearchEngine.FindAll([]byte(content), []byte(query))
	a.SearchResults = res
	a.LastOpLatency = res.Duration
}
