package session

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"gpad/internal/fileio"
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
	ShadowPath string `json:"shadow_path,omitempty"`
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
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}

	if err := os.Chmod(dir, 0700); err != nil {
		return nil, err
	}
	walPath := filepath.Join(dir, "session.wal")
	f, err := os.OpenFile(walPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0o600)
	if err != nil {
		return nil, err
	}

	if err := f.Chmod(0600); err != nil {
		f.Close()
		return nil, err
	}

	// Write magic header if new file
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.Size() < 4 {
		if err := f.Truncate(0); err != nil {
			f.Close()
			return nil, err
		}
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

	if _, err := m.loadSnapshotLocked(true); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return nil, err
	}
	d, err := os.Open(dir)
	if err != nil {
		f.Close()
		return nil, err
	}
	err = d.Sync()
	d.Close()
	if err != nil {
		f.Close()
		return nil, err
	}
	return m, nil
}

// Dir returns the directory path managed by this session manager.
func (m *Manager) Dir() string {
	if m == nil {
		return ""
	}
	return m.dir
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
		info, err := m.walFile.Stat()
		if err != nil {
			return err
		}
		if info.Size() > 1024*1024 || m.appends > 100 {
			return m.compactWAL(data)
		}
	}

	return m.writeWALRecordLocked(data)
}

// compactWAL compacts the WAL to a single latest checkpoint record.
func (m *Manager) compactWAL(latestData []byte) error {
	if m.walFile == nil {
		return fmt.Errorf("session wal closed")
	}
	var checkpoint bytes.Buffer
	checkpoint.Write(walMagic[:])
	checkpoint.Write(walFrame(latestData))
	saveErr := fileio.SaveFile(m.walPath, checkpoint.Bytes())
	reopened, err := os.OpenFile(m.walPath, os.O_RDWR|os.O_APPEND, 0600)
	// The pathname has already changed. Never keep writing to the old inode.
	closeErr := m.walFile.Close()
	m.walFile = reopened
	if err != nil {
		return err
	}
	if saveErr != nil {
		return saveErr
	}
	if closeErr != nil {
		return closeErr
	}
	m.appends = 1
	return nil
}

func walFrame(data []byte) []byte {
	frame := make([]byte, 12+len(data))
	copy(frame, walMagic[:])
	binary.LittleEndian.PutUint32(frame[4:8], uint32(len(data)))
	binary.LittleEndian.PutUint32(frame[8:12], crc32.ChecksumIEEE(data))
	copy(frame[12:], data)
	return frame
}

// writeWALRecordLocked writes a binary frame: [Magic 4B][Length 4B][CRC32 4B][Payload NB].
// Caller must hold m.mu.
func (m *Manager) writeWALRecordLocked(data []byte) error {
	if m.walFile == nil {
		return fmt.Errorf("session wal closed")
	}

	start, err := m.walFile.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	frame := walFrame(data)
	n, err := m.walFile.Write(frame)
	if err == nil && n != len(frame) {
		err = io.ErrShortWrite
	}
	if err != nil {
		if rollback := m.walFile.Truncate(start); rollback != nil {
			_ = m.walFile.Close()
			m.walFile = nil
			return fmt.Errorf("append: %v; rollback: %w", err, rollback)
		}
		return err
	}
	return m.walFile.Sync()
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
	return m.loadSnapshotLocked(false)
}

func (m *Manager) loadSnapshotLocked(repair bool) (*SessionState, error) {
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

	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	validEnd := int64(4)
	var latestState *SessionState
	headerBuf := make([]byte, 12)

	for {
		_, err := io.ReadFull(f, headerBuf)
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}

		if !bytes.Equal(headerBuf[0:4], walMagic[:]) {
			// Corrupted frame header; stop replay
			break
		}

		length := binary.LittleEndian.Uint32(headerBuf[4:8])
		expectedCrc := binary.LittleEndian.Uint32(headerBuf[8:12])

		// Reject impossible lengths before allocating attacker-controlled memory.
		if int64(length) > info.Size()-validEnd-12 {
			break
		}
		payload := make([]byte, length)
		if _, err := io.ReadFull(f, payload); err != nil {
			// Incomplete record; ignore partial tail, but preserve genuine I/O errors.
			if err != io.EOF && err != io.ErrUnexpectedEOF {
				return nil, err
			}
			break
		}

		if crc32.ChecksumIEEE(payload) != expectedCrc {
			// Checksum mismatch; corrupted frame, ignore and stop
			break
		}

		var st SessionState
		if err := json.Unmarshal(payload, &st); err == nil {
			latestState = &st
			validEnd += int64(12 + len(payload))
		} else {
			break
		}
	}
	if repair && info.Size() != validEnd {
		if err := m.walFile.Truncate(validEnd); err != nil {
			return nil, err
		}
	}

	if latestState == nil {
		return &SessionState{ActiveIndex: 0, Tabs: nil}, nil
	}
	return latestState, nil
}
