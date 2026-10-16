package fileio

import (
	"fmt"
	"os"
)

// FilePayload contains content and whether the payload uses memory mapping.
type FilePayload struct {
	Data    []byte
	UnmapFn func() error
}

// Close releases resources (such as unmapping memory).
func (f *FilePayload) Close() error {
	if f.UnmapFn != nil {
		return f.UnmapFn()
	}
	return nil
}

// LoadFile reads the contents of the given file.
func LoadFile(path string) (*FilePayload, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if info.IsDir() {
		return nil, fmt.Errorf("%s is a directory", path)
	}
	if info.Size() == 0 {
		return &FilePayload{Data: []byte{}}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return &FilePayload{Data: data}, nil
}

// SaveFile atomically writes data to the target file.
func SaveFile(path string, data []byte) error {
	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
