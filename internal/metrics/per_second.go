package metrics

import (
	"monkeytype-tui/internal/typing"
)

// PerSecond holds a single one-second interval's worth of typing statistics,
// relative to the test start (first keystroke).
type PerSecond struct {
	Sec          int     // 0-based second index (0 = first second of the test)
	RawWPM       float64 // raw WPM for chars typed in this interval
	Errors       int     // number of incorrect keystrokes in this interval
	CorrectChars int     // correct keystrokes in this interval
	TotalChars   int     // all forward (non-backspace) keystrokes in this interval
}

// bucketPerSecond groups non-backspace keystrokes into half-open one-second
// intervals [Ns, (N+1)s) relative to startMs (the first keystroke timestamp).
//
// Boundary rule: a keystroke at exactly Ns falls into bucket N (not N-1).
// Example: keystroke at startMs+1000ms → bucket index 1 ([1000, 2000)).
//
// Backspace events (Typed == 0) are excluded from bucketing; they affect the
// final char state (and thus accuracy) but are not forward keystrokes for WPM.
func bucketPerSecond(log []typing.Keystroke, startMs int64) []PerSecond {
	if len(log) == 0 {
		return nil
	}

	// Find the maximum offset to size the bucket slice.
	var maxOffsetMs int64
	for _, k := range log {
		if k.Typed == 0 {
			continue // skip backspace events
		}
		offset := k.TimeMs - startMs
		if offset > maxOffsetMs {
			maxOffsetMs = offset
		}
	}

	numBuckets := int(maxOffsetMs/1000) + 1
	buckets := make([]PerSecond, numBuckets)
	for i := range buckets {
		buckets[i].Sec = i
	}

	for _, k := range log {
		if k.Typed == 0 {
			continue // backspace — not a forward keystroke
		}
		offset := k.TimeMs - startMs
		if offset < 0 {
			offset = 0
		}
		idx := int(offset / 1000) // half-open [Ns, (N+1)s)
		if idx >= numBuckets {
			idx = numBuckets - 1
		}
		buckets[idx].TotalChars++
		if k.Correct {
			buckets[idx].CorrectChars++
		} else {
			buckets[idx].Errors++
		}
	}

	// Compute RawWPM per bucket: (totalChars / 5) * 60
	// Each bucket represents 1 second, so minutes = 1/60.
	for i := range buckets {
		buckets[i].RawWPM = float64(buckets[i].TotalChars) / 5.0 * 60.0
	}

	return buckets
}
