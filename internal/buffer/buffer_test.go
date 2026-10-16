package buffer

import (
	"testing"
)

func TestDocumentLifecycle(t *testing.T) {
	content := "Hello Gopad++\nLine 2 text\nLine 3 ending with LF"
	doc := NewDocument("tab-1", "Untitled-1", content, "")

	if doc.Title != "Untitled-1" {
		t.Fatalf("expected title 'Untitled-1', got %q", doc.Title)
	}
	if doc.LineCount() != 3 {
		t.Fatalf("expected 3 lines, got %d", doc.LineCount())
	}
	if doc.IsDirty() {
		t.Fatalf("new document should not be dirty")
	}

	doc.Buffer.Edit(struct{ Start, End int; Text string }{Start: 0, End: 0, Text: "// "})
	if !doc.IsDirty() {
		t.Fatalf("document should be marked dirty after edit")
	}

	doc.MarkSaved()
	if doc.IsDirty() {
		t.Fatalf("document should not be dirty after MarkSaved()")
	}

	line, col := doc.CaretPosition()
	if line < 1 || col < 1 {
		t.Fatalf("invalid line/col: line=%d, col=%d", line, col)
	}
}
