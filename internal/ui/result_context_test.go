package ui

import (
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/storage"
)

// ctxRecord builds a history record in the time/30 bucket.
func ctxRecord(wpm float64, hoursAgo int) storage.Record {
	return storage.Record{
		Time:   time.Date(2026, 1, 2, 15, 0, 0, 0, time.UTC).Add(-time.Duration(hoursAgo) * time.Hour),
		Mode:   "time",
		Length: 30,
		WPM:    int(wpm),
		NetWPM: wpm,
	}
}

// An empty history is the first-run state, not a zero-valued comparison.
func TestResultContextFor_EmptyHistory(t *testing.T) {
	got := ResultContextFor(nil, ctxRecord(80, 0))
	if got.HasHistory {
		t.Error("an empty history must not report having history")
	}
	if got.Rank != 1 || got.Total != 1 {
		t.Errorf("first run should stand alone: rank %d of %d", got.Rank, got.Total)
	}
	if got.PB != 0 || got.Avg10 != 0 {
		t.Errorf("no history means no personal best or average: %+v", got)
	}
}

// The personal best, average and rank are all scoped to the run's own bucket:
// a 60-second best must not answer a question about a 30-second run.
func TestResultContextFor_BucketScoped(t *testing.T) {
	other := ctxRecord(200, 1)
	other.Length = 60
	hist := []storage.Record{ctxRecord(70, 5), ctxRecord(90, 4), other, ctxRecord(80, 3)}

	got := ResultContextFor(hist, ctxRecord(85, 0))
	if !got.HasHistory {
		t.Fatal("three comparable runs should count as history")
	}
	if got.PB != 90 {
		t.Errorf("PB = %.0f, want 90 (the 200 belongs to another bucket)", got.PB)
	}
	if got.Avg10 != 80 {
		t.Errorf("avg = %.0f, want 80", got.Avg10)
	}
	if got.Rank != 2 || got.Total != 4 {
		t.Errorf("rank %d of %d, want 2 of 4", got.Rank, got.Total)
	}
}

// The average uses the most recent window, not the whole history.
func TestResultContextFor_AverageUsesRecentWindow(t *testing.T) {
	var hist []storage.Record
	for i := 0; i < 20; i++ {
		// Oldest first, matching how history is stored.
		wpm := 40.0
		if i >= 10 {
			wpm = 100
		}
		hist = append(hist, ctxRecord(wpm, 20-i))
	}
	if got := ResultContextFor(hist, ctxRecord(90, 0)).Avg10; got != 100 {
		t.Errorf("avg over the last %d runs = %.0f, want 100", avgWindow, got)
	}
}

// A tie ranks ahead of this run: repeating a score is not an improvement on it.
func TestResultContextFor_TiesRankAhead(t *testing.T) {
	hist := []storage.Record{ctxRecord(85, 2), ctxRecord(85, 1)}
	if got := ResultContextFor(hist, ctxRecord(85, 0)); got.Rank != 3 {
		t.Errorf("rank = %d, want 3 of %d", got.Rank, got.Total)
	}
}

// A run that cannot hold a personal best has no comparable bucket, so it reads
// as a first run rather than being ranked against runs it cannot be compared to.
func TestResultContextFor_IneligibleRunHasNoStanding(t *testing.T) {
	hist := []storage.Record{ctxRecord(70, 2), ctxRecord(90, 1)}
	strict := ctxRecord(85, 0)
	strict.Strict = true
	if got := ResultContextFor(hist, strict); got.HasHistory {
		t.Errorf("a strict run must not be ranked against ordinary runs: %+v", got)
	}
	code := ctxRecord(85, 0)
	code.Mode = "code"
	if got := ResultContextFor(hist, code); got.HasHistory {
		t.Errorf("a code run must not be ranked against word runs: %+v", got)
	}
}

// Unranked keeps the facts about the history and drops only this run's place.
func TestResultContext_Unranked(t *testing.T) {
	full := ResultContextFor([]storage.Record{ctxRecord(90, 1)}, ctxRecord(85, 0))
	got := full.Unranked()
	if got.Rank != 0 || got.Total != 0 {
		t.Errorf("unranked still claims a place: %+v", got)
	}
	if got.PB != full.PB || got.Avg10 != full.Avg10 || !got.HasHistory {
		t.Errorf("unranked dropped facts about the history: %+v", got)
	}
}
