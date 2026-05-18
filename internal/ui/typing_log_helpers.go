package ui

import "github.com/bavanchun/Typeburn/internal/typing"

// liveWPM estimates current WPM from forward keystrokes in the log.
// Used for the live header display; returns 0 when elapsed < 500ms (too noisy).
// Full accuracy is computed via metrics.Compute at test completion.
func liveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
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
