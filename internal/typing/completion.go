package typing

import (
	"monkeytype-tui/internal/config"
)

// Complete reports whether the test is finished according to the active mode.
//
//   - ModeTime:  the caller signals completion by passing nowMs >= the time limit
//     (stored in wordTarget as milliseconds). The engine itself does not track
//     wall time; the caller (UI layer or test harness) drives the clock.
//   - ModeWords: complete when the user has typed exactly wordTarget words.
//     A word is considered typed when the trailing space has been entered OR
//     when the last word in the sequence is fully typed (no trailing space needed).
//   - ModeQuote: complete when the typed buffer exactly matches the full target.
func (e *Engine) Complete(nowMs int64) bool {
	switch e.mode {
	case config.ModeTime:
		return nowMs >= int64(e.wordTarget)

	case config.ModeWords:
		return countCompletedWords(e.typed, e.target) >= e.wordTarget

	case config.ModeQuote:
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
// Only the first wordTarget words are considered; extra typed runes beyond the
// target are ignored for counting purposes.
func countCompletedWords(typed, target []rune) int {
	if len(typed) == 0 || len(target) == 0 {
		return 0
	}

	completed := 0
	inWord := false
	wordStart := 0

	for i, r := range target {
		if r == ' ' {
			if inWord {
				// Check if this word AND its space are covered by typed.
				// Space position is i; we need typed[i] to exist and be a space.
				if i < len(typed) {
					completed++
				}
				inWord = false
			}
		} else {
			if !inWord {
				wordStart = i
				inWord = true
			}
		}
		_ = wordStart
	}

	// Last word: complete if all its runes are typed (no trailing space required).
	if inWord {
		// Find the end of the last word in target.
		lastWordEnd := len(target)
		if lastWordEnd <= len(typed) {
			completed++
		}
	}

	return completed
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
