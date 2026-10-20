package session

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"
)

func TestSessionWALBasic(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state1 := SessionState{
		ActiveIndex: 0,
		Tabs: []TabSession{
			{ID: "t1", Title: "Draft 1", Content: "Content 1", CursorLine: 1, CursorCol: 1, IsDirty: true},
		},
	}
	if err := mgr.SaveSnapshot(state1); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}

	state2 := SessionState{
		ActiveIndex: 1,
		Tabs: []TabSession{
			{ID: "t1", Title: "Draft 1", Content: "Content 1 edited", CursorLine: 2, CursorCol: 3, IsDirty: true},
			{ID: "t2", Title: "Draft 2", Content: "New Note 2", CursorLine: 1, CursorCol: 1, IsDirty: true},
		},
	}
	if err := mgr.SaveSnapshot(state2); err != nil {
		t.Fatalf("SaveSnapshot 2 failed: %v", err)
	}

	if err := mgr.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Reopen and recover
	mgr2, err := NewManager(dir)
	if err != nil {
		t.Fatalf("reopen NewManager failed: %v", err)
	}
	defer mgr2.Close()

	recovered, err := mgr2.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}

	if recovered.ActiveIndex != 1 {
		t.Fatalf("expected ActiveIndex 1, got %d", recovered.ActiveIndex)
	}
	if len(recovered.Tabs) != 2 {
		t.Fatalf("expected 2 tabs, got %d", len(recovered.Tabs))
	}
	if recovered.Tabs[0].Content != "Content 1 edited" {
		t.Fatalf("expected 'Content 1 edited', got %q", recovered.Tabs[0].Content)
	}
}

func TestSessionWALCrashRecovery_TruncatedTail(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state := SessionState{
		ActiveIndex: 0,
		Tabs: []TabSession{
			{ID: "t1", Title: "Note 1", Content: "Valid text before crash", CursorLine: 1, CursorCol: 1},
		},
	}
	if err := mgr.SaveSnapshot(state); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	_ = mgr.Close()

	// Simulate crash: append incomplete garbage bytes at end of WAL
	walPath := filepath.Join(dir, "session.wal")
	f, err := os.OpenFile(walPath, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open wal failed: %v", err)
	}
	_, _ = f.Write([]byte{'G', 'P', 'W', '1', 0, 0, 1, 0}) // Incomplete 8-byte corrupted header
	_ = f.Close()

	// Recover should successfully read valid state before crash
	mgr2, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager after crash failed: %v", err)
	}
	defer mgr2.Close()

	recovered, err := mgr2.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if len(recovered.Tabs) != 1 || recovered.Tabs[0].Content != "Valid text before crash" {
		t.Fatalf("expected recovered valid state, got: %+v", recovered)
	}
}

func TestSessionWALCrashRecovery_CorruptedCRC(t *testing.T) {
	dir := t.TempDir()
	mgr, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	state := SessionState{
		ActiveIndex: 0,
		Tabs: []TabSession{
			{ID: "t1", Title: "Valid Note", Content: "Good data", CursorLine: 1, CursorCol: 1},
		},
	}
	if err := mgr.SaveSnapshot(state); err != nil {
		t.Fatalf("SaveSnapshot failed: %v", err)
	}
	_ = mgr.Close()

	// Corrupt the last 2 bytes of the payload
	walPath := filepath.Join(dir, "session.wal")
	info, err := os.Stat(walPath)
	if err != nil {
		t.Fatalf("stat failed: %v", err)
	}

	f, err := os.OpenFile(walPath, os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}
	// Append bad record with invalid CRC32
	badHeader := make([]byte, 12)
	copy(badHeader[0:4], walMagic[:])
	binary.LittleEndian.PutUint32(badHeader[4:8], 10)
	binary.LittleEndian.PutUint32(badHeader[8:12], 0xDEADBEEF) // False CRC
	_, _ = f.Seek(0, 2)
	_, _ = f.Write(badHeader)
	_, _ = f.Write([]byte("0123456789"))
	_ = f.Close()

	_ = info

	// Recover
	mgr2, err := NewManager(dir)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}
	defer mgr2.Close()

	recovered, err := mgr2.LoadSnapshot()
	if err != nil {
		t.Fatalf("LoadSnapshot failed: %v", err)
	}
	if len(recovered.Tabs) != 1 || recovered.Tabs[0].Content != "Good data" {
		t.Fatalf("expected fallback to good data, got: %+v", recovered)
	}
}

