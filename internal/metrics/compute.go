package metrics

import (
	"github.com/bavanchun/Typeburn/v2/internal/mode"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// Result holds all derived metrics after a completed test.
// All values are computed post-hoc from the keystroke log.
type Result struct {
	NetWPM            float64 // correct chars / 5 / minutes
	RawWPM            float64 // all typed chars / 5 / minutes
	Accuracy          float64 // 100 * correctFinal / (correctFinal + incorrectFinal)
	KeystrokeAccuracy float64 // 100 * correctForward / totalForward (non-backspace keystrokes)
	Consistency       float64 // 100 * tanh(1 - CV) of per-second raw WPM samples
	CPS               float64 // total typed chars / (durationMs / 1000)

	CorrectChars   int // chars in Correct final state
	IncorrectChars int // chars in Incorrect/IncorrectSpace final state (uncorrected)
	ExtraChars     int // chars typed past target length

	// Errors is every wrong key the run is charged for: the uncorrected errors
	// left in the final state, plus every keystroke strict mode refused. A
	// strict run has almost nothing wrong in its final state — the cursor never
	// moves past a wrong key — so counting only the final state would report a
	// run full of mistakes as error-free.
	Errors int

	DurationMs int64       // effective test duration (endMs - startMs, after AFK trim)
	PerSecond  []PerSecond // per-second breakdown

	// AFKTrimmed records that trailing inactivity cut the clock back to the last
	// keystroke. Every rate below is then extrapolated from the burst before the
	// user stopped, so such a run is displayed but never persisted or ranked.
	AFKTrimmed bool

	KeyMisses []KeyMiss // per-key fumble tally (nil when no key was ever missed)
}

// Compute derives all metrics from the keystroke log, mode, and caller-supplied
// endMs (the timestamp at which the test was declared complete).
//
// Clock starts on the first keystroke in the log; pre-first-keystroke duration
// is zero and all rate metrics return 0 to avoid divide-by-zero.
//
// For ModeTime, AFK trailing trim is applied before computing duration:
// if the gap between the last forward keystroke and endMs is >7s, endMs is
// adjusted to the last keystroke time.
//
// Accuracy uses FINAL char state (after all backspace corrections):
//   - A char typed wrong then corrected via backspace counts as Correct → 100%.
//   - An uncorrected error counts as Incorrect → penalises accuracy.
//   - Zero chars typed → Accuracy = 100, all others = 0.
//
// An empty log is the only run that reports perfect accuracy for free: nothing
// was typed, so nothing was typed wrong. A run with no measurable duration —
// every keystroke inside one millisecond, or an AFK trim that collapsed the
// window onto the last keystroke — still reports the accuracy of what it
// contains; only the rates, which have nothing to divide by, come back zero.
func Compute(log []typing.Keystroke, mode mode.Mode, endMs int64) Result {
	if len(log) == 0 {
		return Result{Accuracy: 100, KeystrokeAccuracy: 100}
	}

	// Apply AFK trim (no-op for non-Time modes).
	log, endMs, afkTrimmed := TrimAFK(log, mode, endMs)

	// startMs = first keystroke time (first entry in log).
	startMs := log[0].TimeMs

	durationMs := endMs - startMs
	if durationMs < 0 {
		durationMs = 0
	}

	// Compute final char state by replaying the log.
	// finalState maps target-position index → last typed rune (0 = deleted).
	// extraTyped counts runes typed past target length.
	finalState, extraTyped := replayFinalState(log)

	// Count correct, incorrect, extra, missed from finalState.
	correct, incorrect, extra := 0, 0, extraTyped
	for _, r := range finalState {
		if r.correct {
			correct++
		} else {
			incorrect++
		}
	}

	// Forward keystrokes (non-backspace) for RawWPM, CPS and keystroke accuracy.
	// A strict-blocked keystroke counts here: the user pressed the key, and it
	// was wrong. Only the reconstructed buffer above ignores it.
	var totalTyped, correctForward, blocked int
	for _, k := range log {
		if k.Typed == 0 {
			continue
		}
		totalTyped++
		if k.Correct {
			correctForward++
		}
		if k.Blocked {
			blocked++
		}
	}

	// Rates need a window long enough to extrapolate from. An AFK trim that cut
	// the clock back onto a short burst leaves one that is not: the same five
	// keys that read as a plausible pace over a second read as four-figure WPM
	// over eighty milliseconds. The counts and accuracies below are still facts
	// about what was typed and are reported either way.
	var netWPM, rawWPM, cps float64
	if durationMs >= minRateWindowMs {
		minutes := float64(durationMs) / 60000.0
		seconds := float64(durationMs) / 1000.0
		netWPM = float64(correct) / 5.0 / minutes
		rawWPM = float64(totalTyped) / 5.0 / minutes
		cps = float64(totalTyped) / seconds
	}

	// Accuracy: 100 * correct / (correct + incorrect) on final state.
	var accuracy float64
	if correct+incorrect == 0 {
		accuracy = 100
	} else {
		accuracy = 100.0 * float64(correct) / float64(correct+incorrect)
	}

	var keystrokeAccuracy float64
	if totalTyped == 0 {
		keystrokeAccuracy = 100
	} else {
		keystrokeAccuracy = 100.0 * float64(correctForward) / float64(totalTyped)
	}

	// Per-second buckets and consistency. Every bucket is kept for the graph;
	// consistency samples only the seconds that fully elapsed.
	perSec := bucketPerSecond(log, startMs)
	cons := Consistency(consistencySamples(perSec, durationMs))

	return Result{
		NetWPM:            netWPM,
		RawWPM:            rawWPM,
		Accuracy:          accuracy,
		KeystrokeAccuracy: keystrokeAccuracy,
		Consistency:       cons,
		CPS:               cps,
		CorrectChars:      correct,
		IncorrectChars:    incorrect,
		ExtraChars:        extra,
		Errors:            incorrect + blocked,
		DurationMs:        durationMs,
		PerSecond:         perSec,
		AFKTrimmed:        afkTrimmed,
		KeyMisses:         KeyHeatmap(log),
	}
}

// charResult holds the final correctness state for one target position.
type charResult struct {
	correct bool
}

// replayFinalState determines the final correctness of each target position
// from the buffer typing.ReplayBuffer reconstructs, plus the count of extra
// runes typed past the target (Target == 0 marks a position with nothing to
// match). The reconstruction itself lives in the typing package so that this
// and the UI's live replay cannot drift apart.
func replayFinalState(log []typing.Keystroke) ([]charResult, int) {
	var results []charResult
	extraTyped := 0
	for _, k := range typing.ReplayBuffer(log) {
		if k.Target == 0 {
			extraTyped++
			continue
		}
		results = append(results, charResult{correct: k.Correct})
	}
	return results, extraTyped
}
