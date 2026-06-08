package storage

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// baseTime is a fixed reference time used to build deterministic test records.
var baseTime = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

// makeRecord builds a Record with a given offset (seconds from baseTime) and WPM.
func makeRecord(offsetSec int, wpm int) Record {
	return Record{
		Time:        baseTime.Add(time.Duration(offsetSec) * time.Second),
		Mode:        "time",
		Length:      30,
		WPM:         wpm,
		NetWPM:      float64(wpm) + 0.42,
		RawWPM:      float64(wpm) + 5,
		Accuracy:    97.0,
		Consistency: 90.0,
	}
}

// withTempDataHome sets XDG_DATA_HOME to a fresh temp directory for the
// duration of the test, then restores the original value.
func withTempDataHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_DATA_HOME", dir)
	return dir
}

// TestLoadHistory_MissingFile checks that LoadHistory returns empty slice when
// history.json does not exist (expected on first run).
func TestLoadHistory_MissingFile(t *testing.T) {
	withTempDataHome(t)
	got := LoadHistory()
	if got == nil {
		got = []Record{} // normalise nil to empty for len check
	}
	if len(got) != 0 {
		t.Errorf("want empty slice, got %d records", len(got))
	}
}

// TestLoadHistory_CorruptFile checks that LoadHistory returns empty slice and
// does not panic or error when history.json contains invalid JSON.
func TestLoadHistory_CorruptFile(t *testing.T) {
	withTempDataHome(t)
	path, err := HistoryPath()
	if err != nil {
		t.Fatalf("HistoryPath: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not valid json[[["), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	got := LoadHistory()
	if len(got) != 0 {
		t.Errorf("corrupt file: want 0 records, got %d", len(got))
	}
}

// TestAppendHistory_RoundTrip checks that a record written via AppendHistory
// is readable by LoadHistory with all fields intact.
func TestAppendHistory_RoundTrip(t *testing.T) {
	withTempDataHome(t)
	rec := makeRecord(0, 88)
	after, err := AppendHistory(rec)
	if err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}
	if len(after) != 1 {
		t.Fatalf("want 1 record after append, got %d", len(after))
	}

	loaded := LoadHistory()
	if len(loaded) != 1 {
		t.Fatalf("want 1 record after load, got %d", len(loaded))
	}
	got := loaded[0]
	if got.WPM != 88 {
		t.Errorf("WPM: want 88, got %d", got.WPM)
	}
	if got.Mode != "time" {
		t.Errorf("Mode: want time, got %q", got.Mode)
	}
	if math.Abs(got.Accuracy-97.0) > 0.001 {
		t.Errorf("Accuracy: want 97.0, got %f", got.Accuracy)
	}
}

// TestAppendHistory_Cap200_KeepsNewest checks that appending 205 records
// results in exactly 200 records and the oldest records are dropped.
func TestAppendHistory_Cap200_KeepsNewest(t *testing.T) {
	withTempDataHome(t)

	const total = 205
	for i := 0; i < total; i++ {
		_, err := AppendHistory(makeRecord(i, 60+i))
		if err != nil {
			t.Fatalf("AppendHistory(%d): %v", i, err)
		}
	}

	loaded := LoadHistory()
	if len(loaded) != historyCapMax {
		t.Fatalf("cap: want %d records, got %d", historyCapMax, len(loaded))
	}

	// Newest 200: offsets 5..204 (offsetSec 5..204), WPM 65..264.
	// Oldest kept record should have offsetSec = total-historyCapMax = 5.
	oldestExpectedWPM := 60 + (total - historyCapMax)
	if loaded[0].WPM != oldestExpectedWPM {
		t.Errorf("oldest kept WPM: want %d, got %d", oldestExpectedWPM, loaded[0].WPM)
	}
	// Newest record should be the last appended: WPM = 60 + 204 = 264.
	newestExpectedWPM := 60 + (total - 1)
	if loaded[len(loaded)-1].WPM != newestExpectedWPM {
		t.Errorf("newest WPM: want %d, got %d", newestExpectedWPM, loaded[len(loaded)-1].WPM)
	}
}

// TestAppendHistory_NoTmpResidue checks that no .tmp file is left behind after
// a successful atomic write.
func TestAppendHistory_NoTmpResidue(t *testing.T) {
	dir := withTempDataHome(t)
	_, err := AppendHistory(makeRecord(0, 75))
	if err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(dir, "typeburn"))
	if err != nil {
		t.Fatalf("ReadDir: %v", err)
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Errorf("unexpected .tmp residue: %s", e.Name())
		}
	}
}

