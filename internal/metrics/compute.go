package metrics

import (
	"github.com/bavanchun/Typeburn/internal/mode"
	"github.com/bavanchun/Typeburn/internal/typing"
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
	Errors         int // alias for IncorrectChars (uncorrected errors)

	DurationMs int64       // effective test duration (endMs - startMs, after AFK trim)
	PerSecond  []PerSecond // per-second breakdown

	KeyMisses []KeyMiss // per-key fumble tally (nil on empty/zero-duration logs)
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
func Compute(log []typing.Keystroke, mode mode.Mode, endMs int64) Result {
	if len(log) == 0 {
		return Result{Accuracy: 100, KeystrokeAccuracy: 100}
	}

	// Apply AFK trim (no-op for non-Time modes).
	log, endMs = TrimAFK(log, mode, endMs)

	// startMs = first keystroke time (first entry in log).
	startMs := log[0].TimeMs

	durationMs := endMs - startMs
	if durationMs <= 0 {
		return Result{Accuracy: 100, KeystrokeAccuracy: 100}
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

	// Total forward keystrokes (non-backspace) for RawWPM and CPS.
	var totalTyped int
	var correctForward int
	for _, k := range log {
		if k.Typed != 0 {
			totalTyped++
			if k.Correct {
				correctForward++
			}
		}
	}

	minutes := float64(durationMs) / 60000.0
	seconds := float64(durationMs) / 1000.0

	netWPM := float64(correct) / 5.0 / minutes
	rawWPM := float64(totalTyped) / 5.0 / minutes
	cps := float64(totalTyped) / seconds

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

	// Per-second buckets and consistency.
	perSec := bucketPerSecond(log, startMs)
	rawSamples := make([]float64, len(perSec))
	for i, ps := range perSec {
		rawSamples[i] = ps.RawWPM
	}
	cons := Consistency(rawSamples)

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
		Errors:            incorrect,
		DurationMs:        durationMs,
		PerSecond:         perSec,
		KeyMisses:         KeyHeatmap(log),
	}
}

// charResult holds the final correctness state for one target position.
type charResult struct {
	correct bool
}

// replayFinalState replays the keystroke log to determine the final typed rune
// at each target position. Backspace events (Typed == 0) pop the last position.
// Returns a slice of charResult indexed by target position, and the count of
// extra runes typed past the target.
func replayFinalState(log []typing.Keystroke) ([]charResult, int) {
	// We reconstruct the typed buffer step by step.
	type slot struct {
		typed   rune
		target  rune
		correct bool
		isExtra bool
	}
	var buf []slot

	for _, k := range log {
		if k.Typed == 0 {
			// Backspace: pop last slot.
			if len(buf) > 0 {
				buf = buf[:len(buf)-1]
			}
			continue
		}
		buf = append(buf, slot{
			typed:   k.Typed,
			target:  k.Target,
			correct: k.Correct,
			isExtra: k.Target == 0, // target==0 means extra (past end)
		})
	}

	var results []charResult
	extraTyped := 0
	for _, s := range buf {
		if s.isExtra {
			extraTyped++
		} else {
			results = append(results, charResult{correct: s.correct})
		}
	}
	return results, extraTyped
}
