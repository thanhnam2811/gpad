package buffer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
	"unsafe"

	"github.com/egoist/mygo/ui/editor"
	"gpad/internal/fileio"
)

// Candidate A: Synchronous Ingestion (Blocks UI thread)
func loadDocumentSync(path string) (*Document, time.Duration, error) {
	t0 := time.Now()
	payload, err := fileio.LoadFile(path)
	if err != nil {
		return nil, 0, err
	}
	defer payload.Close()

	title := filepath.Base(path)
	str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
	doc := NewDocument(path, title, str, path)
	dur := time.Since(t0)
	return doc, dur, nil
}

// Candidate B: Asynchronous Ingestion (Immediate UI Viewport, Background Worker)
type AsyncDocResult struct {
	Doc          *Document
	UIBlockTime  time.Duration
	TotalRopeDur time.Duration
}

func loadDocumentAsync(path string) (*Document, time.Duration, <-chan time.Duration, error) {
	t0 := time.Now()
	fi, err := os.Stat(path)
	if err != nil {
		return nil, 0, nil, err
	}

	title := filepath.Base(path)
	doc := &Document{
		ID:          path,
		Path:        path,
		Title:       title,
		Encoding:    "UTF-8",
		CRLF:        false,
		Dirty:       false,
		Loading:     true,
		LoadingSize: fi.Size(),
	}
	uiBlock := time.Since(t0)

	doneCh := make(chan time.Duration, 1)
	go func() {
		bgStart := time.Now()
		payload, err := fileio.LoadFile(path)
		if err != nil {
			return
		}
		defer payload.Close()

		str := unsafe.String(unsafe.SliceData(payload.Data), len(payload.Data))
		buf := editor.NewBuffer(str)
		st := &editor.State{Buffer: buf}

		doc.SetBufferReady(buf, st)
		doneCh <- time.Since(bgStart)
	}()

	return doc, uiBlock, doneCh, nil
}

func BenchmarkDocumentIngestion_500M(b *testing.B) {
	path := "../../scratch/test_500m.log"
	info, err := os.Stat(path)
	if err != nil {
		b.Skipf("500M test log not found: %v", err)
	}
	b.SetBytes(info.Size())

	b.Run("Candidate_A_SyncIngestion_UIBlock", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc, dur, err := loadDocumentSync(path)
			if err != nil || doc == nil {
				b.Fatalf("failed: %v", err)
			}
			_ = dur
		}
	})

	b.Run("Candidate_B_AsyncIngestion_UIBlock", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			doc, uiBlock, doneCh, err := loadDocumentAsync(path)
			if err != nil || doc == nil {
				b.Fatalf("failed: %v", err)
			}
			<-doneCh
			_ = uiBlock
		}
	})
}
