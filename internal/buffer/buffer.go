package buffer

import (
	"strings"
	"unicode/utf8"

	"github.com/egoist/mygo/ui/editor"
)

// Document wraps an editor.Buffer with document-level metadata and metrics.
type Document struct {
	ID       string
	Path     string
	Title    string
	Buffer   *editor.Buffer
	State    *editor.State
	Encoding string
	CRLF     bool
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
		ID:       id,
		Path:     path,
		Title:    title,
		Buffer:   buf,
		State:    st,
		Encoding: "UTF-8",
		CRLF:     crlf,
	}
	return doc
}

// IsDirty returns true if document has unsaved modifications.
func (d *Document) IsDirty() bool {
	if d.Buffer == nil {
		return false
	}
	return d.Buffer.Modified()
}

// MarkSaved resets the modified flag.
func (d *Document) MarkSaved() {
	if d.Buffer != nil {
		d.Buffer.MarkSaved()
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
func (d *Document) CaretPosition() (line, col int) {
	if d.State == nil || d.Buffer == nil {
		return 1, 1
	}
	pos := d.State.CursorPos()
	line = pos.Line + 1
	lineText := d.Buffer.Line(pos.Line)
	byteOffset := pos.Col
	if byteOffset > len(lineText) {
		byteOffset = len(lineText)
	}
	col = utf8.RuneCountInString(lineText[:byteOffset]) + 1
	return line, col
}
