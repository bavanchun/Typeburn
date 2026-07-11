package metrics_test

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// buildLog is a helper that runs an Engine and returns its keystroke log.
func buildLog(target string, mode config.Mode, wordTarget int, input string) []typing.Keystroke {
	e := typing.New(target, mode, wordTarget)
	var ms int64 = 1000
	for _, r := range input {
		e.Apply(r, ms)
		ms += 500
	}
	return e.Log()
}

// TestNetAndRawWPM validates the WPM formulas.
func TestNetAndRawWPM(t *testing.T) {
	t.Run("NetWPM = correctChars/5/minutes", func(t *testing.T) {
		// target "hello world" (11 chars incl space), type all correct in 60s
		// correctChars=11, minutes=1 → NetWPM = 11/5/1 = 2.2
		log := buildLogTimed("hello world", config.ModeWords, 2, "hello world", 0, 60000)
		r := metrics.Compute(log, config.ModeWords, 60000)
		want := 11.0 / 5.0 / 1.0
		if math.Abs(r.NetWPM-want) > 0.1 {
			t.Errorf("NetWPM: want %.2f, got %.2f", want, r.NetWPM)
		}
	})

	t.Run("RawWPM = allTypedChars/5/minutes", func(t *testing.T) {
		// type 10 chars (some wrong) in 60s
		// "hello world", type "hellx world" (1 error, not corrected)
		log := buildLogTimed("hello world", config.ModeWords, 2, "hellx world", 0, 60000)
		r := metrics.Compute(log, config.ModeWords, 60000)
		// all 11 chars typed, minutes=1 → RawWPM = 11/5/1 = 2.2
		want := 11.0 / 5.0 / 1.0
		if math.Abs(r.RawWPM-want) > 0.1 {
			t.Errorf("RawWPM: want %.2f, got %.2f", want, r.RawWPM)
		}
	})

	t.Run("NetWPM lower than RawWPM when errors present", func(t *testing.T) {
		log := buildLogTimed("hello world", config.ModeWords, 2, "hellx world", 0, 60000)
		r := metrics.Compute(log, config.ModeWords, 60000)
		if r.NetWPM >= r.RawWPM {
			t.Errorf("NetWPM (%.2f) should be < RawWPM (%.2f) with errors", r.NetWPM, r.RawWPM)
		}
	})

	t.Run("Zero duration guard: no panic, zero WPM", func(t *testing.T) {
		// empty log → startMs == 0, duration == 0
		r := metrics.Compute(nil, config.ModeWords, 0)
		if r.NetWPM != 0 || r.RawWPM != 0 {
			t.Errorf("empty log: want 0 WPMs, got net=%.2f raw=%.2f", r.NetWPM, r.RawWPM)
		}
	})
}

// TestAccuracy validates accuracy formula and corrected-error edge case.
func TestAccuracy(t *testing.T) {
	t.Run("All correct → 100%", func(t *testing.T) {
		log := buildLogTimed("hi", config.ModeWords, 1, "hi", 0, 5000)
		r := metrics.Compute(log, config.ModeWords, 5000)
		if math.Abs(r.Accuracy-100.0) > 0.01 {
			t.Errorf("want 100%% accuracy, got %.2f", r.Accuracy)
		}
	})

	t.Run("Corrected error → accuracy 100%% for that char", func(t *testing.T) {
		// type 'x', backspace, type 'h' → final state correct
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('x', 1000)
		e.Backspace(1500)
		e.Apply('h', 2000)
		e.Apply('i', 2500)
		log := e.Log()
		r := metrics.Compute(log, config.ModeWords, 3000)
		if math.Abs(r.Accuracy-100.0) > 0.01 {
			t.Errorf("corrected error: want 100%% accuracy, got %.2f", r.Accuracy)
		}
	})

	t.Run("One uncorrected error reduces accuracy", func(t *testing.T) {
		// "hi" typed as "hx" — 1 correct, 1 incorrect
		log := buildLogTimed("hi", config.ModeWords, 1, "hx", 0, 5000)
		r := metrics.Compute(log, config.ModeWords, 5000)
		want := 100.0 * 1.0 / 2.0 // 50%
		if math.Abs(r.Accuracy-want) > 0.1 {
			t.Errorf("want accuracy %.2f, got %.2f", want, r.Accuracy)
		}
	})

	t.Run("Zero chars typed → accuracy 100%, no panic", func(t *testing.T) {
		r := metrics.Compute(nil, config.ModeWords, 0)
		if math.Abs(r.Accuracy-100.0) > 0.01 {
			t.Errorf("empty: want 100%% accuracy, got %.2f", r.Accuracy)
		}
	})
}

