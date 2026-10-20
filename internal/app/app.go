package app

import (
	"context"
	"fmt"
	"log"
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

// CloseAction defines the user response to an unsaved tab close confirmation.
type CloseAction int

const (
	CloseActionCancel CloseAction = iota
	CloseActionSave
	CloseActionDiscard
)

// CloseConfirmHandler is invoked when closing a modified document.
type CloseConfirmHandler func(doc *buffer.Document) CloseAction

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
	autoSaveDone   chan struct{}
	stopOnce       sync.Once
	persistErr     error
	sessionRetry   bool
	shadowPaths    map[string]bool
	restoreErr     error
	OnConfirmClose CloseConfirmHandler

	searchMu         sync.Mutex
	searchTimer      *time.Timer
	searchCancel     context.CancelFunc
	searchGeneration uint64
	searchStopped    bool
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
	sessMgr, managerErr := session.NewManager(sessionDir)

	a := &App{
		Tabs:         make([]*buffer.Document, 0, 8),
		ActiveTab:    0,
		SessionMgr:   sessMgr,
		SearchEngine: search.NewAdaptiveSearchDispatcher(),
		tabCounter:   0,
		stopAutoSave: make(chan struct{}),
		autoSaveDone: make(chan struct{}),
		restoreErr:   managerErr,
	}

	// Try restoring session; if none or empty, open default Untitled-1
	if err := a.RestoreSession(); err != nil {
		a.restoreErr = err
	}
	if a.restoreErr != nil {
		a.recordSessionError(a.restoreErr)
	}
	if len(a.Tabs) == 0 {
		a.NewTab("", "")
	}

	// Start continuous background auto-save loop
	go a.startAutoSaveLoop()

	return a
}

// startAutoSaveLoop periodically checks for buffer edits and persists drafts.
func (a *App) startAutoSaveLoop() {
	defer close(a.autoSaveDone)
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
	a.stopOnce.Do(func() {
		if a.stopAutoSave != nil {
			close(a.stopAutoSave)
		}
		if a.autoSaveDone != nil {
			<-a.autoSaveDone
		}
		a.searchMu.Lock()
		a.searchStopped = true
		a.searchGeneration++
		if a.searchTimer != nil {
			a.searchTimer.Stop()
		}
		if a.searchCancel != nil {
			a.searchCancel()
		}
		a.searchMu.Unlock()
		a.persistSession()
		if a.SessionMgr != nil {
			a.recordSessionError(a.SessionMgr.Close())
		}
	})
}

// CheckAutoSave checks if any tab has been modified since last save and commits to WAL.
func (a *App) CheckAutoSave() {
	a.mu.Lock()
	modified := a.sessionRetry
	for _, doc := range a.Tabs {
		if doc != nil && doc.CheckModified() {
			modified = true
		}
	}
	a.mu.Unlock()

	if modified {
		a.persistSession()
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
	if content != "" {
		doc.SetDirty(true)
	}

	a.Tabs = append(a.Tabs, doc)
	a.ActiveTab = len(a.Tabs) - 1
	a.mu.Unlock()

	a.refreshDocumentSearch()
	a.persistSession()
	return doc
}

// shadowDraftDir returns the directory where heavy file draft changes are persisted.
func (a *App) shadowDraftDir() string {
	if a.SessionMgr == nil || a.SessionMgr.Dir() == "" {
		homeDir, _ := os.UserHomeDir()
		return filepath.Join(homeDir, ".config", "gpad", "session", "shadows")
	}
	return filepath.Join(a.SessionMgr.Dir(), "shadows")
}

// CloseTab closes tab at index. If the document has unsaved modifications and
// OnConfirmClose is registered, it asks the user for confirmation.
// Returns true if the tab was closed, false if cancelled.
func (a *App) CloseTab(index int) bool {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return false
	}
	doc := a.Tabs[index]
	handler := a.OnConfirmClose
	a.mu.Unlock()

	if doc != nil && doc.IsDirty() && handler != nil {
		action := handler(doc)
		switch action {
		case CloseActionCancel:
			return false
		case CloseActionSave:
			if doc.IsLoading() || doc.LoadingError() != nil {
				return false
			}
			if doc.Path != "" {
				if err := fileio.SaveFile(doc.Path, []byte(doc.Content())); err != nil {
					return false
				}
				doc.MarkSaved()
			} else {
				// Untitled document requires explicit path
				return false
			}
		case CloseActionDiscard:
			// User chose to discard unsaved changes
			// Draft cleanup follows successful checkpoint publication.
		}
	}

	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) || a.Tabs[index] != doc {
		// Tab changed concurrently
		a.mu.Unlock()
		return false
	}

	// Draft cleanup follows successful checkpoint publication.
	a.Tabs = append(a.Tabs[:index], a.Tabs[index+1:]...)
	if len(a.Tabs) == 0 {
		a.tabCounter++
		newDoc := buffer.NewDocument(fmt.Sprintf("doc-%d", a.tabCounter), fmt.Sprintf("Untitled %d", a.tabCounter), "", "")
		a.Tabs = append(a.Tabs, newDoc)
		a.ActiveTab = 0
		a.mu.Unlock()
		a.refreshDocumentSearch()
		a.persistSession()
		return true
	}

	if a.ActiveTab >= len(a.Tabs) {
		a.ActiveTab = len(a.Tabs) - 1
	}
	a.mu.Unlock()

	a.refreshDocumentSearch()
	a.persistSession()
	return true
}

