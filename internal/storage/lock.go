package storage

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// lockTimeout bounds how long a writer waits for the history lock. Losing a
// result is worse than writing one without the lock held, so acquisition is
// always bounded: the caller degrades and reports it rather than blocking.
const lockTimeout = 2 * time.Second

// lockPollInterval is how often a waiting writer retries the non-blocking lock.
const lockPollInterval = 5 * time.Millisecond

// errLockUnavailable reports that no advisory lock could be taken — the wait
// timed out, the platform has no file locking, or the filesystem (NFS, some
// container overlays) does not implement it. It is never fatal: the caller
// continues without the lock and surfaces a notice.
var errLockUnavailable = errors.New("storage: history lock unavailable")

// fileLock holds an advisory exclusive lock on a lock file. The lock lives on
// a file that is never renamed, because a rename replaces the inode and would
// silently drop exclusion between a writer holding the old file and one that
// opened the new one.
type fileLock struct{ f *os.File }

// historyLockPath returns the lock file guarding writes to historyFile.
func historyLockPath(historyFile string) string {
	return filepath.Join(filepath.Dir(historyFile), "history.lock")
}

// acquireHistoryLock takes the exclusive history lock, retrying until timeout
// elapses. It always attempts at least once. Every failure mode is wrapped in
// errLockUnavailable so callers have a single condition to degrade on.
func acquireHistoryLock(historyFile string, timeout time.Duration) (*fileLock, error) {
	path := historyLockPath(historyFile)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, fmt.Errorf("%w: open lock file: %v", errLockUnavailable, err)
	}

	deadline := time.Now().Add(timeout)
	for {
		locked, err := tryLockFile(f)
		if err != nil {
			_ = f.Close()
			return nil, fmt.Errorf("%w: %v", errLockUnavailable, err)
		}
		if locked {
			return &fileLock{f: f}, nil
		}
		if !time.Now().Before(deadline) {
			_ = f.Close()
			return nil, fmt.Errorf("%w: still held after %s", errLockUnavailable, timeout)
		}
		time.Sleep(lockPollInterval)
	}
}

// release drops the lock. Closing the file releases it even if the explicit
// unlock fails, so a writer can never leave the lock stuck behind.
func (l *fileLock) release() {
	_ = unlockFile(l.f)
	_ = l.f.Close()
}
