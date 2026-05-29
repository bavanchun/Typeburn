//go:build windows

package update

import (
	"fmt"
	"os"
)

// replaceBinary swaps a running Windows executable. A running .exe cannot be
// overwritten in place, so the running binary is moved aside, the new one is
// moved in, and the old copy is best-effort removed (it may stay locked while
// the process runs — restoreInterruptedUpdate cleans it up on the next launch).
//
// If the second rename fails after the first succeeded, the original is rolled
// back so target is never left missing. newBin must reside in target's
// directory so both renames are same-filesystem.
func replaceBinary(target, newBin string) error {
	old := target + ".old"
	_ = os.Remove(old) // clear any stale aside-copy from a prior run

	if err := os.Rename(target, old); err != nil {
		return fmt.Errorf("update: move running exe aside: %w", err)
	}
	if err := os.Rename(newBin, target); err != nil {
		if rbErr := os.Rename(old, target); rbErr != nil {
			return fmt.Errorf("update: install failed (%v) and rollback failed (%v); restore %q from %q manually",
				err, rbErr, target, old)
		}
		return fmt.Errorf("update: install new exe: %w", err)
	}
	_ = os.Remove(old) // best-effort; may be locked by the running process
	return nil
}

// restoreInterruptedUpdate recovers from a crash between the two renames: if
// target is missing but target+".old" exists, the running exe was moved aside
// but the new one never landed — restore the original. Safe to call at startup.
func restoreInterruptedUpdate(target string) {
	if _, err := os.Stat(target); err == nil {
		return // target present, nothing to recover
	}
	old := target + ".old"
	if _, err := os.Stat(old); err == nil {
		_ = os.Rename(old, target)
	}
}
