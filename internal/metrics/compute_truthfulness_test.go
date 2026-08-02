package metrics_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// typeInto applies each rune of s at 100ms intervals from ts and returns the
// timestamp after the last one.
func typeInto(e *typing.Engine, s string, ts int64) int64 {
	for _, r := range s {
		e.Apply(r, ts)
		ts += 100
	}
	return ts
}

// TestCompute_ZeroDurationReportsWhatWasTyped: a run with no measurable
// duration still has facts about what the user pressed. Reporting 100% for it
// invented a flawless run out of a single wrong keystroke, and because that was
// also the first record in its bucket it became a permanent personal best.
func TestCompute_ZeroDurationReportsWhatWasTyped(t *testing.T) {
	// One wrong key at t=1000 in a 15s-idle Time run: the trim pulls endMs back
	// onto that keystroke, leaving a window of zero.
	log := []typing.Keystroke{{TimeMs: 1000, Typed: 'q', Target: 'h', Correct: false}}

	r := metrics.Compute(log, config.ModeTime, 16000)

	if r.DurationMs != 0 {
		t.Fatalf("precondition: want a zero-length window, got %dms", r.DurationMs)
	}
	if r.Accuracy != 0 {
		t.Errorf("Accuracy is %.1f for a run whose only keystroke was wrong, want 0", r.Accuracy)
	}
	if r.KeystrokeAccuracy != 0 {
		t.Errorf("KeystrokeAccuracy is %.1f, want 0", r.KeystrokeAccuracy)
	}
	if r.IncorrectChars != 1 || r.CorrectChars != 0 {
		t.Errorf("want 0 correct / 1 incorrect, got %d / %d", r.CorrectChars, r.IncorrectChars)
	}
}

// TestCompute_EmptyLogIsStillPerfect keeps the one case that legitimately
// reports 100%: nothing was typed, so nothing was typed wrong.
func TestCompute_EmptyLogIsStillPerfect(t *testing.T) {
	r := metrics.Compute(nil, config.ModeTime, 30000)
	if r.Accuracy != 100 || r.KeystrokeAccuracy != 100 {
		t.Errorf("empty log: want 100/100, got %.1f/%.1f", r.Accuracy, r.KeystrokeAccuracy)
	}
}

// TestCompute_AFKTrimIsReportedAndNoRateIsExtrapolated covers the run the
// plausibility floor caught: three keys inside a tenth of a second in a
// 30-second test. The trim leaves a window far too short to project a
// per-minute rate from, and the caller has to be able to tell the run apart
// from one that was actually typed.
func TestCompute_AFKTrimIsReportedAndNoRateIsExtrapolated(t *testing.T) {
	log := []typing.Keystroke{
		{TimeMs: 1000, Typed: 'a', Target: 'a', Correct: true},
		{TimeMs: 1050, Typed: 'b', Target: 'b', Correct: true},
		{TimeMs: 1100, Typed: 'c', Target: 'c', Correct: true},
	}

	r := metrics.Compute(log, config.ModeTime, 1000+30_000)

	if !r.AFKTrimmed {
		t.Error("a run whose clock was cut back must say so")
	}
	if r.NetWPM != 0 || r.RawWPM != 0 || r.CPS != 0 {
		t.Errorf("rates projected from a %dms window: net %.1f raw %.1f cps %.1f",
			r.DurationMs, r.NetWPM, r.RawWPM, r.CPS)
	}
	// The characters themselves are still real.
	if r.CorrectChars != 3 {
		t.Errorf("want 3 correct chars, got %d", r.CorrectChars)
	}
}

