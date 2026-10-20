package app

import (
	"bytes"
	"context"
	"github.com/egoist/mygo/ui/editor"
	"gpad/internal/search"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"gpad/internal/buffer"
	"gpad/internal/session"
)

func TestAuditUntitledAndEmptyDirtyRecovery(t *testing.T) {
	for _, content := range []string{"initial untitled text", ""} {
		t.Run(content, func(t *testing.T) {
			dir := t.TempDir()
			m, err := session.NewManager(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer m.Close()
			a := &App{SessionMgr: m}
			d := a.NewTab("draft", content)
			if content == "" {
				d.SetDirty(true)
				d.Path = filepath.Join(dir, "original")
				os.WriteFile(d.Path, []byte("old disk text"), 0600)
			}
			if err := a.SaveSession(); err != nil {
				t.Fatal(err)
			}
			b := &App{SessionMgr: m}
			if err := b.RestoreSession(); err != nil {
				t.Fatal(err)
			}
			if got := b.Tabs[0].Content(); got != content {
				t.Fatalf("recovered %q, want %q", got, content)
			}
		})
	}
}

func TestAuditShadowFailurePreservesCheckpoint(t *testing.T) {
	dir := t.TempDir()
	m, err := session.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	a := &App{SessionMgr: m}
	d := a.NewTab("draft", "checkpoint")
	d.SetDirty(true)
	if err := a.SaveSession(); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, "shadows")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "shadows"), []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	d.Buffer.Edit(struct {
		Start, End int
		Text       string
	}{0, d.Len(), strings.Repeat("x", 2*1024*1024)})
	if err := a.SaveSession(); err == nil {
		t.Fatal("shadow failure reported success")
	}
	state, err := m.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if state.Tabs[0].Content != "checkpoint" {
		t.Fatalf("checkpoint replaced: %+v", state.Tabs[0])
	}
}

func TestAuditMappedEditorOwnsText(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source")
	text := strings.Repeat("mapped original\n", 1000)
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	a := &App{}
	if err := a.OpenFile(path); err != nil {
		t.Fatal(err)
	}
	// Replacing/truncating the source after OpenFile must not affect rope storage.
	if err := os.WriteFile(path, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	if got := a.Tabs[0].Content(); got != text {
		t.Fatal("editor retained mapped storage")
	}
}

func TestAuditConcurrentStop(t *testing.T) {
	a := NewAppWithSessionDir(t.TempDir())
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); a.Stop() }()
	}
	wg.Wait()
}

func TestAuditDocumentReadySignal(t *testing.T) {
	d := &buffer.Document{Loading: true, ReadyCh: make(chan struct{})}
	ready := buffer.NewDocument("id", "title", "text", "")
	d.SetBufferReady(ready.Buffer, ready.State)
	select {
	case <-d.ReadyCh:
	default:
		t.Fatal("ReadyCh not closed")
	}
}

// A blocked engine makes cancellation and publication interleavings deterministic.
type auditBlockingEngine struct {
	search.Engine
	started chan struct{}
	release chan struct{}
}

func (e *auditBlockingEngine) FindAllContext(ctx context.Context, data, query []byte) search.Result {
	close(e.started)
	<-e.release
	return search.Result{Count: 99}
}

func TestAuditSearchRejectsStaleGeneration(t *testing.T) {
	engine := &auditBlockingEngine{started: make(chan struct{}), release: make(chan struct{})}
	a := &App{SearchEngine: engine}
	a.NewTab("draft", "abc")
	done := make(chan struct{})
	go func() { defer close(done); a.executeAsyncSearch("old") }()
	<-engine.started
	a.TriggerSearch("")
	close(engine.release)
	<-done
	if a.GetSearchResults().Count != 0 || a.IsSearchingActive() {
		t.Fatal("canceled result published")
	}
	// A timer callback already dispatched before Stop cannot install a new context.
	a.searchMu.Lock()
	old := a.searchGeneration
	a.searchMu.Unlock()
	a.TriggerSearch("")
	a.executeSearchGeneration("old", old)
	if a.GetSearchResults().Count != 0 {
		t.Fatal("stale timer result published")
	}
}

func TestAuditWorkflowReportsSessionFailure(t *testing.T) {
	a := NewAppWithSessionDir(t.TempDir())
	defer a.Stop()
	if err := a.SessionMgr.Close(); err != nil {
		t.Fatal(err)
	}
	a.NewTab("draft", "valuable")
	if a.SessionError() == nil {
		t.Fatal("workflow dropped WAL failure")
	}
}

