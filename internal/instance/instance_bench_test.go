package instance

import (
	"path/filepath"
	"testing"
)

func BenchmarkSingleInstance_Forward(b *testing.B) {
	tempDir := b.TempDir()
	sockPath := filepath.Join(tempDir, "bench_gpad.sock")

	l, err := StartListener(sockPath, func(path, token string) {}, func(token string) {})
	if err != nil {
		b.Fatalf("StartListener failed: %v", err)
	}
	defer l.Close()

	files := []string{"benchmark_sample.txt"}

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		ok := TryForward(sockPath, files)
		if !ok {
			b.Fatalf("TryForward failed")
		}
	}
}
