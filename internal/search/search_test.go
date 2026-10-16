package search

import (
	"bytes"
	"regexp"
	"testing"
)

func TestAdaptiveSearchSmallAndLarge(t *testing.T) {
	dispatcher := NewAdaptiveSearchDispatcher()

	// 1. Small buffer test (< 256KB)
	smallData := []byte("The quick brown fox jumps over the lazy dog. The fox is fast.")
	smallRes := dispatcher.FindAll(smallData, []byte("fox"))
	if smallRes.Count != 2 {
		t.Fatalf("expected 2 matches, got %d", smallRes.Count)
	}
	if smallRes.Worker != "SIMD-Sequential" {
		t.Fatalf("expected SIMD-Sequential worker, got %s", smallRes.Worker)
	}

	// 2. Large buffer test (>= 256KB)
	largeBuf := bytes.Repeat([]byte("GoPad++ fast text editor. Keyword123 appears here. "), 10000) // ~510KB
	largeRes := dispatcher.FindAll(largeBuf, []byte("Keyword123"))
	if largeRes.Count != 10000 {
		t.Fatalf("expected 10000 matches, got %d", largeRes.Count)
	}
	if largeRes.Worker != "Parallel-WorkerPool" {
		t.Fatalf("expected Parallel-WorkerPool worker, got %s", largeRes.Worker)
	}

	// 3. Regexp test
	re := regexp.MustCompile(`Keyword\d+`)
	regRes := dispatcher.FindRegex(largeBuf, re)
	if regRes.Count != 10000 {
		t.Fatalf("expected 10000 regex matches, got %d", regRes.Count)
	}
}

func TestAdaptiveSearchBoundarySpanning(t *testing.T) {
	dispatcher := NewAdaptiveSearchDispatcher()
	// Set artificial small threshold to trigger parallel workers
	dispatcher.crossoverThreshold = 100

	partA := bytes.Repeat([]byte("A"), 50)
	keyword := []byte("BOUNDARY_KEYWORD")
	partB := bytes.Repeat([]byte("B"), 50)

	// Keyword spans right across the 50-byte chunk boundary
	combined := append(partA, keyword...)
	combined = append(combined, partB...)

	res := dispatcher.FindAll(combined, keyword)
	if res.Count != 1 {
		t.Fatalf("expected 1 match across boundary, got %d", res.Count)
	}
	if res.Matches[0].Offset != 50 {
		t.Fatalf("expected match offset 50, got %d", res.Matches[0].Offset)
	}
}
