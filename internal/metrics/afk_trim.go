package metrics

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/typing"
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
// Returns the (possibly unchanged) log and the effective endMs to use for
// metric computation. The log itself is never modified; only endMs changes.
func TrimAFK(log []typing.Keystroke, mode config.Mode, endMs int64) ([]typing.Keystroke, int64) {
	if mode != config.ModeTime {
		return log, endMs
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
		return log, endMs
	}

	gap := endMs - lastKeyMs
	if gap > afkThresholdMs {
		return log, lastKeyMs
	}

	return log, endMs
}
