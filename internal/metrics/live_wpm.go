package metrics

import "github.com/bavanchun/Typeburn/v2/internal/typing"

// minRateWindowMs is the shortest window a per-minute rate may be extrapolated
// from. Below it the sample is noise: five keys inside 80 ms scale to a
// four-figure WPM, a number that says nothing about the person who typed them.
// Every rate in this package respects the same floor, so the live header and
// the final result agree on when there is nothing yet to report.
const minRateWindowMs int64 = 500

// LiveWPM estimates current net WPM from forward keystrokes in the log.
// Returns 0 when the elapsed window is too short to extrapolate from, or the
// log is empty.
// This is the cheap O(n) live-display estimate; full metrics come from Compute.
func LiveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
	if elapsedMs < minRateWindowMs || len(log) == 0 {
		return 0
	}
	var forward int
	for _, k := range log {
		if k.Typed != 0 {
			forward++
		}
	}
	return LiveWPMFromCount(forward, elapsedMs)
}

// LiveWPMFromCount estimates current net WPM from a forward-keystroke count.
func LiveWPMFromCount(forward int, elapsedMs int64) float64 {
	if elapsedMs < minRateWindowMs || forward <= 0 {
		return 0
	}
	return float64(forward) / 5.0 / (float64(elapsedMs) / 60000.0)
}
