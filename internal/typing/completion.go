package typing

import (
	"github.com/bavanchun/Typeburn/v2/internal/mode"
)

// Complete reports whether the test is finished according to the active mode.
//
//   - ModeTime:  the caller signals completion by passing nowMs >= the time
//     limit (limit, in milliseconds). The engine itself does not track wall
//     time; the caller (UI layer or test harness) drives the clock.
//   - ModeWords: complete when the user has typed exactly limit words.
//     A word is considered typed when the trailing space has been entered OR
//     when the last word in the sequence is fully typed (no trailing space needed).
//   - ModeQuote / ModeCode: complete when the typed buffer exactly matches
//     the full target (Code's target is the user-supplied snippet; literal
//     '\n'/'\t' are ordinary target runes).
func (e *Engine) Complete(nowMs int64) bool {
	switch e.mode {
	case mode.ModeTime:
		return nowMs >= int64(e.limit)

	case mode.ModeWords:
		return countCompletedWords(e.typed, e.target) >= e.limit

	case mode.ModeQuote, mode.ModeCode:
		return runesEqual(e.typed, e.target)

	default:
		return false
	}
}

// countCompletedWords counts how many words from the target the user has
// fully typed. A word is complete when:
//   - Its trailing space has been typed (mid-sequence words), OR
//   - It is the last word and the typed runes cover it entirely.
//
// Counting is position-based over the whole target: it measures how far the
// cursor has advanced, not whether the runes it passed were correct. Typing
// past the end of the target adds nothing, because there are no further target
// words to complete.
func countCompletedWords(typed, target []rune) int {
	if len(typed) == 0 || len(target) == 0 {
		return 0
	}

	completed := 0
	inWord := false

	for i, r := range target {
		if r == ' ' {
			if inWord {
				if i < len(typed) {
					completed++
				}
				inWord = false
			}
			continue
		}
		inWord = true
	}

	// Last word: complete if all its runes are typed (no trailing space required).
	if inWord && len(target) <= len(typed) {
		completed++
	}

	return completed
}

// countWords counts the space-separated words in a target text. It is the
// denominator Progress reports for a timed run, whose word count is a property
// of the generated text rather than a goal the user chose.
func countWords(target []rune) int {
	words := 0
	inWord := false
	for _, r := range target {
		if r == ' ' {
			inWord = false
			continue
		}
		if !inWord {
			words++
			inWord = true
		}
	}
	return words
}

// runesEqual reports whether a and b contain identical rune sequences.
func runesEqual(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
