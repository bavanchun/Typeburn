package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"

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
// Sort order: oldest first in the file (ascending Time); the caller or UI
// reverses for display. Cap rotation drops the front (oldest) entries.
func AppendHistory(r Record) ([]Record, error) {
	records := LoadHistory()
	records = append(records, r)

	// Sort ascending by time so cap drops the oldest entries (front).
	sort.Slice(records, func(i, j int) bool {
		return records[i].Time.Before(records[j].Time)
	})

	// Cap: keep only the newest historyCapMax entries.
	if len(records) > historyCapMax {
		records = records[len(records)-historyCapMax:]
	}

	path, err := HistoryPath()
	if err != nil {
		return records, err
	}

	// Ensure parent directory exists before writing.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return records, err
	}

	data, err := json.Marshal(records)
	if err != nil {
		return records, err
	}

	if err := atomicWrite(path, data); err != nil {
		return records, err
	}

	return records, nil
}
