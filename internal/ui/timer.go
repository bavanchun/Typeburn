package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// tickInterval is the wall-clock polling cadence for the typing screen timer.
const tickInterval = 100 * time.Millisecond

// tickMsg carries the fire time from a tea.Tick so Update can compute elapsed.
type tickMsg struct{ t time.Time }

// tickCmd schedules a single 100ms tick. Update re-arms it on every tickMsg to
// maintain a steady ~100ms sample rate without accumulated drift.
func tickCmd() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg{t: t}
	})
}

// elapsedMs returns the wall-clock milliseconds between startMs and now.
// Returns 0 when startMs is 0 (test not yet started).
func elapsedMs(startMs int64, now time.Time) int64 {
	if startMs == 0 {
		return 0
	}
	ms := now.UnixMilli() - startMs
	if ms < 0 {
		return 0
	}
	return ms
}

// limitMs converts a ModeTime duration in seconds to a millisecond limit.
func limitMs(seconds int) int64 { return int64(seconds) * 1000 }
