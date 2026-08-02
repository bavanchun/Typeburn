package storage

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

// lockRegionLen is the byte range locked in the lock file. The file has no
// contents; one byte is enough to make the whole file the exclusion token.
const lockRegionLen = 1

// tryLockFile takes a non-blocking exclusive lock on the first byte of f.
// LOCKFILE_FAIL_IMMEDIATELY keeps the call from blocking, so the caller's
// timeout is the only thing that governs how long a writer waits. Windows
// releases the lock when the handle closes, including on process death.
func tryLockFile(f *os.File) (bool, error) {
	flags := uint32(windows.LOCKFILE_EXCLUSIVE_LOCK | windows.LOCKFILE_FAIL_IMMEDIATELY)
	err := windows.LockFileEx(windows.Handle(f.Fd()), flags, 0, lockRegionLen, 0, new(windows.Overlapped))
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, windows.ERROR_LOCK_VIOLATION), errors.Is(err, windows.ERROR_IO_PENDING):
		return false, nil
	default:
		return false, err
	}
}

// unlockFile releases the lock held on f.
func unlockFile(f *os.File) error {
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, lockRegionLen, 0, new(windows.Overlapped))
}