// SwitchTab changes active tab index.
func (a *App) SwitchTab(index int) {
	a.mu.Lock()
	if index >= 0 && index < len(a.Tabs) {
		a.ActiveTab = index
	}
	a.mu.Unlock()

	a.refreshDocumentSearch()
	a.persistSession()
}

// NextTab cycles to next tab (Ctrl+Tab).
func (a *App) NextTab() {
	a.mu.Lock()
	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab + 1) % len(a.Tabs)
	}
	a.mu.Unlock()

	a.refreshDocumentSearch()
	a.persistSession()
}

// PrevTab cycles to previous tab.
func (a *App) PrevTab() {
	a.mu.Lock()
	if len(a.Tabs) > 1 {
		a.ActiveTab = (a.ActiveTab - 1 + len(a.Tabs)) % len(a.Tabs)
	}
	a.mu.Unlock()

	a.refreshDocumentSearch()
	a.persistSession()
}

// CloseOtherTabs closes all tabs except the tab at index.
func (a *App) CloseOtherTabs(index int) {
	a.mu.Lock()
	if index < 0 || index >= len(a.Tabs) {
		a.mu.Unlock()
		return
	}
	target := a.Tabs[index]
	a.mu.Unlock()

	for i := len(a.Tabs) - 1; i >= 0; i-- {
		a.mu.Lock()
		if i >= len(a.Tabs) {
			a.mu.Unlock()
			continue
		}
		doc := a.Tabs[i]
		a.mu.Unlock()
		if doc == target {
			continue
		}
		a.CloseTab(i)
	}
}

// CloseTabsToRight closes all tabs positioned to the right of index.
func (a *App) CloseTabsToRight(index int) {
	a.mu.Lock()
	total := len(a.Tabs)
	a.mu.Unlock()
	for i := total - 1; i > index; i-- {
		a.CloseTab(i)
	}
}

// CloseAllTabs resets tabs to a single blank Untitled document, respecting confirmations.
func (a *App) CloseAllTabs() {
	for len(a.Tabs) > 0 {
		origLen := len(a.Tabs)
		if !a.CloseTab(len(a.Tabs) - 1) {
			break
		}
		if len(a.Tabs) == 1 && a.Tabs[0].Path == "" && !a.Tabs[0].IsDirty() && origLen == 1 {
			break
		}
	}
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
			a.refreshDocumentSearch()
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

		a.refreshDocumentSearch()
		sessionErr := a.persistSession()

		go func() {
			start := time.Now()
			a.loadDocument(path, doc, 0, 0)
			a.mu.Lock()
			a.LastOpLatency = time.Since(start)
			a.mu.Unlock()
		}()

		return sessionErr
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

	a.refreshDocumentSearch()
	return a.persistSession()
}

