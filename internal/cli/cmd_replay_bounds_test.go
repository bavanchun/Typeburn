package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

func replayFixture(t *testing.T, log []typing.Keystroke, endMs int64) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "log.json")
	writeReplayFixture(t, path, replayInput{
		SchemaVersion: replaySchemaV1,
		Mode:          config.ModeWords,
		EndMs:         endMs,
		Log:           log,
	})
	return path
}

// TestLoadReplayInput_RejectsImpossibleTimestamps: the log is recorded in order
// against a monotonic clock, so anything else came from a file the program did
// not write and must not be handed to the metrics path.
func TestLoadReplayInput_RejectsImpossibleTimestamps(t *testing.T) {
	for _, tc := range []struct {
		name  string
		log   []typing.Keystroke
		endMs int64
		want  string
	}{
		{
			name:  "negative first timestamp",
			log:   []typing.Keystroke{{TimeMs: -5, Typed: 'a', Target: 'a', Correct: true}},
			endMs: 1000,
			want:  "non-negative",
		},
		{
			name: "time runs backwards",
			log: []typing.Keystroke{
				{TimeMs: 500, Typed: 'a', Target: 'a', Correct: true},
				{TimeMs: 100, Typed: 'b', Target: 'b', Correct: true},
			},
			endMs: 1000,
			want:  "non-decreasing",
		},
		{
			name: "span beyond the limit",
			log: []typing.Keystroke{
				{TimeMs: 0, Typed: 'a', Target: 'a', Correct: true},
				{TimeMs: 1 << 32, Typed: 'b', Target: 'b', Correct: true},
			},
			endMs: 1 << 32,
			want:  "span",
		},
		{
			name:  "end_ms beyond the limit",
			log:   []typing.Keystroke{{TimeMs: 0, Typed: 'a', Target: 'a', Correct: true}},
			endMs: 1 << 32,
			want:  "end_ms",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := loadReplayInput(replayFixture(t, tc.log, tc.endMs))
			if err == nil {
				t.Fatal("expected a validation error")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not mention %q", err, tc.want)
			}
			if ExitCode(err) != ExitIO {
				t.Errorf("want IO exit, got %d", ExitCode(err))
			}
		})
	}
}

// TestLoadReplayInput_RejectsASpanBeforeAllocatingIt is the memory assertion.
// The metrics path allocates one bucket per second of span, so a span of 2^32
// ms is a ~170 MiB allocation chosen by whoever wrote the file. Rejecting it
// after the fact is not a fix; it has to be rejected first.
func TestLoadReplayInput_RejectsASpanBeforeAllocatingIt(t *testing.T) {
	path := replayFixture(t, []typing.Keystroke{
		{TimeMs: 0, Typed: 'a', Target: 'a', Correct: true},
		{TimeMs: 1 << 32, Typed: 'b', Target: 'b', Correct: true},
	}, 1<<32)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)

	if _, err := loadReplayInput(path); err == nil {
		t.Fatal("a 2^32 ms span was accepted")
	}

	runtime.ReadMemStats(&after)
	const budget = 32 << 20
	if grew := after.TotalAlloc - before.TotalAlloc; grew > budget {
		t.Errorf("rejecting the log allocated %d bytes, budget %d — it was sized from the span first", grew, budget)
	}
}

// TestLoadReplayInput_RejectsAnOversizeFile bounds the read itself: a path can
// name anything, including something that is not a file worth loading whole.
func TestLoadReplayInput_RejectsAnOversizeFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "huge.json")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", maxReplayBytes+16)), 0600); err != nil {
		t.Fatal(err)
	}

	_, err := loadReplayInput(path)

	if err == nil || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("got err %v, want an oversize-file error", err)
	}
}

// TestLoadReplayInput_AcceptsARealisticRun guards the other direction: the
// bounds must not reject a log the program itself would write.
func TestLoadReplayInput_AcceptsARealisticRun(t *testing.T) {
	log := make([]typing.Keystroke, 300)
	for i := range log {
		log[i] = typing.Keystroke{TimeMs: int64(i) * 180, Typed: 'a', Target: 'a', Correct: true}
	}

	if _, err := loadReplayInput(replayFixture(t, log, int64(len(log))*180)); err != nil {
		t.Fatalf("a 54-second run was rejected: %v", err)
	}
}
