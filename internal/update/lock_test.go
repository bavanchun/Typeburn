package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestAcquireUpdateLock(t *testing.T) {
	dir := t.TempDir()

	release, err := acquireUpdateLock(dir)
	if err != nil {
		t.Fatalf("first acquire failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".typeburn-update.lock")); err != nil {
		t.Fatalf("lock file not created: %v", err)
	}

	// A second acquire while held must fail.
	if _, err := acquireUpdateLock(dir); err == nil {
		t.Error("second acquire should fail while lock held")
	}

	release()

	// After release, the lock file is gone and acquire succeeds again.
	if _, err := os.Stat(filepath.Join(dir, ".typeburn-update.lock")); !os.IsNotExist(err) {
		t.Errorf("lock file should be removed after release, stat err = %v", err)
	}
	release2, err := acquireUpdateLock(dir)
	if err != nil {
		t.Fatalf("re-acquire after release failed: %v", err)
	}
	release2()
}

// The lock has to name its owner, otherwise a later run cannot tell an
// abandoned lock from one a running update still needs.
func TestAcquireUpdateLock_RecordsItsOwner(t *testing.T) {
	dir := t.TempDir()
	release, err := acquireUpdateLock(dir)
	if err != nil {
		t.Fatalf("acquire: %v", err)
	}
	defer release()

	rec, ok := readLockRecord(filepath.Join(dir, lockFileName))
	if !ok {
		t.Fatal("lock carries no owner record")
	}
	if rec.pid != os.Getpid() {
		t.Errorf("lock pid = %d, want %d", rec.pid, os.Getpid())
	}
	if age := time.Since(rec.taken); age < 0 || age > time.Minute {
		t.Errorf("lock timestamp is %s old, want roughly now", age)
	}
}

// liveHolderPID starts a long-running child and returns its PID. Two updaters
// renaming over the same binary can destroy it, so a lock a real process still
// holds must be honoured however inconvenient that is.
func liveHolderPID(t *testing.T) int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("helper relies on a POSIX sleep binary")
	}
	c := exec.Command("/bin/sleep", "60")
	if err := c.Start(); err != nil {
		t.Fatalf("start live holder: %v", err)
	}
	t.Cleanup(func() {
		_ = c.Process.Kill()
		_ = c.Wait()
	})
	return c.Process.Pid
}

func TestAcquireUpdateLock_RefusesWhileTheOwnerIsAlive(t *testing.T) {
	dir := t.TempDir()
	writeLockRecord(t, filepath.Join(dir, lockFileName), liveHolderPID(t), time.Now())

	release, err := acquireUpdateLock(dir)
	if err == nil {
		release()
		t.Fatal("acquired a lock that a running process still holds")
	}
	if !strings.Contains(err.Error(), "already in progress") {
		t.Errorf("error = %v, want the in-progress refusal", err)
	}
	if _, statErr := os.Stat(filepath.Join(dir, lockFileName)); statErr != nil {
		t.Errorf("a live holder's lock was deleted: %v", statErr)
	}
}

// A lock older than the whole update can possibly take means the owner is
// wedged, or its PID has been recycled onto an unrelated process. Liveness
// alone would leave the user stuck forever in both cases.
func TestAcquireUpdateLock_ReclaimsWhenTheLockOutlivesAnyUpdate(t *testing.T) {
	dir := t.TempDir()
	writeLockRecord(t, filepath.Join(dir, lockFileName), liveHolderPID(t), time.Now().Add(-lockMaxAge-time.Minute))

	release, err := acquireUpdateLock(dir)
	if err != nil {
		t.Fatalf("a lock older than %s must be reclaimable: %v", lockMaxAge, err)
	}
	release()
}

// A lock with no owner record is either mid-write by a process that has just
// created it, or debris. Within the write grace it is honoured.
func TestAcquireUpdateLock_HonoursAnOwnerlessLockWithinTheWriteGrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	release, err := acquireUpdateLock(dir)
	if err == nil {
		release()
		t.Fatal("acquired a lock that another process may still be writing")
	}
}

// Past the write grace an ownerless lock can only be debris — including one
// left by a version that recorded no owner at all, which is exactly the state a
// user bricked by an earlier interrupt is stuck in.
func TestAcquireUpdateLock_ReclaimsAnOwnerlessLockAfterTheWriteGrace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, lockFileName)
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	aged := time.Now().Add(-lockWriteGrace - time.Minute)
	if err := os.Chtimes(path, aged, aged); err != nil {
		t.Fatal(err)
	}

	release, err := acquireUpdateLock(dir)
	if err != nil {
		t.Fatalf("an ownerless lock older than %s must be reclaimable: %v", lockWriteGrace, err)
	}
	release()
}

// The sweep runs in the user's install directory, so it must match only the
// names the updater itself writes.
func TestSweepUpdateArtifacts_TouchesOnlyUpdaterFiles(t *testing.T) {
	dir := t.TempDir()
	mine := []string{"checksums.txt", "typeburn.new", "typeburn.exe.new", "typeburn_9.9.9_linux_amd64.tar.gz", "typeburn_9.9.9_windows_amd64.zip"}
	theirs := []string{"typeburn", "typeburn.exe", "notes.txt", "typeburn-backup.tar.gz", "mytypeburn_1.0.0_linux_amd64.tar.gz"}
	for _, n := range append(append([]string{}, mine...), theirs...) {
		if err := os.WriteFile(filepath.Join(dir, n), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	sweepUpdateArtifacts(dir)

	for _, n := range mine {
		if _, err := os.Stat(filepath.Join(dir, n)); !os.IsNotExist(err) {
			t.Errorf("%s survived the sweep and would block the next update", n)
		}
	}
	for _, n := range theirs {
		if _, err := os.Stat(filepath.Join(dir, n)); err != nil {
			t.Errorf("%s was deleted but is not an updater temp file", n)
		}
	}
}
