package fileio

import (
	"path/filepath"
	"testing"
)

func BenchmarkAuditSave(b *testing.B) {
	path := filepath.Join(b.TempDir(), "document")
	data := make([]byte, 4096)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := SaveFile(path, data); err != nil {
			b.Fatal(err)
		}
	}
}
