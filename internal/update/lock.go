package update

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// lockFileName is the O_EXCL sentinel held for the duration of an update.
const lockFileName = ".typeburn-update.lock"

const (
	// lockMaxAge bounds how long a lock is honoured even when the process that
	// took it is still running. A whole update is bounded by a 90s HTTP client
	// timeout, so a lock this old means the owner is wedged or its PID has been
	// recycled onto an unrelated process. Generous on purpose: refusing an
	// update costs the user one retry, whereas two updaters renaming over the
	// same binary can destroy it.
	lockMaxAge = 15 * time.Minute

	// lockWriteGrace covers the window between creating the lock file and
	// writing the owner record into it. A lock carrying no readable record that
	// is younger than this may belong to a process still mid-write, so it is
	// honoured; an older one is debris from a killed run (or from a version that
	// recorded no owner at all) and may be reclaimed.
	lockWriteGrace = 5 * time.Second
)

// acquireUpdateLock takes the update lock in dir and returns a release func.
// The Windows rename-aside swap is not idempotent under concurrency, so this
// serialization is correctness, not gold-plating. Cross-platform (plain O_EXCL
// file, no flock).
//
// A lock whose owner is gone is reclaimed rather than reported, because an
// update killed by a signal or a power cut leaves one behind and no later run
// would ever clear it. A lock that a live process still holds is honoured.
func acquireUpdateLock(dir string) (func(), error) {
	lockPath := filepath.Join(dir, lockFileName)

	err := createLockFile(lockPath)
	if os.IsExist(err) && reclaimStaleLock(lockPath) {
		err = createLockFile(lockPath)
	}
	switch {
	case os.IsExist(err):
		return nil, fmt.Errorf("update: another update is already in progress (remove %s if stale)", lockPath)
	case err != nil:
		return nil, fmt.Errorf("update: acquire lock: %w", err)
	}

	// Holding the lock makes every other updater temp file in dir unambiguously
	// debris from a run that never unwound. Clear it before this update creates
	// its own, because each of those files is an O_EXCL target too and would
	// otherwise block the run just as permanently as the lock did.
	sweepUpdateArtifacts(dir)
	return func() { _ = os.Remove(lockPath) }, nil
}

// createLockFile creates the lock exclusively and records the owning PID plus
// the time it was taken, so a later run can tell an abandoned lock from a live
// one. The record is written after the exclusive create — the create is what
// provides mutual exclusion — so a reader can briefly observe an empty file;
// lockWriteGrace covers that window.
func createLockFile(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, writeErr := fmt.Fprintf(f, "pid %d\ntaken %d\n", os.Getpid(), time.Now().Unix())
	closeErr := f.Close()
	if writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		_ = os.Remove(path)
		return writeErr
	}
	return nil
}

// lockRecord is the owner information stored inside a lock file.
type lockRecord struct {
	pid   int
	taken time.Time
}

// readLockRecord parses the owner record. ok is false when the file is missing,
// unreadable, or carries no owner — which is what a lock from an older version,
// or one caught mid-write, looks like.
func readLockRecord(path string) (rec lockRecord, ok bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return lockRecord{}, false
	}
	var gotPID, gotTaken bool
	for line := range strings.SplitSeq(string(data), "\n") {
		key, val, found := strings.Cut(strings.TrimSpace(line), " ")
		if !found {
			continue
		}
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "pid":
			rec.pid, gotPID = int(n), true
		case "taken":
			rec.taken, gotTaken = time.Unix(n, 0), true
		}
	}
	return rec, gotPID && gotTaken
}

// reclaimStaleLock removes an existing lock when it cannot belong to a running
// update, and reports whether it did. Deliberately conservative: when in doubt
// the lock is left alone and the caller refuses the update.
func reclaimStaleLock(path string) bool {
	rec, ok := readLockRecord(path)
	if !ok {
		return olderThan(path, lockWriteGrace) && os.Remove(path) == nil
	}
	if processAlive(rec.pid) && time.Since(rec.taken) < lockMaxAge {
		return false
	}
	return os.Remove(path) == nil
}

// olderThan reports whether path was last modified more than d ago.
func olderThan(path string, d time.Duration) bool {
	info, err := os.Stat(path)
	return err == nil && time.Since(info.ModTime()) > d
}

// sweepUpdateArtifacts removes the temp files an update creates in dir: the
// checksums list, the release archive, and the extracted binary. Only names the
// updater itself writes are matched, so nothing else in the install directory
// is touched. Callers must already hold the lock.
func sweepUpdateArtifacts(dir string) {
	names := []string{"checksums.txt", "typeburn.new", "typeburn.exe.new"}
	for _, pattern := range []string{"typeburn_*.tar.gz", "typeburn_*.zip"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		for _, m := range matches {
			names = append(names, filepath.Base(m))
		}
	}
	for _, n := range names {
		_ = os.Remove(filepath.Join(dir, n))
	}
}
