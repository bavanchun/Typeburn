package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// writeRawHistory puts exact bytes at the history path and returns that path.
func writeRawHistory(t *testing.T, data []byte) string {
	t.Helper()
	path, err := HistoryPath()
	if err != nil {
		t.Fatalf("HistoryPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

// findQuarantine returns the single quarantine file beside path.
func findQuarantine(t *testing.T, path string) string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	var found []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), filepath.Base(path)+".corrupt-") {
			found = append(found, filepath.Join(filepath.Dir(path), e.Name()))
		}
	}
	if len(found) != 1 {
		t.Fatalf("want exactly 1 quarantine file, got %v", found)
	}
	return found[0]
}

// TestAppendHistory_UnreadableFileIsQuarantined proves that a truncated
// history file is moved aside byte for byte instead of being overwritten by
// the next append, so the user can still recover the records by hand.
func TestAppendHistory_UnreadableFileIsQuarantined(t *testing.T) {
	withTempDataHome(t)
	records := make([]Record, 0, 200)
	for i := range 200 {
		records = append(records, makeRecord(i, 60+i))
	}
	valid, err := json.Marshal(records)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	corrupt := valid[:len(valid)-3]
	path := writeRawHistory(t, corrupt)

	after, notice, err := AppendHistoryWithNotice(makeRecord(1000, 99))
	if err != nil {
		t.Fatalf("AppendHistoryWithNotice: %v", err)
	}
	if len(after) != 1 {
		t.Errorf("want the new record alone after quarantine, got %d records", len(after))
	}
	if notice.Kind != NoticeQuarantined {
		t.Fatalf("notice kind: want NoticeQuarantined, got %v (%q)", notice.Kind, notice.Message)
	}
	if notice.Message == "" {
		t.Error("quarantine notice must carry a message for the user")
	}

	quarantined := findQuarantine(t, path)
	if notice.Path != quarantined {
		t.Errorf("notice path %q, quarantine file %q", notice.Path, quarantined)
	}
	saved, err := os.ReadFile(quarantined)
	if err != nil {
		t.Fatalf("read quarantine file: %v", err)
	}
	if string(saved) != string(corrupt) {
		t.Errorf("quarantined bytes differ from the original: %d vs %d bytes", len(saved), len(corrupt))
	}
}

// TestAppendHistory_EmptyFileIsNotQuarantined proves an empty history file is
// treated as no history rather than as corruption, so the user is not warned
// about a file that never held anything.
func TestAppendHistory_EmptyFileIsNotQuarantined(t *testing.T) {
	withTempDataHome(t)
	path := writeRawHistory(t, nil)

	_, notice, err := AppendHistoryWithNotice(makeRecord(0, 70))
	if err != nil {
		t.Fatalf("AppendHistoryWithNotice: %v", err)
	}
	if !notice.IsZero() {
		t.Errorf("empty file raised %q", notice.Message)
	}
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if strings.Contains(e.Name(), ".corrupt-") {
			t.Errorf("empty file was quarantined as %s", e.Name())
		}
	}
}

// TestQuarantinePath_KeepsEarlierQuarantines proves a second corruption in the
// same second does not overwrite the first quarantine file.
func TestQuarantinePath_KeepsEarlierQuarantines(t *testing.T) {
	withTempDataHome(t)
	path := writeRawHistory(t, []byte("not json"))
	now := time.Date(2026, 8, 2, 9, 30, 0, 0, time.UTC)

	first := quarantinePath(path, now)
	if err := os.WriteFile(first, []byte("first"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	second := quarantinePath(path, now)
	if second == first {
		t.Fatalf("second quarantine reused %q and would destroy the first", first)
	}
}