func TestAuditMissingDraftPreservesWAL(t *testing.T) {
	dir := t.TempDir()
	m, err := session.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveSnapshot(session.SessionState{Tabs: []session.TabSession{{ID: "draft", IsDirty: true, ShadowPath: filepath.Join(dir, "missing")}}}); err != nil {
		t.Fatal(err)
	}
	m.Close()
	before, err := os.ReadFile(filepath.Join(dir, "session.wal"))
	if err != nil {
		t.Fatal(err)
	}
	a := NewAppWithSessionDir(dir)
	a.Stop()
	after, err := os.ReadFile(filepath.Join(dir, "session.wal"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed recovery overwrote checkpoint")
	}
	if a.SessionError() == nil {
		t.Fatal("missing draft failure hidden")
	}
}

func TestAuditAsyncLoadFailureCompletes(t *testing.T) {
	a := &App{}
	d := &buffer.Document{Loading: true, ReadyCh: make(chan struct{})}
	a.loadDocument(filepath.Join(t.TempDir(), "missing"), d, 0, 0)
	if d.IsLoading() || d.LoadingError() == nil {
		t.Fatal("load failure not published")
	}
	select {
	case <-d.ReadyCh:
	default:
		t.Fatal("failure left readiness waiter blocked")
	}
}

func TestAuditAutosaveDoesNotReadUIState(t *testing.T) {
	a := NewAppWithSessionDir(t.TempDir())
	defer a.Stop()
	doc := a.NewTab("draft", "one\ntwo\nthree")
	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 100; i++ {
			doc.State.GoTo(editor.Pos{Line: i % 3})
			doc.CaretPosition()
		}
	}()
	for i := 0; i < 100; i++ {
		if err := a.SaveSession(); err != nil {
			t.Fatal(err)
		}
	}
	<-done
}

func TestAuditImmutablePrivateShadowGenerations(t *testing.T) {
	dir := t.TempDir()
	m, err := session.NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	a := &App{SessionMgr: m}
	d1 := a.NewTab("one", strings.Repeat("a", 2*1024*1024))
	d1.ID = "/one/shared.txt"
	d2 := a.NewTab("two", strings.Repeat("b", 2*1024*1024))
	d2.ID = "/two/shared.txt"
	if err := a.SaveSession(); err != nil {
		t.Fatal(err)
	}
	state, err := m.LoadSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	p1, p2 := state.Tabs[0].ShadowPath, state.Tabs[1].ShadowPath
	if p1 == p2 {
		t.Fatal("basename collision overwrote draft")
	}
	for _, path := range []string{p1, p2} {
		st, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if st.Mode().Perm() != 0600 {
			t.Fatalf("draft mode %o", st.Mode().Perm())
		}
	}
	// A failed WAL publication must leave the old referenced generation intact.
	m.Close()
	d1.Buffer.Edit(struct {
		Start, End int
		Text       string
	}{0, 1, "z"})
	if err := a.SaveSession(); err == nil {
		t.Fatal("closed WAL save succeeded")
	}
	data, err := os.ReadFile(p1)
	if err != nil {
		t.Fatal(err)
	}
	if data[0] != 'a' {
		t.Fatal("failed checkpoint mutated old draft")
	}
	st, err := os.Stat(filepath.Join(dir, "shadows"))
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0700 {
		t.Fatalf("shadow directory mode %o", st.Mode().Perm())
	}
}

func TestAuditSearchClearsResultsOnTabSwitch(t *testing.T) {
	a := &App{SearchEngine: search.NewAdaptiveSearchDispatcher()}
	a.NewTab("one", "one")
	a.NewTab("two", "two")
	a.PerformSearch("two")
	a.SwitchTab(0)
	if a.GetSearchResults().Count != 0 {
		t.Fatal("previous tab's matches retained")
	}
}

func TestAuditCloseCannotSaveUnloadedDraft(t *testing.T) {
	for _, failed := range []bool{false, true} {
		path := filepath.Join(t.TempDir(), "original")
		if err := os.WriteFile(path, []byte("valuable original"), 0600); err != nil {
			t.Fatal(err)
		}
		doc := &buffer.Document{Path: path, Dirty: true, Loading: true, ReadyCh: make(chan struct{})}
		if failed {
			doc.SetLoadError(os.ErrNotExist)
		}
		a := &App{Tabs: []*buffer.Document{doc}, OnConfirmClose: func(*buffer.Document) CloseAction { return CloseActionSave }}
		if a.CloseTab(0) {
			t.Fatal("unloaded draft closed with Save")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "valuable original" {
			t.Fatalf("original overwritten: %q %v", data, err)
		}
	}
}

func TestAuditManagerCreationFailureReported(t *testing.T) {
	path := filepath.Join(t.TempDir(), "blocked")
	if err := os.WriteFile(path, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}
	a := NewAppWithSessionDir(path)
	defer a.Stop()
	if err := a.SaveSession(); err == nil {
		t.Fatal("unavailable session manager reported successful save")
	}
	if a.SessionError() == nil {
		t.Fatal("manager failure not reported")
	}
}
