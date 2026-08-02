package metrics

import (
	"fmt"
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/mode"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// The formula tests all pass with a Result that reports 1800 wpm, because they
// check that Compute agrees with its own arithmetic. Agreement is not
// correctness: an expression can be faithfully evaluated and still produce a
// number no human could have typed.
//
// This is the floor no amount of formula coverage supplied. It is not a
// re-derivation — it asserts only what must be true of any real run.

// maxHumanWPM is well above the fastest recorded human typist (~220 wpm on
// standard text). It is a floor for "impossible", deliberately not a limit on
// "impressive": a value past it did not come from a person typing.
const maxHumanWPM = 400

// assertPlausible fails when a Result reports a value no human could produce.
func assertPlausible(t *testing.T, label string, r Result) {
	t.Helper()
	for _, p := range implausibilities(r) {
		t.Errorf("%s: %s", label, p)
	}
}

// implausibilities returns one message per violated property, empty when the
// Result is believable.
//
// It reports rather than failing so a caller can also assert the negative — that
// a known-bad case is still bad — without a throwaway *testing.T standing in as
// a recorder. That trick relies on unexported testing internals and would take
// the whole test goroutine down through runtime.Goexit if this ever grew a
// Fatalf.
func implausibilities(r Result) []string {
	var out []string

	for _, f := range []struct {
		name string
		v    float64
	}{
		{"NetWPM", r.NetWPM}, {"RawWPM", r.RawWPM}, {"Accuracy", r.Accuracy},
		{"KeystrokeAccuracy", r.KeystrokeAccuracy}, {"Consistency", r.Consistency},
		{"CPS", r.CPS},
	} {
		if math.IsNaN(f.v) || math.IsInf(f.v, 0) {
			out = append(out, fmt.Sprintf("%s is %v — not a number a screen can render", f.name, f.v))
		}
		if f.v < 0 {
			out = append(out, fmt.Sprintf("%s is %v, want >= 0", f.name, f.v))
		}
	}

	for _, f := range []struct {
		name string
		v    float64
	}{
		{"Accuracy", r.Accuracy}, {"KeystrokeAccuracy", r.KeystrokeAccuracy},
		{"Consistency", r.Consistency},
	} {
		if f.v > 100 {
			out = append(out, fmt.Sprintf("%s is %v, want <= 100", f.name, f.v))
		}
	}

	if r.NetWPM > maxHumanWPM {
		out = append(out, fmt.Sprintf("NetWPM is %v — above %d, so it did not come from a person typing",
			r.NetWPM, maxHumanWPM))
	}
	if r.RawWPM > maxHumanWPM {
		out = append(out, fmt.Sprintf("RawWPM is %v — above %d", r.RawWPM, maxHumanWPM))
	}
	// Net counts only correct characters, so it can never exceed raw.
	if r.NetWPM > r.RawWPM+1e-9 {
		out = append(out, fmt.Sprintf("NetWPM %v exceeds RawWPM %v", r.NetWPM, r.RawWPM))
	}
	if r.DurationMs < 0 {
		out = append(out, fmt.Sprintf("DurationMs is %d", r.DurationMs))
	}
	return out
}

// keystrokes builds a correct-typing log at a fixed interval, starting at t0.
func keystrokes(t0, intervalMs int64, n int) []typing.Keystroke {
	log := make([]typing.Keystroke, n)
	for i := range log {
		log[i] = typing.Keystroke{
			TimeMs: t0 + int64(i)*intervalMs, Typed: 'a', Target: 'a', Correct: true,
		}
	}
	return log
}

// plausibilityCase names a run and the log/mode/end it produces.
type plausibilityCase struct {
	name  string
	log   []typing.Keystroke
	mode  mode.Mode
	endMs int64
}

// knownImplausible lists cases whose Result is not yet believable. A listed
// case that becomes plausible fails until its entry is deleted, and an entry no
// case produces fails as stale, so the debt cannot quietly become permanent.
var knownImplausible = map[string]bool{}

func plausibilityCases() []plausibilityCase {
	return []plausibilityCase{
		{"time/steady", keystrokes(1000, 200, 150), mode.ModeTime, 1000 + 30_000},
		{"time/afk-after-a-burst", keystrokes(1000, 20, 5), mode.ModeTime, 1000 + 30_000},
		{"words/steady", keystrokes(1000, 180, 125), mode.ModeWords, 1000 + 22_500},
		{"quote/slow", keystrokes(1000, 900, 40), mode.ModeQuote, 1000 + 36_000},
		{"time/empty", nil, mode.ModeTime, 30_000},
		{"time/single-key", keystrokes(1000, 0, 1), mode.ModeTime, 1000 + 30_000},
	}
}

// TestCompute_NeverReportsAnImpossibleResult asserts that every metric Compute
// hands to the UI and to storage is a value a human run could have produced.
func TestCompute_NeverReportsAnImpossibleResult(t *testing.T) {
	seen := map[string]bool{}
	for _, tc := range plausibilityCases() {
		r := Compute(tc.log, tc.mode, tc.endMs)

		if knownImplausible[tc.name] {
			seen[tc.name] = true
			if len(implausibilities(r)) == 0 {
				t.Errorf("%s is listed in knownImplausible but now passes — delete its entry", tc.name)
			}
			continue
		}
		assertPlausible(t, tc.name, r)
	}

	for name := range knownImplausible {
		if !seen[name] {
			t.Errorf("knownImplausible has %q, which no case produces — stale entry", name)
		}
	}
}

// TestPlausibility_RejectsAnImpossibleResult proves the floor can fail. A
// checker that never rejects anything is indistinguishable from no checker,
// and that is precisely the shape of the coverage this replaces.
func TestPlausibility_RejectsAnImpossibleResult(t *testing.T) {
	for _, bad := range []Result{
		{NetWPM: 1800, RawWPM: 1800},
		{NetWPM: 50, RawWPM: 40},
		{Accuracy: 140},
		{Consistency: -3},
		{NetWPM: math.Inf(1), RawWPM: math.Inf(1)},
	} {
		if len(implausibilities(bad)) == 0 {
			t.Errorf("the plausibility floor accepted %+v", bad)
		}
	}
}
