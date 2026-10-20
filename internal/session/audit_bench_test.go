package session

import (
	"testing"
)

func BenchmarkAuditSnapshot(b *testing.B) {
	m, err := NewManager(b.TempDir())
	if err != nil {
		b.Fatal(err)
	}
	defer m.Close()
	state := SessionState{Tabs: []TabSession{{Content: string(make([]byte, 4096))}}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := m.SaveSnapshot(state); err != nil {
			b.Fatal(err)
		}
	}
}
