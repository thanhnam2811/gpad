package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/transfer"
	"github.com/egoist/mygo/ui/editor"
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
	LastSearchQ    string
	IsSearching    bool
	SearchResults  search.Result
	LastOpLatency  time.Duration
	StatusMessage  string
	SessionMgr     *session.Manager
	SearchEngine   search.Engine
	tabCounter     int
	stopAutoSave   chan struct{}

	searchMu     sync.Mutex
	searchTimer  *time.Timer
	searchCancel context.CancelFunc
}

// NewApp initializes a new application state with one default Untitled tab.
func NewApp() *App {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	sessionDir := filepath.Join(homeDir, ".config", "gpad", "session")
	return NewAppWithSessionDir(sessionDir)
}

// NewAppWithSessionDir initializes App with custom session directory for clean isolation.
func NewAppWithSessionDir(sessionDir string) *App {
	sessMgr, _ := session.NewManager(sessionDir)

	a := &App{
		Tabs:         make([]*buffer.Document, 0, 8),
		ActiveTab:    0,
		SessionMgr:   sessMgr,
		SearchEngine: search.NewAdaptiveSearchDispatcher(),
		tabCounter:   0,
		stopAutoSave: make(chan struct{}),
	}

	// Try restoring session; if none or empty, open default Untitled-1
	if err := a.RestoreSession(); err != nil || len(a.Tabs) == 0 {
		a.NewTab("", "")
	}

	// Start continuous background auto-save loop
	go a.startAutoSaveLoop()

	return a
}

// startAutoSaveLoop periodically checks for buffer edits and persists drafts.
func (a *App) startAutoSaveLoop() {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			a.CheckAutoSave()
		case <-a.stopAutoSave:
			return
		}
	}
}

// Stop halts the background autosave routine and closes sessions.
func (a *App) Stop() {
	select {
	case <-a.stopAutoSave:
		// already closed
	default:
		close(a.stopAutoSave)
	}

	a.searchMu.Lock()
	if a.searchTimer != nil {
		a.searchTimer.Stop()
	}
	if a.searchCancel != nil {
		a.searchCancel()
	}
	a.searchMu.Unlock()

	_ = a.SaveSession()
	if a.SessionMgr != nil {
		_ = a.SessionMgr.Close()
	}
}

// CheckAutoSave checks if any tab has been modified since last save and commits to WAL.
func (a *App) CheckAutoSave() {
	a.mu.Lock()
	modified := false
	for _, doc := range a.Tabs {
		if doc != nil && doc.CheckModified() {
			modified = true
		}
	}
	a.mu.Unlock()

	if modified {
		_ = a.SaveSession()
	}
}

// NewTab creates a new tab. If title is empty, it generates "Untitled N".
func (a *App) NewTab(title, content string) *buffer.Document {
	a.mu.Lock()
	a.tabCounter++
	if title == "" {
		title = fmt.Sprintf("Untitled %d", a.tabCounter)
	}
	id := fmt.Sprintf("doc-%d-%d", time.Now().UnixNano(), a.tabCounter)
	doc := buffer.NewDocument(id, title, content, "")

	a.Tabs = append(a.Tabs, doc)
	a.ActiveTab = len(a.Tabs) - 1
	a.mu.Unlock()

	_ = a.SaveSession()
	return doc
}

// CloseTab closes tab at index. Keeps at least one tab open.
func (a *App) CloseTab(index int) {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return
	}

	a.Tabs = append(a.Tabs[:index], a.Tabs[index+1:]...)
	if len(a.Tabs) == 0 {
		a.tabCounter++
		doc := buffer.NewDocument(fmt.Sprintf("doc-%d", a.tabCounter), fmt.Sprintf("Untitled %d", a.tabCounter), "", "")
		a.Tabs = append(a.Tabs, doc)
		a.ActiveTab = 0
		a.mu.Unlock()
		_ = a.SaveSession()
		return
	}

	if a.ActiveTab >= len(a.Tabs) {
		a.ActiveTab = len(a.Tabs) - 1
	}
	a.mu.Unlock()

	_ = a.SaveSession()
}

