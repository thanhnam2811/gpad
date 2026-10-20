package session

import (
	"os"
	"path/filepath"
	"testing"
)

func FuzzWALCorruptTail(f *testing.F) {
	f.Add([]byte("GPW1\xff\xff\xff\xff\x00\x00\x00\x00"))
	f.Add([]byte("partial"))
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, tail []byte) {
		if len(tail) > 4096 {
			t.Skip()
		}
		dir := t.TempDir()
		m, err := NewManager(dir)
		if err != nil {
			t.Fatal(err)
		}
		if err := m.SaveSnapshot(SessionState{ActiveIndex: 7}); err != nil {
			t.Fatal(err)
		}
		m.Close()
		file, err := os.OpenFile(filepath.Join(dir, "session.wal"), os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := file.Write(tail); err != nil {
			t.Fatal(err)
		}
		file.Close()
		m, err = NewManager(dir)
		if err != nil {
			t.Fatal(err)
		}
		defer m.Close()
		// Arbitrary tails may encode a valid frame; every subsequent acknowledged
		// save must remain recoverable after invalid-tail repair.
		if err := m.SaveSnapshot(SessionState{ActiveIndex: 9}); err != nil {
			t.Fatal(err)
		}
		state, err := m.LoadSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		if state.ActiveIndex != 9 {
			t.Fatalf("new checkpoint lost: %+v", state)
		}
	})
}