// TestCompute_TypedRunIsNotMarkedAFK guards the other direction: a run typed
// through to the end must not be withheld.
func TestCompute_TypedRunIsNotMarkedAFK(t *testing.T) {
	var log []typing.Keystroke
	for i := 0; i < 150; i++ {
		log = append(log, typing.Keystroke{
			TimeMs: 1000 + int64(i)*200, Typed: 'a', Target: 'a', Correct: true,
		})
	}
	r := metrics.Compute(log, config.ModeTime, 1000+30_000)
	if r.AFKTrimmed {
		t.Error("a run typed up to the final seconds must not be treated as abandoned")
	}
	if r.NetWPM <= 0 {
		t.Errorf("want a real rate, got %.2f", r.NetWPM)
	}
}

// TestCompute_StrictAgreesWithTheEngineBuffer is the cross-check the strict
// desync escaped: whatever the engine ends up holding, the replay must count
// exactly that. The run below types three letters, hammers a wrong key five
// times against a blocked cursor, deletes what it has and retypes the target in
// full — so the engine holds "abcdef" and every character in it is correct.
func TestCompute_StrictAgreesWithTheEngineBuffer(t *testing.T) {
	e := typing.NewStrict("abcdef", config.ModeWords, 1, true)
	ts := typeInto(e, "abc", 1000)
	ts = typeInto(e, "zzzzz", ts)
	for i := 0; i < 3; i++ {
		e.Backspace(ts)
		ts += 100
	}
	ts = typeInto(e, "abcdef", ts)

	if got := string(e.Typed()); got != "abcdef" {
		t.Fatalf("precondition: engine holds %q, want %q", got, "abcdef")
	}

	r := metrics.Compute(e.Log(), config.ModeWords, ts)

	if r.CorrectChars != 6 {
		t.Errorf("CorrectChars=%d, but the engine holds 6 correct characters", r.CorrectChars)
	}
	if r.IncorrectChars != 0 {
		t.Errorf("IncorrectChars=%d, but nothing wrong survived in the buffer", r.IncorrectChars)
	}
	if r.ExtraChars != 0 {
		t.Errorf("ExtraChars=%d, but nothing was typed past the target", r.ExtraChars)
	}
	if r.Accuracy != 100 {
		t.Errorf("Accuracy=%.2f on a buffer that matches the target exactly", r.Accuracy)
	}
}

// TestCompute_StrictErrorsAreKeystrokeLevel: under strict mode a wrong key
// never reaches the final state, so counting only the final state reports a run
// full of mistakes as error-free. Every wrong key is charged, including the
// ones the user went back and fixed.
func TestCompute_StrictErrorsAreKeystrokeLevel(t *testing.T) {
	e := typing.NewStrict("abcdef", config.ModeWords, 1, true)
	ts := typeInto(e, "abc", 1000)
	ts = typeInto(e, "zzzzz", ts) // five refused keys
	ts = typeInto(e, "def", ts)

	r := metrics.Compute(e.Log(), config.ModeWords, ts)

	if r.Errors != 5 {
		t.Errorf("Errors=%d, want 5 — every key the mode refused", r.Errors)
	}
	// 6 correct out of the 11 keys pressed.
	if got := r.KeystrokeAccuracy; got < 54.5 || got > 54.6 {
		t.Errorf("KeystrokeAccuracy=%.2f, want ~54.55 (6 of 11 keys)", got)
	}
}

// TestCompute_NonStrictErrorsStillCountTheFinalState: outside strict mode a
// corrected mistake is not an error, which is the behaviour accuracy has always
// described. Only strict runs changed.
func TestCompute_NonStrictErrorsStillCountTheFinalState(t *testing.T) {
	e := typing.New("abcdef", config.ModeWords, 1)
	ts := typeInto(e, "abx", 1000)
	e.Backspace(ts)
	ts += 100
	ts = typeInto(e, "cdeZ", ts)

	r := metrics.Compute(e.Log(), config.ModeWords, ts)

	if r.Errors != 1 {
		t.Errorf("Errors=%d, want 1 — the corrected mistake is not charged, the surviving one is", r.Errors)
	}
	if r.Errors != r.IncorrectChars {
		t.Errorf("outside strict mode Errors (%d) and IncorrectChars (%d) must agree",
			r.Errors, r.IncorrectChars)
	}
}
