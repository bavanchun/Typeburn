package ui

import (
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// liveWPMFromCount estimates current WPM from a forward-keystroke count.
// Used for the live header display; returns 0 when elapsed < 500ms (too noisy).
// Full accuracy is computed via metrics.Compute at test completion.
func liveWPMFromCount(forward int, elapsedMs int64) float64 {
	return metrics.LiveWPMFromCount(forward, elapsedMs)
}

// typedFromLog reconstructs the current typed-rune slice by replaying the
// keystroke log. Engine.typed is unexported; the log is the public API.
//
// The replay itself belongs to the typing package, which owns what a log means.
// This file once carried its own copy of the loop, and when strict mode started
// logging keystrokes it had refused, both copies reconstructed a buffer the
// engine never held — the same defect twice, because the rule was written twice.
func typedFromLog(log []typing.Keystroke) []rune {
	return typing.TypedFromLog(log)
}
