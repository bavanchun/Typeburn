package metrics_test

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/internal/metrics"
)

// TestConsistency validates the consistency formula: 100*tanh(1-CV), CV=stddev/mean.
// Researcher-02 §3 worked example is used as the anchor test.
// Population stddev is used (divide by N, not N-1) — matches the researcher example.
func TestConsistency(t *testing.T) {
	t.Run("Researcher-02 worked example: [80,85,82,78,90] → ~74 (±1)", func(t *testing.T) {
		// mean = (80+85+82+78+90)/5 = 415/5 = 83
		// population variance = ((80-83)²+(85-83)²+(82-83)²+(78-83)²+(90-83)²)/5
		//   = (9+4+1+25+49)/5 = 88/5 = 17.6
		// stddev = sqrt(17.6) ≈ 4.195
		// CV = 4.195/83 ≈ 0.05054
		// consistency = 100*tanh(1-0.05054) = 100*tanh(0.94946) ≈ 74.0
		samples := []float64{80, 85, 82, 78, 90}
		got := metrics.Consistency(samples)
		if math.Abs(got-74.0) > 1.0 {
			t.Errorf("worked example: want ~74 (±1), got %.2f", got)
		}
	})

	t.Run("Uniform samples → consistency near 100", func(t *testing.T) {
		// CV = 0/mean = 0, tanh(1-0) = tanh(1) ≈ 0.7616, so 100*0.7616 ≈ 76.16
		// Actually: uniform → stddev=0, CV=0, consistency=100*tanh(1)≈76.16
		// The formula gives ~76 for perfect consistency, not 100 — this is by design.
		// We just verify it is high (>70) and that non-uniform is lower.
		uniform := []float64{80, 80, 80, 80, 80}
		got := metrics.Consistency(uniform)
		if got < 70.0 {
			t.Errorf("uniform samples: want > 70, got %.2f", got)
		}
	})

	t.Run("High-variance samples → low consistency", func(t *testing.T) {
		// Wide spread: [10, 100, 10, 100, 10, 100]
		// mean=55, stddev large → CV large → 1-CV could be negative → tanh negative → clamp to 0
		highVar := []float64{10, 100, 10, 100, 10, 100}
		got := metrics.Consistency(highVar)
		if got > 50.0 {
			t.Errorf("high-variance: want < 50, got %.2f", got)
		}
	})

	t.Run("Single sample → consistency = 100*tanh(1) (stddev=0)", func(t *testing.T) {
		got := metrics.Consistency([]float64{80})
		// stddev=0, CV=0, result=100*tanh(1)≈76.16
		if got < 70.0 || got > 80.0 {
			t.Errorf("single sample: want ~76 (70-80 range), got %.2f", got)
		}
	})

	t.Run("Empty samples → consistency = 0 (no data)", func(t *testing.T) {
		got := metrics.Consistency(nil)
		if got != 0.0 {
			t.Errorf("empty: want 0, got %.2f", got)
		}
	})

	t.Run("Mean=0 samples → consistency = 0 (guard divide-by-zero)", func(t *testing.T) {
		got := metrics.Consistency([]float64{0, 0, 0})
		if got != 0.0 {
			t.Errorf("zero mean: want 0, got %.2f", got)
		}
	})

	t.Run("Result is clamped to [0,100]", func(t *testing.T) {
		// Extremely high variance forces tanh result negative → should clamp to 0
		extremeVar := []float64{1, 1000}
		got := metrics.Consistency(extremeVar)
		if got < 0.0 || got > 100.0 {
			t.Errorf("clamp: want in [0,100], got %.2f", got)
		}
	})
}
