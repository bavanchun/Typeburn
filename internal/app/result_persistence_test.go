package app

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
)

// The defect this file exists for spanned four packages: an AFK trim in
// metrics, a rate computed from what it left, a record built in app, and a
// personal best written by storage. Every one of those packages had tests, and
// every one of them agreed with itself. Nothing asserted across the seam, so a
// run the user abandoned after three keystrokes became a permanent best nobody
// could beat by typing.
//
// These tests run a real keystroke log through the whole chain and assert on
// what ends up on disk.

// abandonedTimeRun is a 30-second Time test with three keystrokes in its first
// tenth of a second and nothing after: the shape that produced the bad best.
func abandonedTimeRun() ui.ResultMsg {
	log := []typing.Keystroke{
		{TimeMs: 1000, Typed: 'a', Target: 'a', Correct: true},
		{TimeMs: 1050, Typed: 'b', Target: 'b', Correct: true},
		{TimeMs: 1100, Typed: 'c', Target: 'c', Correct: true},
	}
	return ui.ResultMsg{
		Result: metrics.Compute(log, config.ModeTime, 1000+30_000),
		Mode:   config.ModeTime,
		Length: 30,
	}
}

// typedTimeRun is the same 30-second test typed all the way through.
func typedTimeRun() ui.ResultMsg {
	log := make([]typing.Keystroke, 150)
	for i := range log {
		log[i] = typing.Keystroke{
			TimeMs: 1000 + int64(i)*200, Typed: 'a', Target: 'a', Correct: true,
		}
	}
	return ui.ResultMsg{
		Result: metrics.Compute(log, config.ModeTime, 1000+30_000),
		Mode:   config.ModeTime,
		Length: 30,
	}
}

// TestResult_AbandonedRunIsNeitherStoredNorRanked walks the full chain.
func TestResult_AbandonedRunIsNeitherStoredNorRanked(t *testing.T) {
	m := sandboxedModel(t)
	msg := abandonedTimeRun()

	if !msg.Result.AFKTrimmed {
		t.Fatal("precondition: this run must reach the root model marked as abandoned")
	}

	if out := decideOutcome(msg, storage.LoadHistory()); out.store || out.isBest {
		t.Errorf("an abandoned run resolved to store=%v isBest=%v", out.store, out.isBest)
	}

	m.handleResultMsg(msg)

	if got := storage.LoadHistory(); len(got) != 0 {
		t.Errorf("history holds %d records; an abandoned run must not be written: %+v", len(got), got)
	}
}

// TestResult_AbandonedRunCannotDisplaceALaterTypedRun is the consequence that
// made the defect permanent: a stored burst sets a bar in its bucket that no
// real run beats, so the user never sees a best again.
func TestResult_AbandonedRunCannotDisplaceALaterTypedRun(t *testing.T) {
	m := sandboxedModel(t)

	m = m.handleResultMsg(abandonedTimeRun())
	typed := typedTimeRun()

	if out := decideOutcome(typed, storage.LoadHistory()); !out.isBest {
		t.Error("the first run the user actually typed must rank as their best")
	}
	m.handleResultMsg(typed)

	hist := storage.LoadHistory()
	if len(hist) != 1 {
		t.Fatalf("want exactly the typed run stored, got %d records: %+v", len(hist), hist)
	}
}

// TestResult_TypedRunIsStoredAndRanked guards the other direction — the gate
// must not swallow ordinary results.
func TestResult_TypedRunIsStoredAndRanked(t *testing.T) {
	m := sandboxedModel(t)

	msg := typedTimeRun()
	out := decideOutcome(msg, storage.LoadHistory())
	after := m.handleResultMsg(msg)

	hist := storage.LoadHistory()
	if len(hist) != 1 {
		t.Fatalf("want 1 stored record, got %d", len(hist))
	}
	if !out.store || !out.isBest || !storage.EligibleForBest(hist[0]) {
		t.Error("a completed Time run must be stored and ranked")
	}
	if after.persistErr != "" {
		t.Errorf("no notice expected on the ordinary path, got %q", after.persistErr)
	}
}

// TestResult_WithholdingIsVisible: silently dropping a result would look like
// the app lost the run, which is worse than the score being wrong. The notice
// has to survive into the frame the user is looking at, at the smallest size
// the app supports.
func TestResult_WithholdingIsVisible(t *testing.T) {
	m := sm_sendSize(sandboxedModel(t), 80, 24).(Model)

	m = m.handleResultMsg(abandonedTimeRun())

	if m.persistErr != afkNotice {
		t.Fatalf("want the withholding notice set, got %q", m.persistErr)
	}
	frame := stripANSI(m.View().Content)
	if !strings.Contains(frame, "not saved to history") {
		t.Errorf("the user is not told why the run is missing:\n%s", frame)
	}
}
