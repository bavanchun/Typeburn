//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package storage

import (
	"errors"
	"os"
)

// errNoFileLocking reports that this platform offers no advisory file lock.
// Writers degrade to unsynchronised appends and say so, rather than failing.
var errNoFileLocking = errors.New("advisory file locking not supported on this platform")

// tryLockFile always reports the platform cannot lock, which the caller turns
// into a degraded write plus a notice.
func tryLockFile(_ *os.File) (bool, error) {
	return false, errNoFileLocking
}

// unlockFile is a no-op: no lock was ever taken.
func unlockFile(_ *os.File) error {
	return nil
}
