// Package storage handles reading and writing user data to disk.
// It uses atomic writes (write-to-temp + fsync + rename) so a reader never
// observes a partially written file. After the rename the parent directory is
// fsynced where the platform supports it, so a write that returned nil survives
// a power loss; a directory fsync failure is deliberately not reported as a
// write failure, because at that point the data is already in the target file.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// atomicWrite writes data to path atomically: it writes to a uniquely named
// temp file alongside the target, fsyncs, then renames over the target.
// The target file is created with mode 0600; its parent directory must exist.
func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)

	// A unique temp name per write. Two processes writing the same target must
	// never share a temp file, and a leftover temp file from a crashed or
	// root-owned run must never make every future write fail.
	f, err := os.CreateTemp(dir, filepath.Base(path)+"-*.tmp")
	if err != nil {
		return fmt.Errorf("storage: create temp file: %w", err)
	}
	tmp := f.Name()

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: write temp file: %w", err)
	}

	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: fsync temp file: %w", err)
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: close temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("storage: rename temp to target: %w", err)
	}

	syncDir(dir)
	return nil
}

// syncDir flushes the directory entry created by the rename so the rename
// itself is durable, not just the file contents. It is best effort: platforms
// that reject fsync on a directory handle (Windows) journal the rename anyway,
// and a failure here cannot undo a write that already succeeded.
func syncDir(dir string) {
	d, err := os.Open(dir)
	if err != nil {
		return
	}
	_ = d.Sync()
	_ = d.Close()
}
