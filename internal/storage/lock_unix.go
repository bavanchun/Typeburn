//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd

package storage

import (
	"errors"
	"os"
	"syscall"
)

// tryLockFile takes a non-blocking exclusive flock. It returns (true, nil) on
// success, (false, nil) when another open file description holds the lock, and
// (false, err) when the filesystem cannot lock at all — some network and
// container filesystems reject flock outright, and that must degrade rather
// than fail the write. The kernel releases an flock when the process dies, so
// a crashed writer cannot leave the lock stuck.
func tryLockFile(f *os.File) (bool, error) {
	err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, syscall.EWOULDBLOCK):
		return false, nil
	default:
		return false, err
	}
}

// unlockFile releases the flock held on f.
func unlockFile(f *os.File) error {
	return syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
}
