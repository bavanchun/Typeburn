//go:build !windows

package update

import (
	"fmt"
	"os"
)

// replaceBinary atomically replaces target with newBin. newBin must already
// reside in target's directory (the extractor writes it there), so the rename is
// same-filesystem and atomic — no cross-device window. On unix, renaming over a
// running executable is safe: the live process keeps the old inode.
//
// The target is lstat-refused if it is a symlink or non-regular file (defends a
// symlinked-target swap). The new binary inherits the target's permission bits
// (or 0o755), with the execute bits forced on. Ownership is never changed.
func replaceBinary(target, newBin string) error {
	mode := os.FileMode(0o755)
	if info, err := os.Lstat(target); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("update: refusing to replace non-regular target %q", target)
		}
		mode = info.Mode().Perm() | 0o111
	}
	if err := os.Chmod(newBin, mode); err != nil {
		return fmt.Errorf("update: set mode on new binary: %w", err)
	}
	if err := os.Rename(newBin, target); err != nil {
		return fmt.Errorf("update: install new binary: %w", err)
	}
	return nil
}

// restoreInterruptedUpdate is a no-op on unix: the single atomic rename leaves
// no half-applied state to recover. It exists so callers can invoke it
// unconditionally across platforms (see replace_windows.go).
func restoreInterruptedUpdate(string) {}
