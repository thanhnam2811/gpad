package session

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Magic bytes identifying Gopad++ WAL file
var walMagic = [4]byte{'G', 'P', 'W', '1'}

// TabSession holds state for a single tab.
type TabSession struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Path       string `json:"path"`
	Content    string `json:"content,omitempty"`
	CursorLine int    `json:"cursor_line"`
	CursorCol  int    `json:"cursor_col"`
	IsDirty    bool   `json:"is_dirty"`
}

// SessionState holds full workspace snapshot.
type SessionState struct {
	ActiveIndex int          `json:"active_index"`
	Tabs        []TabSession `json:"tabs"`
}

// Manager coordinates session saving with instant WAL append and automatic log compaction.
type Manager struct {
	mu          sync.Mutex
	dir         string
	walPath     string
	walFile     *os.File
	latestState *SessionState
	appends     int
}

// NewManager initializes a session manager storing state in dir.
func NewManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	walPath := filepath.Join(dir, "session.wal")
	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	// Write magic header if new file
	info, err := f.Stat()
	if err == nil && info.Size() == 0 {
		if _, werr := f.Write(walMagic[:]); werr != nil {
			_ = f.Close()
			return nil, werr
		}
	}

	m := &Manager{
		dir:     dir,
		walPath: walPath,
		walFile: f,
	}

	return m, nil
}

// SaveSnapshot writes the session snapshot directly to the WAL file with bounded size.
func (m *Manager) SaveSnapshot(state SessionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.latestState = &state
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	m.appends++
	// Compact WAL when it exceeds 1MB or after 100 appends to prevent disk bloat
	if m.walFile != nil {
		if info, err := m.walFile.Stat(); err == nil && (info.Size() > 1024*1024 || m.appends > 100) {
			return m.compactWAL(data)
		}
	}

	return m.writeWALRecordLocked(data)
}

// compactWAL compacts the WAL to a single latest checkpoint record.
func (m *Manager) compactWAL(latestData []byte) error {
	tmpPath := m.walPath + ".compact.tmp"
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0o644)
	if err != nil {
		return m.writeWALRecordLocked(latestData)
	}

	// Write magic
	if _, err := tmpFile.Write(walMagic[:]); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return m.writeWALRecordLocked(latestData)
	}

	// Write latest frame
	header := make([]byte, 12)
	copy(header[0:4], walMagic[:])
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(latestData)))
	checksum := crc32.ChecksumIEEE(latestData)
	binary.LittleEndian.PutUint32(header[8:12], checksum)

	if _, err := tmpFile.Write(header); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return m.writeWALRecordLocked(latestData)
	}
	if _, err := tmpFile.Write(latestData); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return m.writeWALRecordLocked(latestData)
	}

	_ = tmpFile.Sync()
	_ = tmpFile.Close()

	if m.walFile != nil {
		_ = m.walFile.Close()
	}

	if err := os.Rename(tmpPath, m.walPath); err != nil {
		_ = os.Remove(tmpPath)
	}

	// Re-open compacted file for appending
	reopened, err := os.OpenFile(m.walPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	m.walFile = reopened
	m.appends = 1
	return nil
}

// writeWALRecordLocked writes a binary frame: [Magic 4B][Length 4B][CRC32 4B][Payload NB].
// Caller must hold m.mu.
func (m *Manager) writeWALRecordLocked(data []byte) error {
	if m.walFile == nil {
		return fmt.Errorf("session wal closed")
	}

	header := make([]byte, 12)
	copy(header[0:4], walMagic[:])
	binary.LittleEndian.PutUint32(header[4:8], uint32(len(data)))
	checksum := crc32.ChecksumIEEE(data)
	binary.LittleEndian.PutUint32(header[8:12], checksum)

	if _, err := m.walFile.Write(header); err != nil {
		return err
	}
	if _, err := m.walFile.Write(data); err != nil {
		return err
	}
	return nil
}

// Close gracefully closes WAL.
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.walFile != nil {
		err := m.walFile.Close()
		m.walFile = nil
		return err
	}
	return nil
}

// LoadSnapshot replays the WAL file, validates CRC32 on every record, and recovers the latest valid state.
func (m *Manager) LoadSnapshot() (*SessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	f, err := os.Open(m.walPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionState{ActiveIndex: 0, Tabs: nil}, nil
		}
		return nil, err
	}
	defer f.Close()

	// Check magic
	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return &SessionState{ActiveIndex: 0, Tabs: nil}, nil
		}
		return nil, err
	}
	if !bytes.Equal(magic[:], walMagic[:]) {
		return nil, fmt.Errorf("invalid wal magic: %v", magic)
	}

	var latestState *SessionState
	headerBuf := make([]byte, 12)

	for {
		_, err := io.ReadFull(f, headerBuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			break
		}

		if !bytes.Equal(headerBuf[0:4], walMagic[:]) {
			// Corrupted frame header; stop replay
			break
		}

		length := binary.LittleEndian.Uint32(headerBuf[4:8])
		expectedCrc := binary.LittleEndian.Uint32(headerBuf[8:12])

		payload := make([]byte, length)
		if _, err := io.ReadFull(f, payload); err != nil {
			// Incomplete record; ignore partial tail
			break
		}

		if crc32.ChecksumIEEE(payload) != expectedCrc {
			// Checksum mismatch; corrupted frame, ignore and stop
			break
		}

		var st SessionState
		if err := json.Unmarshal(payload, &st); err == nil {
			latestState = &st
		}
	}

	if latestState == nil {
		return &SessionState{ActiveIndex: 0, Tabs: nil}, nil
	}
	return latestState, nil
}
