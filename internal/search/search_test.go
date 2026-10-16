package search

import (
	"testing"
)

func TestSequentialEngineFindAll(t *testing.T) {
	engine := &SequentialEngine{}
	data := []byte("The quick brown fox jumps over the lazy dog. The fox is quick.")
	query := []byte("fox")

	res := engine.FindAll(data, query)
	if res.Count != 2 {
		t.Fatalf("expected 2 matches, got %d", res.Count)
	}

	if res.Matches[0].Offset != 16 {
		t.Fatalf("first match offset expected 16, got %d", res.Matches[0].Offset)
	}
	if res.Matches[1].Offset != 49 {
		t.Fatalf("second match offset expected 49, got %d", res.Matches[1].Offset)
	}
}
