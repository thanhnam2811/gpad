package fileio

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// FilePayload encapsulates the ingested file bytes and its cleanup hook.
type FilePayload struct {
	Data   []byte
	unmap  func() error
	once   sync.Once
	isMmap bool
	err    error
}

// Close safely unmaps or frees memory. Idempotent via sync.Once.
func (f *FilePayload) Close() error {
	if f == nil {
		return nil
	}
	f.once.Do(func() {
		if f.unmap != nil {
			f.err = f.unmap()
		}
	})
	return f.err
}

// IsMmap reports whether payload is backed by memory-mapped pages.
func (f *FilePayload) IsMmap() bool {
	if f == nil {
		return false
	}
	return f.isMmap
}

// LoadFile ingests file using high-performance zero-copy syscall.Mmap on Linux.
// For zero-byte files, it returns an empty slice safely without calling mmap.
func LoadFile(path string) (*FilePayload, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return nil, fmt.Errorf("fileio: %s is a directory", path)
	}

	size := int(stat.Size())
	if size == 0 {
		return &FilePayload{
			Data:   []byte{},
			isMmap: false,
		}, nil
	}

	// Linux MAP_SHARED with PROT_READ for zero-copy kernel page mapping
	data, err := syscall.Mmap(int(file.Fd()), 0, size, syscall.PROT_READ, syscall.MAP_SHARED)
	if err != nil {
		// Fallback to standard os.ReadFile if mmap fails (e.g. pseudo-fs /proc or /sys)
		fallbackData, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, fmt.Errorf("fileio mmap failed (%v) and fallback failed: %w", err, rerr)
		}
		return &FilePayload{
			Data:   fallbackData,
			isMmap: false,
		}, nil
	}

	return &FilePayload{
		Data: data,
		unmap: func() error {
			return syscall.Munmap(data)
		},
		isMmap: true,
	}, nil
}

// SaveFile atomically writes data using a temporary file and atomic os.Rename.
func SaveFile(path string, data []byte) error {
	// Follow existing symlinks so saving edits their target, as opening does.
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		path = resolved
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	} else if st, lerr := os.Lstat(path); lerr == nil && st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("fileio: dangling symlink: %w", err)
	}
	mode := os.FileMode(0600)
	if st, err := os.Stat(path); err == nil {
		if !st.Mode().IsRegular() {
			return fmt.Errorf("fileio: target is not a regular file")
		}
		mode = st.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "gpad-save-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)
	if err := tmpFile.Chmod(mode); err != nil {
		tmpFile.Close()
		return err
	}

	n, err := tmpFile.Write(data)
	if err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if n != len(data) {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("short write: wrote %d of %d bytes", n, len(data))
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		_ = os.Remove(tmpPath)
		return err
	}

	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	// Atomic replace
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
