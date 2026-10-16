package session

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/egoist/mygo/plugins/sqlite"
)

// Candidate 1: Atomic File Replacement (JSON + Temp + os.Rename)
type AtomicFileSaver struct {
	dir string
}

func (s *AtomicFileSaver) Save(data []byte) error {
	tmp := filepath.Join(s.dir, fmt.Sprintf("session-%d.tmp", time.Now().UnixNano()))
	dest := filepath.Join(s.dir, "session_atomic.json")

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

type BinaryWALSaver struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func NewBinaryWALSaver(path string) (*BinaryWALSaver, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	// Write magic if new
	info, err := f.Stat()
	if err == nil && info.Size() == 0 {
		if _, err := f.Write(walMagic[:]); err != nil {
			_ = f.Close()
			return nil, err
		}
	}
	return &BinaryWALSaver{file: f, path: path}, nil
}

func (w *BinaryWALSaver) Save(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Record frame: [Magic 4B][Len 4B][CRC32 4B][Data NB]
	header := make([]byte, 12)
	copy(header[0:4], walMagic[:])
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(data)))
	checksum := crc32.ChecksumIEEE(data)
	binary.LittleEndian.PutUint32(header[8:12], checksum)

	if _, err := w.file.Write(header); err != nil {
		return err
	}
	if _, err := w.file.Write(data); err != nil {
		return err
	}
	return nil
}

func (w *BinaryWALSaver) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Candidate 3: SQLite pure-Go in WAL mode
type SQLiteWALSaver struct {
	db  *sqlite.DB
	ctx context.Context
}

func NewSQLiteWALSaver(path string) (*SQLiteWALSaver, error) {
	ctx := context.Background()
	db, err := sqlite.Open(ctx, path, sqlite.OpenOptions{})
	if err != nil {
		return nil, err
	}

	// Configure SQLite WAL mode with synchronous = NORMAL
	if _, err := db.Execute(ctx, "PRAGMA journal_mode = WAL;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Execute(ctx, "PRAGMA synchronous = NORMAL;"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if _, err := db.Execute(ctx, "CREATE TABLE IF NOT EXISTS session_draft (id TEXT PRIMARY KEY, payload BLOB, updated_at INTEGER);"); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteWALSaver{db: db, ctx: ctx}, nil
}

func (s *SQLiteWALSaver) Save(data []byte) error {
	_, err := s.db.Execute(s.ctx, "INSERT OR REPLACE INTO session_draft (id, payload, updated_at) VALUES ('main', ?, ?);", data, time.Now().UnixNano())
	return err
}

func (s *SQLiteWALSaver) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// Test fixtures
var (
	benchSessionDir string
	payload512K     []byte
	payload5M       []byte
)

func TestMain(m *testing.M) {
	var err error
	benchSessionDir, err = os.MkdirTemp("", "gpad_session_bench_*")
	if err != nil {
		panic(err)
	}

	pattern := bytes.Repeat([]byte("GoPad++ Instant Crash-Resilient Session Draft. "), 20)
	for len(payload512K) < 512*1024 {
		payload512K = append(payload512K, pattern...)
	}
	payload512K = payload512K[:512*1024]

	for len(payload5M) < 5*1024*1024 {
		payload5M = append(payload5M, pattern...)
	}
	payload5M = payload5M[:5*1024*1024]

	code := m.Run()

	_ = os.RemoveAll(benchSessionDir)
	os.Exit(code)
}

func BenchmarkSessionSave_512K(b *testing.B) {
	b.Run("AtomicRename", func(b *testing.B) {
		saver := &AtomicFileSaver{dir: filepath.Join(benchSessionDir, "atomic_512k")}
		_ = os.MkdirAll(saver.dir, 0o755)
		b.SetBytes(int64(len(payload512K)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload512K); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("BinaryWAL", func(b *testing.B) {
		walPath := filepath.Join(benchSessionDir, "wal_512k.wal")
		saver, err := NewBinaryWALSaver(walPath)
		if err != nil {
			b.Fatalf("init failed: %v", err)
		}
		defer saver.Close()

		b.SetBytes(int64(len(payload512K)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload512K); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("SQLiteWAL", func(b *testing.B) {
		sqlitePath := filepath.Join(benchSessionDir, "sqlite_512k.db")
		saver, err := NewSQLiteWALSaver(sqlitePath)
		if err != nil {
			b.Fatalf("init failed: %v", err)
		}
		defer saver.Close()

		b.SetBytes(int64(len(payload512K)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload512K); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})
}

func BenchmarkSessionSave_5M(b *testing.B) {
	b.Run("AtomicRename", func(b *testing.B) {
		saver := &AtomicFileSaver{dir: filepath.Join(benchSessionDir, "atomic_5m")}
		_ = os.MkdirAll(saver.dir, 0o755)
		b.SetBytes(int64(len(payload5M)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload5M); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("BinaryWAL", func(b *testing.B) {
		walPath := filepath.Join(benchSessionDir, "wal_5m.wal")
		saver, err := NewBinaryWALSaver(walPath)
		if err != nil {
			b.Fatalf("init failed: %v", err)
		}
		defer saver.Close()

		b.SetBytes(int64(len(payload5M)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload5M); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})

	b.Run("SQLiteWAL", func(b *testing.B) {
		sqlitePath := filepath.Join(benchSessionDir, "sqlite_5m.db")
		saver, err := NewSQLiteWALSaver(sqlitePath)
		if err != nil {
			b.Fatalf("init failed: %v", err)
		}
		defer saver.Close()

		b.SetBytes(int64(len(payload5M)))
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			if err := saver.Save(payload5M); err != nil {
				b.Fatalf("failed: %v", err)
			}
		}
	})
}
