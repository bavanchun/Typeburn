package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// NoticeKind classifies a storage condition that did not stop the write but
// that the user must be told about.
type NoticeKind int

const (
	// NoticeNone is the zero value: nothing to report.
	NoticeNone NoticeKind = iota

	// NoticeQuarantined means the existing history file could not be parsed
	// and was renamed aside instead of being overwritten. Path holds where it
	// went, so the user can recover the records by hand.
	NoticeQuarantined

	// NoticeUnsynchronised means the write went ahead without the cross-process
	// lock. The record was saved; a second instance writing at the same instant
	// could still overwrite it.
	NoticeUnsynchronised
)

// Notice is a UI-free report of a non-fatal storage condition. Message is a
// single display-ready line; the UI layer decides how to render it. The zero
// value means the write was clean.
type Notice struct {
	Kind    NoticeKind
	Message string
	// Path is the quarantine file, set only for NoticeQuarantined.
	Path string
}

// IsZero reports whether there is nothing to tell the user.
func (n Notice) IsZero() bool { return n.Kind == NoticeNone }

// quarantineHistory renames an unparseable history file aside under a
// timestamped name so the next write cannot destroy it. The user's only copy
// of their data is never deleted, only moved.
func quarantineHistory(path string, now time.Time) Notice {
	dest := quarantinePath(path, now)
	if err := os.Rename(path, dest); err != nil {
		return Notice{
			Kind:    NoticeQuarantined,
			Message: "History file is unreadable and could not be set aside",
		}
	}
	return Notice{
		Kind:    NoticeQuarantined,
		Path:    dest,
		Message: fmt.Sprintf("History file was unreadable; kept a copy as %s", filepath.Base(dest)),
	}
}

// quarantinePath builds a free ".corrupt-<timestamp>" name beside path. It
// never returns a name that already exists unless the same second has already
// been exhausted, so earlier quarantines survive.
func quarantinePath(path string, now time.Time) string {
	base := path + ".corrupt-" + now.UTC().Format("20060102-150405")
	dest := base
	for i := 1; i < 100; i++ {
		if _, err := os.Lstat(dest); errors.Is(err, fs.ErrNotExist) {
			return dest
		}
		dest = fmt.Sprintf("%s-%d", base, i)
	}
	return dest
}

// lockNotice reports that a write proceeded without the cross-process lock.
// The result is still written; the notice only says it was written
// unsynchronised, so a user seeing it can avoid running two instances at once.
func lockNotice() Notice {
	return Notice{
		Kind:    NoticeUnsynchronised,
		Message: "Couldn't lock the history file; result saved without it",
	}
}
