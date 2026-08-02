//go:build windows

package update

import (
	"fmt"
	"os"
)

// replaceBinary swaps a running Windows executable. A running .exe cannot be
// overwritten in place, so the running binary is moved aside, the new one is
// moved in, and the old copy is best-effort removed. That removal can fail
// while the process still holds the file open; the leftover is cleared by the
// first os.Remove below on the next update, and nothing clears it before then.
//
// There is deliberately no startup recovery pass for a crash between the two
// renames. Such a crash leaves target missing, and a recovery that only runs
// when typeburn is launched from target can never execute in that state — the
// user has to reinstall. Do not add one back without that changing.
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
