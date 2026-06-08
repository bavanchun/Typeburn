package ui

import (
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// liveWPM estimates current WPM from forward keystrokes in the log.
// Used for the live header display; returns 0 when elapsed < 500ms (too noisy).
// Full accuracy is computed via metrics.Compute at test completion.
func liveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
	return metrics.LiveWPM(log, elapsedMs)
}

// liveWPMFromCount estimates current WPM from a forward-keystroke count.
func liveWPMFromCount(forward int, elapsedMs int64) float64 {
	return metrics.LiveWPMFromCount(forward, elapsedMs)
}

// typedFromLog reconstructs the current typed-rune slice by replaying the
// keystroke log. Engine.typed is unexported; the log is the public API.
// Backspace events (Typed==0) pop the last rune, mirroring Engine internals.
func typedFromLog(log []typing.Keystroke) []rune {
	var buf []rune
	for _, k := range log {
		if k.Typed == 0 {
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
		} else {
			buf = append(buf, k.Typed)
		}
	}
	return buf
}
