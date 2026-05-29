package update

import (
	"fmt"
	"os"
	"path/filepath"
)

// acquireUpdateLock creates an O_EXCL lock file in dir for the duration of an
// update and returns a release func. If the lock already exists, another update
// is in progress and an error is returned. The Windows rename-aside swap is not
// idempotent under concurrency, so this serialization is correctness, not
// gold-plating. Cross-platform (plain O_EXCL file, no flock).
func acquireUpdateLock(dir string) (func(), error) {
	lockPath := filepath.Join(dir, ".typeburn-update.lock")
	f, err := os.OpenFile(lockPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		if os.IsExist(err) {
			return nil, fmt.Errorf("update: another update is already in progress (remove %s if stale)", lockPath)
		}
		return nil, fmt.Errorf("update: acquire lock: %w", err)
	}
	_ = f.Close()
	return func() { _ = os.Remove(lockPath) }, nil
}