// TestAppendHistory_XDGDataHome checks that HistoryPath respects XDG_DATA_HOME
// when set to an absolute path.
func TestAppendHistory_XDGDataHome(t *testing.T) {
	dir := withTempDataHome(t)
	_, err := AppendHistory(makeRecord(0, 80))
	if err != nil {
		t.Fatalf("AppendHistory: %v", err)
	}
	expectedPath := filepath.Join(dir, "typeburn", "history.json")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Errorf("history.json not found at XDG path %s", expectedPath)
	}
}

// TestAppendHistory_HomeFallback checks that HistoryPath falls back to
// ~/.local/share/typeburn/history.json when XDG_DATA_HOME is unset.
func TestAppendHistory_HomeFallback(t *testing.T) {
	// Unset XDG_DATA_HOME and point HOME to a temp dir.
	t.Setenv("XDG_DATA_HOME", "")
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)

	path, err := HistoryPath()
	if err != nil {
		t.Fatalf("HistoryPath: %v", err)
	}
	want := filepath.Join(tmpHome, ".local", "share", "typeburn", "history.json")
	if path != want {
		t.Errorf("HistoryPath: want %q, got %q", want, path)
	}
}

// TestIsNewBest_FirstResult checks that the first result for a mode is always
// a new best (no prior records in that bucket).
func TestIsNewBest_FirstResult(t *testing.T) {
	r := makeRecord(0, 80)
	if !IsNewBest(nil, r) {
		t.Error("first result should be a new best")
	}
	if !IsNewBest([]Record{}, r) {
		t.Error("first result (empty hist) should be a new best")
	}
}

// TestIsNewBest_HigherWPM checks that a higher WPM for the same mode bucket
// is correctly identified as a new best.
func TestIsNewBest_HigherWPM(t *testing.T) {
	hist := []Record{makeRecord(0, 80)}
	r := makeRecord(10, 85)
	if !IsNewBest(hist, r) {
		t.Error("85 > 80: should be a new best")
	}
}

// TestIsNewBest_EqualWPM checks that an equal WPM is NOT a new best.
func TestIsNewBest_EqualWPM(t *testing.T) {
	hist := []Record{makeRecord(0, 80)}
	r := makeRecord(10, 80)
	if IsNewBest(hist, r) {
		t.Error("80 == 80: should NOT be a new best")
	}
}

// TestIsNewBest_LowerWPM checks that a lower WPM is NOT a new best.
func TestIsNewBest_LowerWPM(t *testing.T) {
	hist := []Record{makeRecord(0, 80)}
	r := makeRecord(10, 75)
	if IsNewBest(hist, r) {
		t.Error("75 < 80: should NOT be a new best")
	}
}

// TestIsNewBest_DifferentModeSameWPM checks that a high WPM in a different mode
// does not affect the new-best result for the target mode.
func TestIsNewBest_DifferentModeSameWPM(t *testing.T) {
	// History has a words/30 record with WPM 100.
	hist := []Record{{
		Time:   baseTime,
		Mode:   "words",
		Length: 30,
		WPM:    100,
	}}
	// New result is time/30 with WPM 50 — first for that bucket.
	r := Record{Time: baseTime.Add(time.Second), Mode: "time", Length: 30, WPM: 50}
	if !IsNewBest(hist, r) {
		t.Error("first result for time/30 should be a new best even with words/30 at 100")
	}
}

// TestIsNewBest_SameModeDifferentLength checks that time/30 and time/60 are
// separate buckets.
func TestIsNewBest_SameModeDifferentLength(t *testing.T) {
	hist := []Record{{Time: baseTime, Mode: "time", Length: 60, WPM: 100}}
	r := Record{Time: baseTime.Add(time.Second), Mode: "time", Length: 30, WPM: 50}
	if !IsNewBest(hist, r) {
		t.Error("time/30 first result should be new best even with time/60 at 100")
	}
}

