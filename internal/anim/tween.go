package anim

// Tween describes a single timed animation: it starts at StartMs, runs for
// DurMs milliseconds, and shapes its progress with Ease. All fields are plain
// data so a Tween is cheap to copy and trivial to construct inline.
type Tween struct {
	StartMs int64
	DurMs   int64
	// Ease maps linear progress [0,1] to eased progress [0,1]. If nil, linear.
	Ease func(float64) float64
}

// Progress returns the eased progress at nowMs, clamped to [0,1]: 0 at or
// before StartMs, 1 at or after StartMs+DurMs. A zero DurMs is treated as an
// instantly-complete tween (returns 1 once started) to avoid divide-by-zero.
func (tw Tween) Progress(nowMs int64) float64 {
	if tw.DurMs <= 0 {
		if nowMs < tw.StartMs {
			return 0
		}
		return 1
	}
	raw := float64(nowMs-tw.StartMs) / float64(tw.DurMs)
	raw = Clamp01(raw)
	if tw.Ease == nil {
		return raw
	}
	return tw.Ease(raw)
}

// Done reports whether the tween has reached or passed its end time.
func (tw Tween) Done(nowMs int64) bool {
	return nowMs >= tw.StartMs+tw.DurMs
}

// LerpFloat interpolates between from and to by t (caller supplies eased t).
func LerpFloat(from, to, t float64) float64 {
	return from + (to-from)*t
}

// LerpInt interpolates between two ints by t and rounds to the nearest int.
// Used by the result count-up; rounding (not truncation) makes it land exactly
// on the target at t=1.
func LerpInt(from, to int, t float64) int {
	v := float64(from) + (float64(to)-float64(from))*t
	if v < 0 {
		return int(v - 0.5)
	}
	return int(v + 0.5)
}
