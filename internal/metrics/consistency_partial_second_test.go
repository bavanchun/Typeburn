package metrics_test

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// evenLog types n characters at a fixed interval starting at t0.
func evenLog(t0, intervalMs int64, n int) []typing.Keystroke {
	log := make([]typing.Keystroke, n)
	for i := range log {
		log[i] = typing.Keystroke{
			TimeMs: t0 + int64(i)*intervalMs, Typed: 'a', Target: 'a', Correct: true,
		}
	}
	return log
}

// perfectConsistency is 100*tanh(1), the score an entirely even typist earns.
// It is the formula's maximum, and nothing an even typist does should fall short
// of it.
var perfectConsistency = 100 * math.Tanh(1)

// TestConsistency_EvenTypingScoresTheMaximumWhateverSecondItEndsIn is the
// property the per-second buckets violated. Buckets are scaled to a full second
// each, so a run ending part-way through its last second reported that second
// at a fraction of the pace the user was actually holding. The invented dip is
// indistinguishable from erratic typing, and an even typist was marked down for
// nothing but where the clock stopped.
//
// 5 characters per second, run length varied across a second boundary:
// every one of these is the same typist.
func TestConsistency_EvenTypingScoresTheMaximumWhateverSecondItEndsIn(t *testing.T) {
	for _, tc := range []struct {
		name       string
		chars      int
		durationMs int64
	}{
		{"ends on the boundary", 15, 3000},
		{"ends 200ms into the next second", 17, 3200},
		{"ends 400ms into the next second", 19, 3400},
		{"ends 800ms into the next second", 23, 3800},
	} {
		t.Run(tc.name, func(t *testing.T) {
			log := evenLog(1000, 200, tc.chars)
			got := metrics.Compute(log, config.ModeWords, 1000+tc.durationMs).Consistency

			if math.Abs(got-perfectConsistency) > 0.01 {
				t.Errorf("consistency %.2f for an entirely even typist, want %.2f",
					got, perfectConsistency)
			}
		})
	}
}

// TestConsistency_PartialSecondStaysInTheGraph: the correction removes the
// partial second from the score, not from the data. The breakdown the result
// graph draws still accounts for every character typed.
func TestConsistency_PartialSecondStaysInTheGraph(t *testing.T) {
	log := evenLog(1000, 200, 17) // 3.2 seconds of typing
	r := metrics.Compute(log, config.ModeWords, 1000+3200)

	if len(r.PerSecond) != 4 {
		t.Fatalf("want 4 per-second buckets, got %d", len(r.PerSecond))
	}
	total := 0
	for _, ps := range r.PerSecond {
		total += ps.TotalChars
	}
	if total != 17 {
		t.Errorf("per-second buckets account for %d of 17 characters", total)
	}
	if r.PerSecond[3].TotalChars != 2 {
		t.Errorf("the partial second must keep its 2 characters, got %d", r.PerSecond[3].TotalChars)
	}
}

// TestConsistency_UnevenTypingIsStillMarkedDown proves the correction did not
// simply stop measuring. A typist who genuinely changes pace between complete
// seconds still scores below the maximum.
func TestConsistency_UnevenTypingIsStillMarkedDown(t *testing.T) {
	// 10 chars in the first second, 2 in the second, 10 in the third.
	var log []typing.Keystroke
	add := func(base int64, n int, step int64) {
		for i := 0; i < n; i++ {
			log = append(log, typing.Keystroke{
				TimeMs: base + int64(i)*step, Typed: 'a', Target: 'a', Correct: true,
			})
		}
	}
	add(1000, 10, 90)
	add(2000, 2, 400)
	add(3000, 10, 90)

	got := metrics.Compute(log, config.ModeWords, 1000+3000).Consistency
	if got >= perfectConsistency-1 {
		t.Errorf("consistency %.2f for a typist who stalled for a second, want well below %.2f",
			got, perfectConsistency)
	}
}

// TestConsistency_RunShorterThanASecondHasNoScore: with no complete second
// there is no per-second sample, and a variance over one invented bucket would
// be a score derived from a single data point.
func TestConsistency_RunShorterThanASecondHasNoScore(t *testing.T) {
	log := evenLog(1000, 100, 4) // 300ms of typing
	if got := metrics.Compute(log, config.ModeWords, 1400).Consistency; got != 0 {
		t.Errorf("consistency %.2f from less than one second of typing, want 0", got)
	}
}
