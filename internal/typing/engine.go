package typing

import (
	"monkeytype-tui/internal/config"
)

// Keystroke records a single typed or deleted character event.
// Target is 0 for Extra positions (past end of target text).
// Correct reflects whether the typed rune matched the target at that moment.
// For backspace events, Typed is set to 0 (null rune) and Correct is false.
type Keystroke struct {
	TimeMs  int64
	Typed   rune
	Target  rune
	Correct bool
}

// Engine maintains the mutable typing state: target buffer, typed buffer,
// keystroke log, and mode metadata. All rune operations are rune-safe
// (no byte indexing), making Engine correct for multi-byte Unicode input
// such as "café" or CJK characters.
type Engine struct {
	target     []rune
	typed      []rune
	log        []Keystroke
	startMs    int64 // 0 until first Apply call
	mode       config.Mode
	wordTarget int // Words: N words to complete; Time: limit in ms; Quote: unused (0)
}

// New creates an Engine for the given target text, mode, and wordTarget.
// For ModeWords, wordTarget is the number of words to type.
// For ModeTime, wordTarget is the time limit in milliseconds.
// For ModeQuote, wordTarget is ignored (0).
func New(target string, mode config.Mode, wordTarget int) *Engine {
	return &Engine{
		target:     []rune(target),
		mode:       mode,
		wordTarget: wordTarget,
	}
}

// StartMs returns the timestamp of the first Apply call, or 0 if no key has
// been pressed yet. Callers use this to compute test duration.
func (e *Engine) StartMs() int64 { return e.startMs }

// Apply records a printable rune keystroke at the given monotonic millisecond
// timestamp. The first call sets startMs. Extra runes past the target length
// are appended and classified as Extra by States().
func (e *Engine) Apply(r rune, nowMs int64) {
	if e.startMs == 0 {
		e.startMs = nowMs
	}

	pos := len(e.typed)
	var target rune
	var correct bool

	if pos < len(e.target) {
		target = e.target[pos]
		correct = (r == target)
	}
	// pos >= len(e.target): extra rune — target stays 0, correct stays false

	e.typed = append(e.typed, r)
	e.log = append(e.log, Keystroke{
		TimeMs:  nowMs,
		Typed:   r,
		Target:  target,
		Correct: correct,
	})
}

// Backspace removes the last typed rune and appends a deletion marker to the
// log (Typed=0, Correct=false). It is a no-op when the typed buffer is empty.
func (e *Engine) Backspace(nowMs int64) {
	if len(e.typed) == 0 {
		return
	}
	if e.startMs == 0 {
		e.startMs = nowMs
	}
	pos := len(e.typed) - 1
	var target rune
	if pos < len(e.target) {
		target = e.target[pos]
	}
	e.typed = e.typed[:pos]
	e.log = append(e.log, Keystroke{
		TimeMs:  nowMs,
		Typed:   0,
		Target:  target,
		Correct: false,
	})
}

// States returns a CharState slice covering all target positions plus any extra
// typed runes. The current cursor position is marked Current; positions behind
// it are Correct/Incorrect/IncorrectSpace; positions ahead are Untyped.
//
// IncorrectSpace is assigned when the target rune is a space and the typed rune
// is not (or vice versa), making word-boundary errors visually distinct.
func (e *Engine) States() []CharState {
	cursor := len(e.typed)
	total := len(e.target)
	if cursor > total {
		total = cursor
	}

	states := make([]CharState, total)

	for i := 0; i < total; i++ {
		switch {
		case i == cursor:
			states[i] = Current
		case i > cursor:
			states[i] = Untyped
		case i >= len(e.target):
			// typed past end of target → Extra
			states[i] = Extra
		default:
			typed := e.typed[i]
			tgt := e.target[i]
			if typed == tgt {
				states[i] = Correct
			} else if tgt == ' ' || typed == ' ' {
				// wrong char at a space boundary — distinct visual class
				states[i] = IncorrectSpace
			} else {
				states[i] = Incorrect
			}
		}
	}

	return states
}

// Log returns the full keystroke log in chronological order.
// Backspace events have Typed==0.
func (e *Engine) Log() []Keystroke {
	out := make([]Keystroke, len(e.log))
	copy(out, e.log)
	return out
}

// Progress returns (done, total) for the current mode:
//   - ModeWords / ModeTime: (completedWords, wordTarget)
//   - ModeQuote:            (typedRunes, totalTargetRunes)
func (e *Engine) Progress() (done, total int) {
	switch e.mode {
	case config.ModeQuote:
		typed := len(e.typed)
		if typed > len(e.target) {
			typed = len(e.target)
		}
		return typed, len(e.target)
	default:
		// Words and Time both report word progress toward wordTarget.
		return countCompletedWords(e.typed, e.target), e.wordTarget
	}
}
