package metrics

import "github.com/bavanchun/Typeburn/internal/typing"

// LiveWPM estimates current net WPM from forward keystrokes in the log.
// Returns 0 when elapsedMs < 500ms (too noisy) or the log is empty.
// This is the cheap O(n) live-display estimate; full metrics come from Compute.
func LiveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
	if elapsedMs < 500 || len(log) == 0 {
		return 0
	}
	var forward int
	for _, k := range log {
		if k.Typed != 0 {
			forward++
		}
	}
	return float64(forward) / 5.0 / (float64(elapsedMs) / 60000.0)
}
