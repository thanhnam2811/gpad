package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

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

// Manager coordinates session saving and loading.
type Manager struct {
	mu      sync.Mutex
	dir     string
	walPath string
}

// NewManager initializes a session manager storing state in dir.
func NewManager(dir string) (*Manager, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Manager{
		dir:     dir,
		walPath: filepath.Join(dir, "session.wal"),
	}, nil
}

// SaveSnapshot saves the current session state to disk.
func (m *Manager) SaveSnapshot(state SessionState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	tmp := filepath.Join(m.dir, "session.tmp")
	dest := filepath.Join(m.dir, "session.json")
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}

// LoadSnapshot restores the latest session state from disk.
func (m *Manager) LoadSnapshot() (*SessionState, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	dest := filepath.Join(m.dir, "session.json")
	data, err := os.ReadFile(dest)
	if err != nil {
		if os.IsNotExist(err) {
			return &SessionState{ActiveIndex: 0, Tabs: nil}, nil
		}
		return nil, err
	}

	var state SessionState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}
