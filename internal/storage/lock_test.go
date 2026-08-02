package storage

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// holdHistoryLock takes the history lock the way a second instance would and
// releases it when the test ends. Locks are held per open file, so this
// contends with acquireHistoryLock even from the same process.
func holdHistoryLock(t *testing.T, historyFile string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(historyFile), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	f, err := os.OpenFile(historyLockPath(historyFile), os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	locked, err := tryLockFile(f)
	if err != nil || !locked {
		_ = f.Close()
		t.Skipf("this filesystem provides no working file lock (locked=%v, err=%v)", locked, err)
	}
	t.Cleanup(func() {
		_ = unlockFile(f)
		_ = f.Close()
	})
}

// TestAcquireHistoryLock_ExcludesAndReleases proves the lock actually excludes
// a second holder and becomes available again once released.
func TestAcquireHistoryLock_ExcludesAndReleases(t *testing.T) {
	dir := withTempDataHome(t)
	historyFile := filepath.Join(dir, "typeburn", "history.json")
	holdHistoryLock(t, historyFile)

	if lock, err := acquireHistoryLock(historyFile, 20*time.Millisecond); err == nil {
		lock.release()
		t.Fatal("acquired a lock that another holder already had")
	} else if !errors.Is(err, errLockUnavailable) {
		t.Fatalf("want errLockUnavailable, got %v", err)
	}
}

// TestAcquireHistoryLock_ReacquireAfterRelease proves release does not leave
// the lock stuck, which would block every later save.
func TestAcquireHistoryLock_ReacquireAfterRelease(t *testing.T) {
	dir := withTempDataHome(t)
	historyFile := filepath.Join(dir, "typeburn", "history.json")
	if err := os.MkdirAll(filepath.Dir(historyFile), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	first, err := acquireHistoryLock(historyFile, time.Second)
	if err != nil {
		t.Skipf("this filesystem provides no working file lock: %v", err)
	}
	first.release()

	second, err := acquireHistoryLock(historyFile, time.Second)
	if err != nil {
		t.Fatalf("lock stayed held after release: %v", err)
	}
	second.release()
}

// TestAppendHistory_LockUnavailableStillSaves proves the ordering that matters:
// never losing a result outranks never racing. A lock that cannot be taken must
// not hang the save, must not fail it, and must not stay silent about it.
func TestAppendHistory_LockUnavailableStillSaves(t *testing.T) {
	dir := withTempDataHome(t)
	historyFile := filepath.Join(dir, "typeburn", "history.json")
	holdHistoryLock(t, historyFile)

	const timeout = 50 * time.Millisecond
	start := time.Now()
	after, notice, err := appendHistory(makeRecord(0, 84), timeout)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("append must still succeed without the lock: %v", err)
	}
	if len(after) != 1 || after[0].WPM != 84 {
		t.Fatalf("record not written: %+v", after)
	}
	if got := LoadHistory(); len(got) != 1 {
		t.Fatalf("want 1 record on disk, got %d", len(got))
	}
	if notice.Kind != NoticeUnsynchronised {
		t.Errorf("notice kind: want NoticeUnsynchronised, got %v (%q)", notice.Kind, notice.Message)
	}
	if notice.Message == "" {
		t.Error("degraded write must carry a message for the user")
	}
	// Bounded: waiting must end at the timeout, not when the other holder feels
	// like letting go.
	if limit := 20 * timeout; elapsed > limit {
		t.Errorf("append waited %s on an unavailable lock, want under %s", elapsed, limit)
	}
}