func TestAppendAfterCorruptTail(t *testing.T) {
	dir := t.TempDir()
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err = m.SaveSnapshot(SessionState{ActiveIndex: 1}); err != nil {
		t.Fatal(err)
	}
	m.Close()
	f, err := os.OpenFile(filepath.Join(dir, "session.wal"), os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	f.Write([]byte("broken"))
	f.Close()
	m, err = NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err = m.SaveSnapshot(SessionState{ActiveIndex: 2}); err != nil {
		t.Fatal(err)
	}
	s, err := m.LoadSnapshot()
	if err != nil || s.ActiveIndex != 2 {
		t.Fatalf("recovery %+v %v", s, err)
	}
	st, _ := os.Stat(filepath.Join(dir, "session.wal"))
	if st.Mode().Perm() != 0600 {
		t.Fatalf("mode %o", st.Mode().Perm())
	}
}

func TestCompactionFailureKeepsRecoverableWAL(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	if err = m.SaveSnapshot(SessionState{ActiveIndex: 1}); err != nil {
		t.Fatal(err)
	}
	// Deterministically force rename failure without touching user resources.
	oldPath := m.walPath
	m.walPath = filepath.Join(m.dir, "missing", "session.wal")
	m.appends = 100
	if err = m.SaveSnapshot(SessionState{ActiveIndex: 2}); err == nil {
		t.Fatal("expected compaction failure")
	}
	m.walPath = oldPath
	st, err := m.LoadSnapshot()
	if err != nil || st.ActiveIndex != 1 {
		t.Fatalf("prior snapshot lost: %+v %v", st, err)
	}
}

func TestCorruptLengthAndPartialHeaderRecovery(t *testing.T) {
	for _, tail := range [][]byte{{'G', 'P'}, append(append([]byte{}, walMagic[:]...), []byte{'G', 'P', 'W', '1', 255, 255, 255, 255, 0, 0, 0, 0}...)} {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, "session.wal"), tail, 0600); err != nil {
			t.Fatal(err)
		}
		m, err := NewManager(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err = m.SaveSnapshot(SessionState{ActiveIndex: 4}); err != nil {
			t.Fatal(err)
		}
		st, err := m.LoadSnapshot()
		m.Close()
		if err != nil || st.ActiveIndex != 4 {
			t.Fatalf("recovered %+v %v", st, err)
		}
	}
}

func TestAuditLegacyDirectoryPrivate(t *testing.T) {
	dir := t.TempDir()
	if err := os.Chmod(dir, 0755); err != nil {
		t.Fatal(err)
	}
	m, err := NewManager(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer m.Close()
	st, err := os.Stat(dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0700 {
		t.Fatalf("legacy directory mode %o", st.Mode().Perm())
	}
}

func TestAuditFailedRollbackDisablesWAL(t *testing.T) {
	m, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// A read-only descriptor deterministically fails both append and rollback.
	path := m.walPath
	m.walFile.Close()
	m.walFile, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := m.SaveSnapshot(SessionState{ActiveIndex: 1}); err == nil {
		t.Fatal("write failure hidden")
	}
	if m.walFile != nil {
		t.Fatal("failed rollback left WAL writable")
	}
	if err := m.SaveSnapshot(SessionState{ActiveIndex: 2}); err == nil {
		t.Fatal("disabled WAL reported success")
	}
}
