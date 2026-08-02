package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// TestAppendHistory_StaleTempFileDoesNotBlockSaves proves that a leftover
// unwritable temp file cannot lock the user out of saving results.
func TestAppendHistory_StaleTempFileDoesNotBlockSaves(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root ignores file permissions; the stale temp file cannot be made unwritable")
	}
	withTempDataHome(t)
	path, err := HistoryPath()
	if err != nil {
		t.Fatalf("HistoryPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path+".tmp", []byte("stale"), 0400); err != nil {
		t.Fatalf("WriteFile stale temp: %v", err)
	}

	if _, err := AppendHistory(makeRecord(0, 77)); err != nil {
		t.Fatalf("AppendHistory blocked by stale temp file: %v", err)
	}
	if got := LoadHistory(); len(got) != 1 {
		t.Errorf("want 1 persisted record, got %d", len(got))
	}
}

// TestAppendHistory_EqualTimestampsKeepInsertionOrder proves records sharing a
// timestamp keep their relative order when a record arrives out of order — a
// test finished while the clock had stepped back, so the new record is not the
// newest. Sorting has to move things in that case, and it must move only the
// new record, not shuffle the equal-timestamp records already stored.
func TestAppendHistory_EqualTimestampsKeepInsertionOrder(t *testing.T) {
	withTempDataHome(t)
	const perGroup, groups = 12, 3
	for g := range groups {
		for i := range perGroup {
			if _, err := AppendHistory(makeRecord(g, g*perGroup+i)); err != nil {
				t.Fatalf("AppendHistory: %v", err)
			}
		}
	}
	before := LoadHistory()
	if len(before) != groups*perGroup {
		t.Fatalf("setup wrote %d records, want %d", len(before), groups*perGroup)
	}

	const outOfOrderWPM = 999
	if _, err := AppendHistory(makeRecord(0, outOfOrderWPM)); err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}

	after := LoadHistory()
	if len(after) != len(before)+1 {
		t.Fatalf("want %d records, got %d", len(before)+1, len(after))
	}
	kept := make([]int, 0, len(before))
	for _, rec := range after {
		if rec.WPM != outOfOrderWPM {
			kept = append(kept, rec.WPM)
		}
	}
	for i := range before {
		if kept[i] != before[i].WPM {
			t.Fatalf("append reordered stored records at index %d: was %d, now %d",
				i, before[i].WPM, kept[i])
		}
	}
}
