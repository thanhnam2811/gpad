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
