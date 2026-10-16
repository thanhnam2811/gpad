package fileio

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
)

// Candidate 1: os.ReadFile
func readFileStandard(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// Candidate 2: syscall.Mmap zero-copy
func readFileMmap(path string) ([]byte, func() error, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, nil, err
	}

	size := int(stat.Size())
	if size == 0 {
		return []byte{}, func() error { return nil }, nil
	}

	data, err := syscall.Mmap(int(file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		return nil, nil, err
	}

	unmap := func() error {
		return syscall.Munmap(data)
	}

	return data, unmap, nil
}

// Candidate 3: Chunked Reader with bufio.Reader + sync.Pool
var chunkPool = sync.Pool{
	New: func() any {
		b := make([]byte, 64*1024) // 64KB chunk
		return &b
	},
}

func readFileChunkedPool(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()

	out := make([]byte, 0, size)
	reader := bufio.NewReaderSize(file, 64*1024)

	bufPtr := chunkPool.Get().(*[]byte)
	buf := *bufPtr
	defer chunkPool.Put(bufPtr)

	for {
		n, rerr := reader.Read(buf)
		if n > 0 {
			out = append(out, buf[:n]...)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			return nil, rerr
		}
	}

	return out, nil
}

// Test fixtures created once in TestMain
var (
	benchDir     string
	benchFile10K string
	benchFile1M  string
	benchFile50M string
)

func TestMain(m *testing.M) {
	var err error
	benchDir, err = os.MkdirTemp("", "gpad_fileio_bench_*")
	if err != nil {
		panic(err)
	}

	benchFile10K = filepath.Join(benchDir, "file_10k.txt")
	benchFile1M = filepath.Join(benchDir, "file_1m.txt")
	benchFile50M = filepath.Join(benchDir, "file_50m.txt")

	pattern := []byte("The quick brown fox jumps over the lazy dog. Gopad++ high-speed editor.\n")
	createFile := func(path string, size int) {
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		defer f.Close()

		w := bufio.NewWriter(f)
		written := 0
		for written < size {
			n := len(pattern)
			if written+n > size {
				n = size - written
			}
			_, _ = w.Write(pattern[:n])
			written += n
		}
		_ = w.Flush()
	}

	createFile(benchFile10K, 10*1024)
	createFile(benchFile1M, 1*1024*1024)
	createFile(benchFile50M, 50*1024*1024)

	code := m.Run()

	_ = os.RemoveAll(benchDir)
	os.Exit(code)
}

func BenchmarkFileIO_10K(b *testing.B) {
	b.Run("os.ReadFile", func(b *testing.B) {
		b.SetBytes(10 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileStandard(benchFile10K)
			if err != nil || len(data) != 10*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("syscall.Mmap", func(b *testing.B) {
		b.SetBytes(10 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, unmap, err := readFileMmap(benchFile10K)
			if err != nil || len(data) != 10*1024 {
				b.Fatalf("failed: %v", err)
			}
			_ = unmap()
		}
	})

	b.Run("bufio.ChunkPool", func(b *testing.B) {
		b.SetBytes(10 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileChunkedPool(benchFile10K)
			if err != nil || len(data) != 10*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})
}

func BenchmarkFileIO_1M(b *testing.B) {
	b.Run("os.ReadFile", func(b *testing.B) {
		b.SetBytes(1 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileStandard(benchFile1M)
			if err != nil || len(data) != 1*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("syscall.Mmap", func(b *testing.B) {
		b.SetBytes(1 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, unmap, err := readFileMmap(benchFile1M)
			if err != nil || len(data) != 1*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
			_ = unmap()
		}
	})

	b.Run("bufio.ChunkPool", func(b *testing.B) {
		b.SetBytes(1 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileChunkedPool(benchFile1M)
			if err != nil || len(data) != 1*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})
}

func BenchmarkFileIO_50M(b *testing.B) {
	b.Run("os.ReadFile", func(b *testing.B) {
		b.SetBytes(50 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileStandard(benchFile50M)
			if err != nil || len(data) != 50*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("syscall.Mmap", func(b *testing.B) {
		b.SetBytes(50 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, unmap, err := readFileMmap(benchFile50M)
			if err != nil || len(data) != 50*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
			_ = unmap()
		}
	})

	b.Run("bufio.ChunkPool", func(b *testing.B) {
		b.SetBytes(50 * 1024 * 1024)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			data, err := readFileChunkedPool(benchFile50M)
			if err != nil || len(data) != 50*1024*1024 {
				b.Fatalf("failed: %v", err)
			}
		}
	})
}
