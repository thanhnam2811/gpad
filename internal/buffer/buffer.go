package buffer

import (
	"strings"
	"sync"

	"github.com/egoist/mygo/ui/editor"
)

// Document wraps an editor.Buffer with document-level metadata and metrics.
type Document struct {
	mu               sync.RWMutex
	ID               string
	Path             string
	Title            string
	Buffer           *editor.Buffer
	State            *editor.State
	Encoding         string
	CRLF             bool
	Dirty            bool
	LastSavedVersion uint64
	Loading          bool
	LoadingSize      int64
	ReadyCh          chan struct{}
}

// NewDocument creates a new Document instance from raw content.
func NewDocument(id, title, content string, path string) *Document {
	crlf := strings.Contains(content, "\r\n")
	buf := editor.NewBuffer(content)
	if path != "" {
		if lang := editor.LanguageFor(path); lang != nil {
			buf.SetLanguage(lang)
		}
	}

	st := &editor.State{
		Buffer: buf,
	}

	doc := &Document{
		ID:               id,
		Path:             path,
		Title:            title,
		Buffer:           buf,
		State:            st,
		Encoding:         "UTF-8",
		CRLF:             crlf,
		Dirty:            false,
		LastSavedVersion: buf.Version(),
		Loading:          false,
	}
	return doc
}

// SetBufferReady atomically updates the document with the loaded buffer and state.
func (d *Document) SetBufferReady(buf *editor.Buffer, st *editor.State) {
	d.mu.Lock()
	d.Buffer = buf
	d.State = st
	d.Loading = false
	if buf != nil {
		d.LastSavedVersion = buf.Version()
	}
	d.mu.Unlock()
}

// IsLoading returns whether the document is still being loaded asynchronously.
func (d *Document) IsLoading() bool {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.Loading
}

// IsDirty returns true if document has unsaved modifications.
func (d *Document) IsDirty() bool {
	if d.Dirty {
		return true
	}
	if d.Buffer == nil {
		return false
	}
	return d.Buffer.Modified()
}

// SetDirty manually sets the dirty flag (e.g. when restoring an unsaved draft).
func (d *Document) SetDirty(dirty bool) {
	d.Dirty = dirty
}

// CheckModified detects if the buffer version changed since last check.
func (d *Document) CheckModified() bool {
	if d.Buffer == nil {
		return false
	}
	v := d.Buffer.Version()
	if v != d.LastSavedVersion {
		d.LastSavedVersion = v
		d.Dirty = true
		return true
	}
	return false
}

// MarkSaved resets the modified flag and syncs version.
func (d *Document) MarkSaved() {
	d.Dirty = false
	if d.Buffer != nil {
		d.Buffer.MarkSaved()
		d.LastSavedVersion = d.Buffer.Version()
	}
}

// Content returns the full text content of the document.
func (d *Document) Content() string {
	if d.Buffer == nil {
		return ""
	}
	return d.Buffer.String()
}

// Len returns the byte length of the document text.
func (d *Document) Len() int {
	if d.Buffer == nil {
		return 0
	}
	return d.Buffer.Len()
}

// LineCount returns total number of lines.
func (d *Document) LineCount() int {
	if d.Buffer == nil {
		return 0
	}
	return d.Buffer.LineCount()
}

// LineEndingDesc returns "CRLF" or "LF".
func (d *Document) LineEndingDesc() string {
	if d.CRLF {
		return "CRLF"
	}
	return "LF"
}

// CaretPosition returns current line and column (1-indexed) of primary caret.
// Zero allocations: reads directly from editor.State cursor without extracting line text.
func (d *Document) CaretPosition() (line, col int) {
	d.mu.RLock()
	st := d.State
	buf := d.Buffer
	d.mu.RUnlock()

	if st == nil || buf == nil {
		return 1, 1
	}
	pos := st.CursorPos()
	return pos.Line + 1, pos.Col + 1
}

