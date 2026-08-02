package metrics

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// TestBucketPerSecond_CapsTheBucketCount is the backstop for a caller that did
// not validate its log. The slice is sized from a timestamp difference, so a
// keystroke far in the future is an allocation request; the cap turns it into a
// bounded, if meaningless, result instead.
func TestBucketPerSecond_CapsTheBucketCount(t *testing.T) {
	log := []typing.Keystroke{
		{TimeMs: 0, Typed: 'a', Target: 'a', Correct: true},
		{TimeMs: 1 << 32, Typed: 'b', Target: 'b', Correct: true},
	}

	got := bucketPerSecond(log, 0)

	if len(got) != maxBuckets {
		t.Fatalf("got %d buckets, want the cap of %d", len(got), maxBuckets)
	}
	// Both keystrokes must still be counted: the far one folds into the last
	// bucket rather than being dropped.
	var total int
	for _, b := range got {
		total += b.TotalChars
	}
	if total != len(log) {
		t.Errorf("counted %d keystrokes, want %d", total, len(log))
	}
}

// TestBucketPerSecond_LeavesRealRunsAlone: the cap is a day of typing, so it
// must be invisible to every run the program can actually record.
func TestBucketPerSecond_LeavesRealRunsAlone(t *testing.T) {
	log := make([]typing.Keystroke, 120)
	for i := range log {
		log[i] = typing.Keystroke{TimeMs: int64(i) * 1000, Typed: 'a', Target: 'a', Correct: true}
	}

	if got := len(bucketPerSecond(log, 0)); got != 120 {
		t.Errorf("a 120-second run produced %d buckets, want 120", got)
	}
}
