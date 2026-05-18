// Package storage handles reading and writing user settings to disk.
// It uses atomic writes (write-to-temp + fsync + rename) to prevent
// partial-file corruption on crash.
package storage

import (
	"fmt"
	"os"
)

// atomicWrite writes data to path atomically: it writes to a temp file
// alongside the target, fsyncs, then renames over the target.
// The target file is created with mode 0600; its parent directory must exist.
func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("storage: create temp file: %w", err)
	}

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

	return nil
}
