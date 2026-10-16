package app

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"gpad/internal/fileio"
	"gpad/internal/search"
	"gpad/internal/session"
)

// Benchmark comparing Shadow Draft persistence vs Full String Serialization on 50MB and 500MB buffers.
func BenchmarkApp_ShadowDraft_Persistence(b *testing.B) {
	tmpDir := b.TempDir()
	sample50M := bytes.Repeat([]byte("2026-10-16 LOG LINE FOR BENCHMARK COMPARISON OF PERSISTENCE ARCHITECTURES\n"), 700000) // ~50MB

	b.Run("Candidate_A_FullStringSerialization_50M", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate JSON serialization of entire 50MB string (old approach)
			state := session.SessionState{
				ActiveIndex: 0,
				Tabs: []session.TabSession{
					{
						ID:         "doc-bench",
						Title:      "large.log",
						Content:    string(sample50M),
						IsDirty:    true,
						CursorLine: 1,
						CursorCol:  1,
					},
				},
			}
			data, err := json.Marshal(state)
			if err != nil {
				b.Fatal(err)
			}
			outPath := filepath.Join(tmpDir, fmt.Sprintf("out_a_%d.json", i))
			_ = os.WriteFile(outPath, data, 0o644)
			_ = os.Remove(outPath)
		}
	})

	b.Run("Candidate_B_ShadowDraft_MetadataPointer_50M", func(b *testing.B) {
		shadowDir := filepath.Join(tmpDir, "shadows")
		_ = os.MkdirAll(shadowDir, 0o755)

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Write raw draft to shadow file directly (no JSON serialization of text)
			shadowFile := filepath.Join(shadowDir, fmt.Sprintf("bench_%d.draft", i))
			if err := os.WriteFile(shadowFile, sample50M, 0o644); err != nil {
				b.Fatal(err)
			}
			// WAL state only contains lightweight metadata pointer
			state := session.SessionState{
				ActiveIndex: 0,
				Tabs: []session.TabSession{
					{
						ID:         "doc-bench",
						Title:      "large.log",
						ShadowPath: shadowFile,
						IsDirty:    true,
						CursorLine: 1,
						CursorCol:  1,
					},
				},
			}
			meta, err := json.Marshal(state)
			if err != nil {
				b.Fatal(err)
			}
			walPath := filepath.Join(tmpDir, fmt.Sprintf("wal_%d.wal", i))
			_ = os.WriteFile(walPath, meta, 0o644)

			_ = os.Remove(shadowFile)
			_ = os.Remove(walPath)
		}
	})
}

// Benchmark comparing Hybrid Search vs Full Editor String Extraction.
func BenchmarkApp_HybridSearch(b *testing.B) {
	tmpDir := b.TempDir()
	samplePath := filepath.Join(tmpDir, "search_sample_50m.log")
	chunk := []byte("TIMESTAMP=2026-10-16 ERROR: Connection timeout occurred on host worker-42\n")
	data := bytes.Repeat(chunk, 650000) // ~50MB
	if err := os.WriteFile(samplePath, data, 0o644); err != nil {
		b.Fatal(err)
	}

	query := []byte("Connection timeout")

	b.Run("Candidate_A_FullStringExtraction_Search", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulates ed.view.text.String() allocating 50MB string on heap
			textStr := string(data)
			var count int
			pos := 0
			for {
				idx := bytes.Index([]byte(textStr[pos:]), query)
				if idx == -1 {
					break
				}
				count++
				pos += idx + len(query)
				if count >= 10000 {
					break
				}
			}
		}
	})

	b.Run("Candidate_B_ZeroLock_MmapWorkerPool_HybridSearch", func(b *testing.B) {
		dispatcher := search.NewAdaptiveSearchDispatcher()
		payload, err := fileio.LoadFile(samplePath)
		if err != nil {
			b.Fatal(err)
		}
		defer payload.Close()

		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := dispatcher.FindAllContext(context.Background(), payload.Data, query)
			if res.Count == 0 {
				b.Fatalf("expected matches")
			}
		}
	})
}
