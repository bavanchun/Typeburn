// Package metrics derives all typing metrics post-hoc from a keystroke log.
// It has zero UI or Bubble Tea dependencies.
package metrics

import "math"

// Consistency converts a slice of per-second raw WPM samples to a 0–100 score
// using the kogasa transformation: 100 * tanh(1 - CV), where CV = stddev/mean.
//
// Formula is pinned to 100*tanh(1-CV). There is minor upstream coefficient
// uncertainty (some sources use a different scalar) — this is an accepted v1
// decision documented here, not an open question. The table test in
// consistency_test.go locks the exact behaviour.
//
// Population stddev (divide by N) is used — this matches the researcher-02
// worked example where [80,85,82,78,90] → stddev≈4.195 → CV≈0.051 →
// consistency≈74 (verified to ±1).
//
// Special cases:
//   - Empty or nil slice          → 0 (no data)
//   - mean == 0                   → 0 (guard divide-by-zero; CV undefined)
//   - tanh result < 0 (high var)  → clamped to 0
//   - result > 100                → clamped to 100 (tanh(1) ≈ 0.76; never exceeds 100)
func Consistency(samples []float64) float64 {
	n := len(samples)
	if n == 0 {
		return 0
	}

	// Compute mean.
	var sum float64
	for _, v := range samples {
		sum += v
	}
	mean := sum / float64(n)

	if mean == 0 {
		return 0
	}

	// Compute population stddev (divide by N).
	var variance float64
	for _, v := range samples {
		d := v - mean
		variance += d * d
	}
	variance /= float64(n)
	stddev := math.Sqrt(variance)

	cv := stddev / mean
	result := 100.0 * math.Tanh(1.0-cv)

	// Clamp to [0, 100].
	if result < 0 {
		return 0
	}
	if result > 100 {
		return 100
	}
	return result
}
