package metrics

import (
	"github.com/bavanchun/Typeburn/v2/internal/mode"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

const afkThresholdMs int64 = 7000 // strictly greater than this triggers AFK trim

// TrimAFK adjusts endMs for trailing inactivity in ModeTime ONLY.
//
// Rule: if the gap between the last keystroke and endMs is strictly >7s,
// effective endMs is set to the last keystroke timestamp, removing the idle
// tail from duration and per-second bucket computation.
//
// ModeWords and ModeQuote are never trimmed, even with long gaps — the test
// ends by completion event, not by time, so idle gaps are intentional pauses.
//
// Returns the (possibly unchanged) log, the effective endMs to use for metric
// computation, and whether endMs was actually moved. The log itself is never
// modified; only endMs changes.
//
// The trimmed flag matters downstream: once the clock is cut back to the last
// keystroke, every rate is extrapolated from the burst the user typed before
// they stopped, not from the test they sat through. Such a run is shown but not
// recorded, so the caller has to be able to tell.
func TrimAFK(log []typing.Keystroke, m mode.Mode, endMs int64) ([]typing.Keystroke, int64, bool) {
	if m != mode.ModeTime {
		return log, endMs, false
	}

	// Find the last forward keystroke (non-backspace).
	lastKeyMs := int64(-1)
	for i := len(log) - 1; i >= 0; i-- {
		if log[i].Typed != 0 {
			lastKeyMs = log[i].TimeMs
			break
		}
	}

	if lastKeyMs < 0 {
		// No forward keystrokes at all — nothing to trim.
		return log, endMs, false
	}

	gap := endMs - lastKeyMs
	if gap > afkThresholdMs {
		return log, lastKeyMs, true
	}

	return log, endMs, false
}
