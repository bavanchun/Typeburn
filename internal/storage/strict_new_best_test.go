package storage

import (
	"testing"
)

func TestIsNewBest_StrictRunsExcluded(t *testing.T) {
	hist := []Record{
		{Mode: "time", Length: 30, WPM: 50, NetWPM: 50.0},
	}

	// Non-strict run with higher WPM -> IsNewBest should be true
	runHigher := Record{Mode: "time", Length: 30, WPM: 60, NetWPM: 60.0, Strict: false}
	if !IsNewBest(hist, runHigher) {
		t.Error("expected non-strict higher WPM run to be a new best")
	}

	// Strict run with higher WPM -> IsNewBest should be false
	runStrictHigher := Record{Mode: "time", Length: 30, WPM: 60, NetWPM: 60.0, Strict: true}
	if IsNewBest(hist, runStrictHigher) {
		t.Error("expected strict run to be excluded from personal bests")
	}

	// Strict run on empty history -> IsNewBest should be false
	if IsNewBest(nil, runStrictHigher) {
		t.Error("expected strict run to be excluded from personal bests even on empty history")
	}
}
