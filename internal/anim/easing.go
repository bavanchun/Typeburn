// Package anim holds pure, UI-free motion math: easing curves, color and value
// interpolation, a time-driven tween, and a clock that reports whether any
// animation is still live. It imports no Bubble Tea / Lip Gloss types — callers
// convert returned values at the UI boundary. Every animation is a pure
// function of (startMs, nowMs, durMs), mirroring metrics.Compute's post-hoc
// replay, so renders are deterministic and unit-testable.
package anim

// Clamp01 constrains t to the [0,1] range. All easing functions assume their
// input is already clamped; callers building progress from raw time should
// clamp first (Tween.Progress does).
func Clamp01(t float64) float64 {
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t
}

// EaseOutCubic decelerates toward the end: fast start, slow finish.
// f(t) = (t-1)^3 + 1. f(0)=0, f(1)=1.
func EaseOutCubic(t float64) float64 {
	u := t - 1
	return u*u*u + 1
}

// EaseOutQuad decelerates with a gentler curve than cubic. f(t) = t*(2-t).
// Used for the WPM count-up so the final digits settle smoothly.
func EaseOutQuad(t float64) float64 {
	return t * (2 - t)
}

// EaseInOutQuad accelerates to the midpoint then decelerates. Piecewise:
// 2t^2 for t<0.5, then -1 + (4-2t)t. f(0)=0, f(0.5)=0.5, f(1)=1.
func EaseInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}