// TestCPS validates characters-per-second calculation.
func TestCPS(t *testing.T) {
	t.Run("CPS = totalTypedChars / (durationMs/1000)", func(t *testing.T) {
		// type 10 chars in 10 seconds → CPS = 1.0
		log := buildLogTimed("hello world", config.ModeWords, 2, "hello worl", 0, 10000)
		r := metrics.Compute(log, config.ModeWords, 10000)
		// 10 chars, 10s → CPS = 1.0
		if math.Abs(r.CPS-1.0) > 0.05 {
			t.Errorf("want CPS=1.0, got %.3f", r.CPS)
		}
	})
}

// TestErrorCount validates error counting.
func TestErrorCount(t *testing.T) {
	t.Run("Errors = uncorrected incorrect chars in final state", func(t *testing.T) {
		// "hi" typed as "hx" → 1 error
		log := buildLogTimed("hi", config.ModeWords, 1, "hx", 0, 5000)
		r := metrics.Compute(log, config.ModeWords, 5000)
		if r.Errors != 1 {
			t.Errorf("want 1 error, got %d", r.Errors)
		}
	})

	t.Run("Corrected error not counted in Errors", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('x', 1000)
		e.Backspace(1500)
		e.Apply('h', 2000)
		e.Apply('i', 2500)
		r := metrics.Compute(e.Log(), config.ModeWords, 3000)
		if r.Errors != 0 {
			t.Errorf("corrected error: want 0 errors, got %d", r.Errors)
		}
	})
}

// TestDurationMs validates duration is endMs - startMs (first keystroke).
func TestDurationMs(t *testing.T) {
	t.Run("Duration = endMs - first keystroke time", func(t *testing.T) {
		log := buildLogTimed("hi", config.ModeWords, 1, "hi", 5000, 15000)
		r := metrics.Compute(log, config.ModeWords, 15000)
		// first key at 5000, endMs=15000 → duration=10000
		if r.DurationMs != 10000 {
			t.Errorf("want DurationMs=10000, got %d", r.DurationMs)
		}
	})
}

// TestPerSecondBuckets validates per-second bucketing.
func TestPerSecondBuckets(t *testing.T) {
	t.Run("Keys in same second land in same bucket", func(t *testing.T) {
		// 5 keys in [0,999ms], 5 keys in [1000,1999ms] → 2 buckets
		log := buildLogTimed("hello world", config.ModeWords, 2, "hello", 0, 4999)
		r := metrics.Compute(log, config.ModeWords, 5000)
		if len(r.PerSecond) == 0 {
			t.Error("want at least one per-second bucket")
		}
	})

	t.Run("On-boundary key at exactly 1000ms goes to second bucket", func(t *testing.T) {
		// bucket [0,1000) and [1000,2000): key at 1000ms → bucket index 1
		// This tests the half-open [Ns, (N+1)s) boundary
		e := typing.New("ab", config.ModeWords, 1)
		e.Apply('a', 0)    // bucket 0: [0,1000)
		e.Apply('b', 1000) // bucket 1: [1000,2000)
		log := e.Log()
		r := metrics.Compute(log, config.ModeWords, 2000)
		if len(r.PerSecond) < 2 {
			t.Errorf("want 2 buckets for on-boundary key, got %d", len(r.PerSecond))
		}
	})
}

// buildLogTimed builds a keystroke log with keys spread evenly between startMs and endMs.
func buildLogTimed(target string, mode config.Mode, wordTarget int, input string, startMs, endMs int64) []typing.Keystroke {
	e := typing.New(target, mode, wordTarget)
	runes := []rune(input)
	n := len(runes)
	if n == 0 {
		return e.Log()
	}
	var step int64 = 1
	if n > 1 {
		step = (endMs - startMs) / int64(n)
		if step < 1 {
			step = 1
		}
	}
	for i, r := range runes {
		e.Apply(r, startMs+int64(i)*step)
	}
	return e.Log()
}