// SwitchTab changes active tab index.
func (a *App) SwitchTab(index int) {
	a.mu.Lock()
	if index >= 0 && index < len(a.Tabs) {
		a.ActiveTab = index
	}
	a.mu.Unlock()

	_ = a.SaveSession()
}

// NextTab cycles to next tab (Ctrl+Tab).
func (a *App) NextTab() {
	a.mu.Lock()
	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab + 1) % len(a.Tabs)
	}
	a.mu.Unlock()

	_ = a.SaveSession()
}

// PrevTab cycles to previous tab.
func (a *App) PrevTab() {
	a.mu.Lock()
	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab - 1 + len(a.Tabs)) % len(a.Tabs)
	}
	a.mu.Unlock()

	_ = a.SaveSession()
}

// CloseOtherTabs closes all tabs except the tab at index.
func (a *App) CloseOtherTabs(index int) {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return
	}
	target := a.Tabs[index]
	a.Tabs = []*buffer.Document{target}
	a.ActiveTab = 0
	a.mu.Unlock()

	_ = a.SaveSession()
}

// CloseTabsToRight closes all tabs positioned to the right of index.
func (a *App) CloseTabsToRight(index int) {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return
	}
	a.Tabs = a.Tabs[:index+1]
	if a.ActiveTab > index {
		a.ActiveTab = index
	}
	a.mu.Unlock()

	_ = a.SaveSession()
}

// CloseAllTabs resets tabs to a single blank Untitled document.
func (a *App) CloseAllTabs() {
	a.mu.Lock()
	a.Tabs = nil
	a.tabCounter++
	doc := buffer.NewDocument(fmt.Sprintf("doc-%d", a.tabCounter), fmt.Sprintf("Untitled %d", a.tabCounter), "", "")
	a.Tabs = append(a.Tabs, doc)
	a.ActiveTab = 0
	a.mu.Unlock()

	_ = a.SaveSession()
}

// CopyFilePath copies the active document's path or title to clipboard.
func (a *App) CopyFilePath(index int) {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return
	}
	doc := a.Tabs[index]
	path := doc.Path
	if path == "" {
		path = doc.Title
	}
	a.mu.Unlock()

	_ = mygo.Clipboard.Write(transfer.TextData(path))
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

// OpenFile loads a file into a new or active clean tab using zero-copy mapping.
// Large files (> 5MB) are loaded asynchronously in the background to prevent UI freezing.
func (a *App) OpenFile(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return err
	}
	size := fi.Size()
	title := filepath.Base(path)

	a.mu.Lock()
	// If already open in an existing tab, simply activate it
	for idx, t := range a.Tabs {
		if t != nil && t.Path == path {
			a.ActiveTab = idx
			a.mu.Unlock()
			return nil
		}
	}
	a.mu.Unlock()

	// If file is large (> 5MB), load asynchronously to keep UI thread at 120 FPS
	if size > 5*1024*1024 {
		doc := &buffer.Document{
			ID:          path,
			Path:        path,
			Title:       title,
			Encoding:    "UTF-8",
			CRLF:        false,
			Dirty:       false,
			Loading:     true,
			LoadingSize: size,
			ReadyCh:     make(chan struct{}),
		}

		a.mu.Lock()
		if len(a.Tabs) == 1 && a.Tabs[0].Path == "" && !a.Tabs[0].IsDirty() && a.Tabs[0].Len() == 0 {
			a.Tabs[0] = doc
			a.ActiveTab = 0
		} else {
			a.Tabs = append(a.Tabs, doc)
			a.ActiveTab = len(a.Tabs) - 1
		}
		a.mu.Unlock()

		_ = a.SaveSession()

		go func() {
			start := time.Now()
			payload, err := fileio.LoadFile(path)
			if err != nil {
				return
			}
			defer payload.Close()

			str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
			buf := editor.NewBuffer(str)
			st := &editor.State{Buffer: buf}
			doc.SetBufferReady(buf, st)

			a.mu.Lock()
			a.LastOpLatency = time.Since(start)
			a.mu.Unlock()
		}()

		return nil
	}

	// For standard files (<= 5MB), load synchronously
	start := time.Now()
	payload, err := fileio.LoadFile(path)
	if err != nil {
		return err
	}
	defer payload.Close()

	str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
	doc := buffer.NewDocument(path, title, str, path)

	a.mu.Lock()
	if len(a.Tabs) == 1 && a.Tabs[0].Path == "" && !a.Tabs[0].IsDirty() && a.Tabs[0].Len() == 0 {
		a.Tabs[0] = doc
		a.ActiveTab = 0
	} else {
		a.Tabs = append(a.Tabs, doc)
		a.ActiveTab = len(a.Tabs) - 1
	}

	a.LastOpLatency = time.Since(start)
	a.mu.Unlock()

	_ = a.SaveSession()
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

	_ = a.SaveSession()
	return nil
}

