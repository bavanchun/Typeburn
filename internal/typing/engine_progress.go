package typing

import (
	"github.com/bavanchun/Typeburn/v2/internal/mode"
)

// Progress returns how far the run has got and how far it goes, in one unit per
// mode. Both halves are always the same unit and total is never zero while
// there is something to type — a caller rendering "done / total" or a
// percentage has no way to detect a denominator that means something else.
//
//   - ModeWords:            (completed words, target word count)
//   - ModeTime:             (completed words, words in the generated text)
//   - ModeQuote / ModeCode: (typed runes, target runes)
//
// ModeTime cannot report the engine's limit: for a timed run that field holds
// the deadline in milliseconds, which Complete needs and which is not a count
// of anything the user types.
func (e *Engine) Progress() (done, total int) {
	switch e.mode {
	case mode.ModeQuote, mode.ModeCode:
		typed := len(e.typed)
		if typed > len(e.target) {
			typed = len(e.target)
		}
		return typed, len(e.target)

	case mode.ModeTime:
		return countCompletedWords(e.typed, e.target), countWords(e.target)

	default:
		return countCompletedWords(e.typed, e.target), e.limit
	}
}
