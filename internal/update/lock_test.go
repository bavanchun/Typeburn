package update

import (
	"os"
	"path/filepath"
	"testing"
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