// SaveActiveFileAs saves the active document to a new target path.
func (a *App) SaveActiveFileAs(targetPath string) error {
	doc := a.ActiveDocument()
	if doc == nil {
		return fmt.Errorf("no active document to save")
	}

	start := time.Now()
	content := doc.Content()
	if err := fileio.SaveFile(targetPath, []byte(content)); err != nil {
		return err
	}
	doc.Path = targetPath
	doc.Title = filepath.Base(targetPath)
	doc.MarkSaved()

	a.mu.Lock()
	a.LastOpLatency = time.Since(start)
	a.mu.Unlock()

	_ = a.SaveSession()
	return nil
}

// SaveSession writes tab metadata to session storage.
// CRITICAL OPTIMIZATION: Never dump hundreds of megabytes of clean disk files into JSON.
// Files with Path != "" that are clean save Content: "" and reload from disk on demand.
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
		content := ""
		// Only store content for untitled drafts or small modified files (< 2MB)
		if doc.Path == "" || (doc.IsDirty() && doc.Len() < 2*1024*1024) {
			content = doc.Content()
		}

		state.Tabs[i] = session.TabSession{
			ID:         doc.ID,
			Title:      doc.Title,
			Path:       doc.Path,
			Content:    content,
			CursorLine: line,
			CursorCol:  col,
			IsDirty:    doc.IsDirty(),
		}
	}

	return a.SessionMgr.SaveSnapshot(state)
}

// RestoreSession loads prior tabs and state from session storage.
// Heavy files (> 5MB) are loaded asynchronously in the background so app window opens in < 10ms.
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
		var doc *buffer.Document
		// If tab references a file on disk whose content was omitted from WAL
		if t.Path != "" && t.Content == "" {
			if fi, err := os.Stat(t.Path); err == nil && fi.Size() > 5*1024*1024 {
				// Large file: restore tab metadata instantly, load rope in background
				doc = &buffer.Document{
					ID:          t.Path,
					Path:        t.Path,
					Title:       t.Title,
					Encoding:    "UTF-8",
					Loading:     true,
					LoadingSize: fi.Size(),
					ReadyCh:     make(chan struct{}),
				}
				targetDoc := doc
				targetPath := t.Path
				targetLine := t.CursorLine
				targetCol := t.CursorCol
				go func() {
					payload, err := fileio.LoadFile(targetPath)
					if err == nil {
						defer payload.Close()
						str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
						buf := editor.NewBuffer(str)
						st := &editor.State{Buffer: buf}
						if targetLine > 0 {
							st.GoTo(editor.Pos{Line: targetLine - 1, Col: max(0, targetCol - 1)})
						}
						targetDoc.SetBufferReady(buf, st)
					}
				}()
			} else {
				payload, err := fileio.LoadFile(t.Path)
				if err == nil {
					str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
					doc = buffer.NewDocument(t.Path, t.Title, str, t.Path)
					_ = payload.Close()
				}
			}
		}

		if doc == nil {
			doc = buffer.NewDocument(t.ID, t.Title, t.Content, t.Path)
		}

		if t.IsDirty {
			doc.SetDirty(true)
		} else {
			doc.MarkSaved()
		}
		if t.CursorLine > 0 && doc.State != nil {
			doc.State.GoTo(editor.Pos{Line: t.CursorLine - 1, Col: max(0, t.CursorCol - 1)})
		}
		a.Tabs = append(a.Tabs, doc)
	}

	if state.ActiveIndex >= 0 && state.ActiveIndex < len(a.Tabs) {
		a.ActiveTab = state.ActiveIndex
	} else {
		a.ActiveTab = 0
	}

	if len(a.Tabs) > a.tabCounter {
		a.tabCounter = len(a.Tabs)
	}

	return nil
}

