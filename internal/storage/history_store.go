package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// historyCapMax is the maximum number of records kept in history.json.
// Oldest records beyond this cap are rotated out on each append.
const historyCapMax = 200

// HistoryPath returns the absolute path to the history file:
// $XDG_DATA_HOME/typeburn/history.json (fallback ~/.local/share/typeburn/).
func HistoryPath() (string, error) {
	dir, err := config.DataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "history.json"), nil
}

// LoadHistory reads and unmarshals the history file. On any error
// (missing file, corrupt JSON, I/O failure) it returns an empty slice.
// Unknown JSON fields are silently ignored via json.Unmarshal's default behaviour.
// This function never returns an error and never panics.
func LoadHistory() []Record {
	path, err := HistoryPath()
	if err != nil {
		return nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Missing file is expected on first run.
		return nil
	}

	var records []Record
	if err := json.Unmarshal(data, &records); err != nil {
		// Corrupt or unreadable JSON — return safe empty slice.
		return nil
	}

	return records
}

// AppendHistory loads the current history, appends r, caps to historyCapMax
// (keeping the newest 200, dropping the oldest), then atomically writes the
// result. It returns the post-write slice and any write error.
//
// Callers that can display a message should prefer AppendHistoryWithNotice:
// this form discards conditions the user would want to know about.
func AppendHistory(r Record) ([]Record, error) {
	records, _, err := appendHistory(r, lockTimeout)
	return records, err
}

// AppendHistoryWithNotice is AppendHistory plus a report of any non-fatal
// condition met on the way — an unreadable history file set aside, or a write
// that had to proceed without the cross-process lock. A zero Notice means the
// write was clean. The record is written in every case where err is nil.
func AppendHistoryWithNotice(r Record) ([]Record, Notice, error) {
	return appendHistory(r, lockTimeout)
}

// appendHistory performs the read-modify-write under an advisory lock so two
// instances cannot each append to the same stale snapshot. The lock is a best
// effort with a bounded wait: saving the result always outranks synchronising
// it, so a lock that cannot be taken degrades to an unsynchronised write plus
// a notice instead of failing or blocking.
//
// Sort order: oldest first in the file (ascending Time); the caller or UI
// reverses for display. Cap rotation drops the front (oldest) entries.
func appendHistory(r Record, timeout time.Duration) ([]Record, Notice, error) {
	path, err := HistoryPath()
	if err != nil {
		return nil, Notice{}, err
	}

	// Ensure parent directory exists before locking or writing.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, Notice{}, err
	}

	var notice Notice
	if lock, lockErr := acquireHistoryLock(path, timeout); lockErr != nil {
		notice = lockNotice()
	} else {
		defer lock.release()
	}

	records, loadNotice := loadForAppend(path, time.Now())
	// A quarantined history file is the more actionable of the two conditions,
	// so it wins when a degraded write also hit an unreadable file.
	if !loadNotice.IsZero() {
		notice = loadNotice
	}
	records = append(records, r)

	// Stable sort ascending by time so the cap drops the oldest entries and
	// records sharing a timestamp keep the order they were appended in.
	sort.SliceStable(records, func(i, j int) bool {
		return records[i].Time.Before(records[j].Time)
	})

	// Cap: keep only the newest historyCapMax entries.
	if len(records) > historyCapMax {
		records = records[len(records)-historyCapMax:]
	}

	data, err := json.Marshal(records)
	if err != nil {
		return records, notice, err
	}

	if err := atomicWrite(path, data); err != nil {
		return records, notice, err
	}

	return records, notice, nil
}

// loadForAppend reads the history that the next write will be based on. Unlike
// LoadHistory it cannot simply discard an unreadable file: the write that
// follows would replace those records with a single new one. An existing file
// that cannot be read or parsed is renamed aside first, so the bytes remain
// recoverable and the returned notice tells the user where they went.
func loadForAppend(path string, now time.Time) ([]Record, Notice) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		// Missing file is expected on first run.
		return nil, Notice{}
	}
	if err == nil {
		// An empty file holds nothing worth preserving — a crash between
		// creating and filling it, not corruption. Treat it as no history.
		if len(bytes.TrimSpace(data)) == 0 {
			return nil, Notice{}
		}
		var records []Record
		if json.Unmarshal(data, &records) == nil {
			return records, Notice{}
		}
	}
	return nil, quarantineHistory(path, now)
}