// TestIsNewBest_QuoteBucket checks that quote mode uses a single bucket
// regardless of length (which is always 0 for quotes).
func TestIsNewBest_QuoteBucket(t *testing.T) {
	hist := []Record{{Time: baseTime, Mode: "quote", Length: 0, WPM: 90}}
	higher := Record{Time: baseTime.Add(time.Second), Mode: "quote", Length: 0, WPM: 95}
	lower := Record{Time: baseTime.Add(2 * time.Second), Mode: "quote", Length: 0, WPM: 85}
	if !IsNewBest(hist, higher) {
		t.Error("95 > 90: quote higher should be new best")
	}
	if IsNewBest(hist, lower) {
		t.Error("85 < 90: quote lower should NOT be new best")
	}
}

// TestIsNewBest_SubWPMWin checks that a new run with a higher NetWPM but the
// same rounded WPM is correctly flagged as a new best. Without float comparison
// 75.4 and 75.0 both round to 75 and the faster run would be missed.
func TestIsNewBest_SubWPMWin(t *testing.T) {
	hist := []Record{{
		Time:   baseTime,
		Mode:   "time",
		Length: 30,
		WPM:    75,
		NetWPM: 75.0,
	}}
	newRec := Record{
		Time:   baseTime.Add(time.Second),
		Mode:   "time",
		Length: 30,
		WPM:    75,
		NetWPM: 75.4,
	}
	if !IsNewBest(hist, newRec) {
		t.Error("75.4 > 75.0: sub-WPM gain should be a new best")
	}
}

// TestIsNewBest_SubWPMTie checks that equal NetWPM values do not trigger a new best.
func TestIsNewBest_SubWPMTie(t *testing.T) {
	hist := []Record{{
		Time:   baseTime,
		Mode:   "time",
		Length: 30,
		WPM:    75,
		NetWPM: 75.0,
	}}
	newRec := Record{
		Time:   baseTime.Add(time.Second),
		Mode:   "time",
		Length: 30,
		WPM:    75,
		NetWPM: 75.0,
	}
	if IsNewBest(hist, newRec) {
		t.Error("75.0 == 75.0: tie should NOT be a new best")
	}
}

// TestIsNewBest_LegacyFallback checks that a legacy record (NetWPM==0) is
// compared by its rounded WPM so a new slower run cannot beat it. Without the
// fallback, effWPM of the legacy record would return 0 and any new run would
// always win.
func TestIsNewBest_LegacyFallback(t *testing.T) {
	// Legacy record: WPM=80, NetWPM absent → stored as 0 after unmarshal.
	hist := []Record{{
		Time:   baseTime,
		Mode:   "time",
		Length: 30,
		WPM:    80,
		NetWPM: 0, // simulates pre-NetWPM JSON record
	}}
	// New run is genuinely slower (NetWPM 60.4 → WPM 60), must NOT beat legacy 80.
	newRec := Record{
		Time:   baseTime.Add(time.Second),
		Mode:   "time",
		Length: 30,
		WPM:    60,
		NetWPM: 60.4,
	}
	if IsNewBest(hist, newRec) {
		t.Error("legacy 80 (WPM fallback) should beat new 60.4: should NOT be a new best")
	}
}

// TestIsNewBest_CodeMode checks that a Code-mode record is never a new best,
// even when it would otherwise be the first result or the highest WPM.
func TestIsNewBest_CodeMode(t *testing.T) {
	// Empty history → would normally be first-ever → still false for code.
	r := Record{Time: baseTime, Mode: "code", Length: 42, WPM: 80, NetWPM: 80.0}
	if IsNewBest(nil, r) {
		t.Error("code mode with empty hist: must never be a new best")
	}
	if IsNewBest([]Record{}, r) {
		t.Error("code mode with empty hist slice: must never be a new best")
	}
	// Existing records for other modes should not affect the code exclusion.
	hist := []Record{makeRecord(0, 40)} // time/30 WPM=40
	if IsNewBest(hist, r) {
		t.Error("code mode: must never be a new best regardless of hist")
	}
}

// TestIsNewBest_FirstInBucket checks that the first result for a mode/length
// bucket is always a new best regardless of other buckets.
func TestIsNewBest_FirstInBucket(t *testing.T) {
	// Only a words/30 record exists; time/30 bucket is empty.
	hist := []Record{{
		Time:   baseTime,
		Mode:   "words",
		Length: 30,
		WPM:    100,
		NetWPM: 100.0,
	}}
	newRec := Record{
		Time:   baseTime.Add(time.Second),
		Mode:   "time",
		Length: 30,
		WPM:    50,
		NetWPM: 50.0,
	}
	if !IsNewBest(hist, newRec) {
		t.Error("first result for time/30 bucket should always be a new best")
	}
}