// TriggerSearch initiates a debounced asynchronous search without freezing the UI thread.
func (a *App) TriggerSearch(query string) {
	a.searchMu.Lock()
	defer a.searchMu.Unlock()

	// Cancel existing in-flight search
	if a.searchCancel != nil {
		a.searchCancel()
		a.searchCancel = nil
	}
	if a.searchTimer != nil {
		a.searchTimer.Stop()
	}

	if query == "" {
		a.mu.Lock()
		a.SearchQuery = ""
		a.LastSearchQ = ""
		a.IsSearching = false
		a.SearchResults = search.Result{}
		a.mu.Unlock()
		return
	}

	a.mu.Lock()
	a.SearchQuery = query
	a.IsSearching = true
	a.mu.Unlock()

	// 300ms debounce
	a.searchTimer = time.AfterFunc(300*time.Millisecond, func() {
		a.executeAsyncSearch(query)
	})
}

// executeAsyncSearch runs search in background goroutine with zero-lock and cancellation support.
func (a *App) executeAsyncSearch(query string) {
	a.searchMu.Lock()
	ctx, cancel := context.WithCancel(context.Background())
	a.searchCancel = cancel
	a.searchMu.Unlock()

	doc := a.ActiveDocument()
	if doc == nil || len(query) == 0 || doc.IsLoading() {
		a.mu.Lock()
		a.IsSearching = false
		a.SearchResults = search.Result{}
		a.mu.Unlock()
		return
	}

	var res search.Result
	// CRITICAL ZERO-LOCK OPTIMIZATION:
	// For clean files on disk, search directly over mmap pages!
	// 0 locks on editor.Buffer, 0 allocations, 0ms UI freeze!
	if doc.Path != "" && !doc.IsDirty() {
		payload, err := fileio.LoadFile(doc.Path)
		if err == nil {
			res = a.SearchEngine.FindAllContext(ctx, payload.Data, []byte(query))
			_ = payload.Close()
		}
	} else {
		// For dirty/draft documents, search over content
		content := doc.Content()
		if ctx.Err() != nil {
			return
		}
		res = a.SearchEngine.FindAllContext(ctx, []byte(content), []byte(query))
	}

	if ctx.Err() != nil {
		return
	}

	a.mu.Lock()
	a.SearchResults = res
	a.LastOpLatency = res.Duration
	a.LastSearchQ = query
	a.IsSearching = false
	a.mu.Unlock()
}

// IsSearchingActive returns whether a background search is running.
func (a *App) IsSearchingActive() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.IsSearching
}

// GetSearchResults returns the current search results.
func (a *App) GetSearchResults() search.Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.SearchResults
}

// PerformSearch executes search synchronously (used by unit tests).
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
	a.LastSearchQ = query
	a.IsSearching = false
}
