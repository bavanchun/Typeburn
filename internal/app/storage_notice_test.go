package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/storage"
)

// Storage can complete a write and still have something to say about it: the
// previous history file was unparseable and had to be set aside, or the write
// went ahead without the cross-process lock. Neither is an error — the result
// is on disk either way — so neither travels the error path, and a caller that
// only checks err reports nothing.
//
// That is how losing a history file stayed silent: the old code discarded the
// unparseable bytes and overwrote them without a word. These tests assert the
// condition reaches the user, not merely that the write succeeded.

// plantCorruptHistory writes an unparseable history file into the sandbox the
// caller has already established, and returns the data directory holding it.
func plantCorruptHistory(t *testing.T) string {
	t.Helper()
	path, err := storage.HistoryPath()
	if err != nil {
		t.Fatalf("HistoryPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`[{"mode":"time","net_`), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return filepath.Dir(path)
}

func TestResult_QuarantineIsReportedToTheUser(t *testing.T) {
	m := sandboxedModel(t)
	plantCorruptHistory(t)

	m = m.handleResultMsg(typedTimeRun())

	if m.persistErr == "" {
		t.Fatal("the history file was corrupt and got set aside, but the user was told nothing")
	}
	if strings.Contains(m.persistErr, "Couldn't save") {
		t.Fatalf("reported as a save failure, but the record was saved: %q", m.persistErr)
	}

	// Setting the old file aside must not cost the run that triggered it.
	if got := len(storage.LoadHistory()); got != 1 {
		t.Fatalf("history holds %d records after the append, want 1", got)
	}
}

func TestResult_QuarantineNoticeNamesTheSavedFile(t *testing.T) {
	m := sandboxedModel(t)
	dir := plantCorruptHistory(t)

	m = m.handleResultMsg(typedTimeRun())

	// A notice saying "your history was corrupt" without saying where it went
	// leaves the user unable to act on it.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read data dir: %v", err)
	}
	var quarantined string
	for _, e := range entries {
		if strings.Contains(e.Name(), "corrupt-") {
			quarantined = e.Name()
		}
	}
	if quarantined == "" {
		t.Fatal("no quarantine file was created")
	}
	if !strings.Contains(m.persistErr, "corrupt-") {
		t.Fatalf("notice %q does not point at the quarantine file %q", m.persistErr, quarantined)
	}
}

func TestResult_CleanAppendSaysNothing(t *testing.T) {
	m := sandboxedModel(t)

	m = m.handleResultMsg(typedTimeRun())

	if m.persistErr != "" {
		t.Fatalf("an ordinary save raised a notice: %q", m.persistErr)
	}
}
