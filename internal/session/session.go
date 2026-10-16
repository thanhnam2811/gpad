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
	"time"
)

// Magic bytes identifying Gopad++ WAL file
var walMagic = [4]byte{'G', 'P', 'W', '1'}

// TabSession holds state for a single tab.
type TabSession struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	CursorLine int    `json:"cursor_line"`
	CursorCol  int    `json:"cursor_col"`
	IsDirty    bool   `json:"is_dirty"`
}

// SessionState holds full workspace snapshot.
type SessionState struct {
	ActiveIndex int          `json:"active_index"`
	Tabs        []TabSession `json:"tabs"`
}

// Manager coordinates session saving with instant WAL append and async disk persistence.
type Manager struct {
	mu           sync.RWMutex
	dir          string
	walPath      string
	walFile      *os.File
	latestState  *SessionState
	saveChan     chan []byte
	stopChan     chan struct{}
	wg           sync.WaitGroup
	asyncEnabled bool
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
		dir:          dir,
		walPath:      walPath,
		walFile:      f,
		saveChan:     make(chan []byte, 16),
		stopChan:     make(chan struct{}),
		asyncEnabled: true,
	}

	m.wg.Add(1)
	go m.backgroundFlusher()

	return m, nil
}

// SaveSnapshot performs zero-latency non-blocking snapshot dispatch.
func (m *Manager) SaveSnapshot(state SessionState) error {
	m.mu.Lock()
	m.latestState = &state
	m.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	// Try non-blocking push to background flusher
	select {
	case m.saveChan <- data:
		return nil
	default:
		// Queue full; write directly to WAL
		return m.writeWALRecord(data)
	}
}

// writeWALRecord writes a binary frame: [Magic 4B][Length 4B][CRC32 4B][Payload NB]
func (m *Manager) writeWALRecord(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

// backgroundFlusher processes incoming snapshot payloads asynchronously.
func (m *Manager) backgroundFlusher() {
	defer m.wg.Done()
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	var pending []byte

	for {
		select {
		case data, ok := <-m.saveChan:
			if !ok {
				if len(pending) > 0 {
					_ = m.writeWALRecord(pending)
				}
				return
			}
			pending = data
		case <-ticker.C:
			if len(pending) > 0 {
				_ = m.writeWALRecord(pending)
				pending = nil
			}
		case <-m.stopChan:
			// Flush remaining
			for {
				select {
				case data := <-m.saveChan:
					pending = data
				default:
					if len(pending) > 0 {
						_ = m.writeWALRecord(pending)
					}
					return
				}
			}
		}
	}
}

// Close gracefully stops the background flusher and closes WAL.
func (m *Manager) Close() error {
	close(m.stopChan)
	m.wg.Wait()

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
			// Incomplete record (e.g. power loss during write); ignore partial tail
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