// SaveActiveFile saves the currently active document to disk.
func (a *App) SaveActiveFile() error {
	doc := a.ActiveDocument()
	if doc == nil || doc.Path == "" || doc.IsLoading() || doc.LoadingError() != nil {
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

	return a.persistSession()
}

// SaveActiveFileAs saves the active document to a new target path.
func (a *App) SaveActiveFileAs(targetPath string) error {
	doc := a.ActiveDocument()
	if doc == nil || doc.IsLoading() || doc.LoadingError() != nil {
		return fmt.Errorf("no ready document to save")
	}

	start := time.Now()
	content := doc.Content()
	if err := fileio.SaveFile(targetPath, []byte(content)); err != nil {
		return err
	}
	a.mu.Lock()
	doc.Path = targetPath
	doc.Title = filepath.Base(targetPath)
	doc.MarkSaved()
	a.LastOpLatency = time.Since(start)
	a.mu.Unlock()

	return a.persistSession()
}

// SaveSession writes tab metadata to session storage.
// Clean disk files omit content. Large drafts use immutable private shadow files
// published before the WAL checkpoint that references them.
func (a *App) SaveSession() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.restoreErr != nil {
		return fmt.Errorf("session recovery failed; preserving checkpoint: %w", a.restoreErr)
	}
	if a.SessionMgr == nil {
		return nil
	}

	if a.shadowPaths == nil {
		previous, err := a.SessionMgr.LoadSnapshot()
		if err != nil {
			return err
		}
		a.shadowPaths = make(map[string]bool)
		if previous != nil {
			for _, tab := range previous.Tabs {
				if tab.ShadowPath != "" {
					a.shadowPaths[tab.ShadowPath] = true
				}
			}
		}
	}

	state := session.SessionState{
		ActiveIndex: a.ActiveTab,
		Tabs:        make([]session.TabSession, len(a.Tabs)),
	}

	shadowDir := a.shadowDraftDir()
	if err := os.MkdirAll(shadowDir, 0o700); err != nil {
		return err
	}
	if err := os.Chmod(shadowDir, 0o700); err != nil {
		return err
	}

	for i, doc := range a.Tabs {
		if doc == nil {
			continue
		}
		if err := doc.LoadingError(); err != nil {
			return fmt.Errorf("load %s: %w", doc.Path, err)
		}
		if doc.IsLoading() && doc.IsDirty() {
			return fmt.Errorf("draft %s still loading", doc.Title)
		}
		line, col := doc.CachedCaretPosition()
		content := ""
		shadowPath := ""

		if doc.IsDirty() || doc.Path == "" {
			if doc.Len() < 2*1024*1024 {
				content = doc.Content()
			} else {
				// Immutable draft generations keep the prior checkpoint valid
				// until the WAL snapshot referencing this generation is durable.
				f, err := os.CreateTemp(shadowDir, "draft-*.draft")
				if err != nil {
					return err
				}
				shadowPath = f.Name()
				if err := f.Close(); err != nil {
					return err
				}
				if err := fileio.SaveFile(shadowPath, []byte(doc.Content())); err != nil {
					return err
				}
			}
		}

		state.Tabs[i] = session.TabSession{
			ID:         doc.ID,
			Title:      doc.Title,
			Path:       doc.Path,
			Content:    content,
			ShadowPath: shadowPath,
			CursorLine: line,
			CursorCol:  col,
			IsDirty:    doc.IsDirty(),
		}
	}

	if err := a.SessionMgr.SaveSnapshot(state); err != nil {
		return err
	}
	keep := make(map[string]bool)
	for _, tab := range state.Tabs {
		if tab.ShadowPath != "" {
			keep[tab.ShadowPath] = true
		}
	}
	// Only remove drafts referenced by the prior checkpoint. Unknown files may
	// be recovery artifacts and are preserved for manual recovery.
	previousPaths := a.shadowPaths
	a.shadowPaths = keep
	for path := range previousPaths {
		if filepath.Dir(path) == shadowDir && !keep[path] {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}
	return nil
}

// RestoreSession loads prior tabs and state from session storage.
// Heavy files load asynchronously. Unsaved shadows must finish loading before
// the workspace can publish replacement checkpoints.
func (a *App) RestoreSession() (resultErr error) {
	defer func() {
		if resultErr != nil {
			a.mu.Lock()
			a.restoreErr = resultErr
			a.mu.Unlock()
		}
	}()
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

		// Draft recovery is transactional: never silently fall back to the
		// original file if unsaved content cannot be loaded.
		if t.ShadowPath != "" {
			fi, err := os.Stat(t.ShadowPath)
			if err != nil {
				return fmt.Errorf("restore draft %s: %w", t.ShadowPath, err)
			}
			if fi.Size() > 5*1024*1024 {
				doc = &buffer.Document{ID: t.ID, Path: t.Path, Title: t.Title, Encoding: "UTF-8", Dirty: true, Loading: true, LoadingSize: fi.Size(), ReadyCh: make(chan struct{})}
				go a.loadDocument(t.ShadowPath, doc, t.CursorLine, t.CursorCol)
			} else {
				payload, err := fileio.LoadFile(t.ShadowPath)
				if err != nil {
					return fmt.Errorf("restore draft %s: %w", t.ShadowPath, err)
				}
				str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
				doc = buffer.NewDocument(t.ID, t.Title, str, t.Path)
				_ = payload.Close()
				doc.SetDirty(true)
			}
		}

		// Case 2: Clean file on disk whose content was omitted from WAL
		if doc == nil && !t.IsDirty && t.Path != "" && t.Content == "" {
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
				go a.loadDocument(t.Path, doc, t.CursorLine, t.CursorCol)
			} else {
				payload, err := fileio.LoadFile(t.Path)
				if err != nil {
					return fmt.Errorf("restore file %s: %w", t.Path, err)
				}
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

		if t.IsDirty || t.ShadowPath != "" {
			doc.SetDirty(true)
		} else {
			doc.MarkSaved()
		}
		if t.CursorLine > 0 && !doc.IsLoading() && doc.State != nil {
			doc.State.GoTo(editor.Pos{Line: t.CursorLine - 1, Col: max(0, t.CursorCol-1)})
			doc.CaretPosition()
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

	if a.searchStopped {
		return
	}
	a.searchGeneration++
	generation := a.searchGeneration
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
		a.executeSearchGeneration(query, generation)
	})
}

// executeAsyncSearch runs search in background goroutine with zero-lock and cancellation support.
func (a *App) executeAsyncSearch(query string) {
	a.searchMu.Lock()
	generation := a.searchGeneration
	a.searchMu.Unlock()
	a.executeSearchGeneration(query, generation)
}

func (a *App) executeSearchGeneration(query string, generation uint64) {
	a.searchMu.Lock()
	if a.searchStopped || generation != a.searchGeneration {
		a.searchMu.Unlock()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	a.searchCancel = cancel
	a.searchMu.Unlock()
	defer cancel()

	doc := a.ActiveDocument()
	if doc == nil || len(query) == 0 || doc.IsLoading() || doc.LoadingError() != nil {
		a.searchMu.Lock()
		defer a.searchMu.Unlock()
		if generation != a.searchGeneration || a.searchStopped {
			return
		}
		a.mu.Lock()
		a.IsSearching = false
		a.SearchResults = search.Result{}
		a.mu.Unlock()
		return
	}
	a.mu.Lock()
	path := doc.Path
	a.mu.Unlock()
	version := doc.Buffer.Version()

	var res search.Result
	// CRITICAL ZERO-LOCK OPTIMIZATION:
	// For clean files on disk, search directly over mmap pages!
	// 0 locks on editor.Buffer, 0 allocations, 0ms UI freeze!
	if path != "" && !doc.IsDirty() {
		payload, err := fileio.LoadFile(path)
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

	a.searchMu.Lock()
	defer a.searchMu.Unlock()
	if ctx.Err() != nil || generation != a.searchGeneration || a.searchStopped {
		return
	}
	a.mu.Lock()
	if a.ActiveTab < 0 || a.ActiveTab >= len(a.Tabs) || a.Tabs[a.ActiveTab] != doc || doc.Path != path || doc.Buffer.Version() != version {
		a.IsSearching = false
		a.SearchResults = search.Result{}
		a.mu.Unlock()
		return
	}
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
		a.mu.Lock()
		a.SearchResults = search.Result{}
		a.mu.Unlock()
		return
	}

	content := doc.Content()
	res := a.SearchEngine.FindAll([]byte(content), []byte(query))
	a.mu.Lock()
	defer a.mu.Unlock()
	a.SearchResults = res
	a.LastOpLatency = res.Duration
	a.LastSearchQ = query
	a.IsSearching = false
}

// recordSessionError makes failures from workflows without error returns visible.
func (a *App) recordSessionError(err error) {
	if err == nil {
		return
	}
	a.mu.Lock()
	a.persistErr = err
	a.sessionRetry = true
	a.StatusMessage = err.Error()
	a.mu.Unlock()
	log.Printf("gpad persistence: %v", err)
}

func (a *App) persistSession() error {
	err := a.SaveSession()
	a.recordSessionError(err)
	if err == nil {
		a.mu.Lock()
		a.sessionRetry = false
		a.mu.Unlock()
	}
	return err
}

// SessionError returns the most recent persistence failure, including shutdown.
func (a *App) SessionError() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.persistErr
}

// loadDocument always publishes a terminal result, including failures.
func (a *App) loadDocument(path string, doc *buffer.Document, line, col int) {
	payload, err := fileio.LoadFile(path)
	if err != nil {
		doc.SetLoadError(err)
		a.recordSessionError(err)
		return
	}
	defer payload.Close()
	// NewBuffer's rope clones each leaf; no mmap-backed string escapes this call.
	text := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
	buf := editor.NewBuffer(text)
	if lang := editor.LanguageFor(doc.Path); lang != nil {
		buf.SetLanguage(lang)
	}
	st := &editor.State{Buffer: buf}
	if line > 0 {
		st.GoTo(editor.Pos{Line: line - 1, Col: max(0, col-1)})
	}
	doc.SetBufferReady(buf, st)
}

// OperationLatency returns the last measured operation time safely across workers.
func (a *App) OperationLatency() time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.LastOpLatency
}

// refreshDocumentSearch invalidates results from a previously active document.
func (a *App) refreshDocumentSearch() {
	a.mu.Lock()
	query := a.SearchQuery
	a.SearchResults = search.Result{}
	a.LastSearchQ = ""
	a.mu.Unlock()
	a.TriggerSearch(query)
}
