package typing

import (
	"github.com/bavanchun/Typeburn/v2/internal/mode"
)

// Keystroke records a single typed or deleted character event.
// Target is 0 for Extra positions (past end of target text).
// Correct reflects whether the typed rune matched the target at that moment.
// For backspace events, Typed is set to 0 (null rune) and Correct is false.
//
// Blocked marks a keystroke that was recorded but refused: in strict mode a
// wrong key does not advance the cursor, so the rune never entered the typed
// buffer. It is omitted from JSON when false so logs written before the field
// existed decode to the non-strict meaning they were recorded with.
type Keystroke struct {
	TimeMs  int64 `json:"time_ms"`
	Typed   rune  `json:"typed"`
	Target  rune  `json:"target"`
	Correct bool  `json:"correct"`
	Blocked bool  `json:"blocked,omitempty"`
}

// Engine maintains the mutable typing state: target buffer, typed buffer,
// keystroke log, and mode metadata. All rune operations are rune-safe
// (no byte indexing), making Engine correct for multi-byte Unicode input
// such as "café" or CJK characters.
type Engine struct {
	target  []rune
	typed   []rune
	log     []Keystroke
	startMs int64 // 0 until first Apply call
	mode    mode.Mode

	// limit is what ends the run, and its unit depends on the mode: a word
	// count for ModeWords, a duration in milliseconds for ModeTime, unused (0)
	// for ModeQuote and ModeCode. It is deliberately not named for either unit,
	// because reading it as the other one is how a 30-second run came to report
	// its progress as 0 out of 30000 words.
	limit  int
	strict bool
}

// New creates an Engine for the given target text, mode, and limit.
// For ModeWords, limit is the number of words to type.
// For ModeTime, limit is the time limit in milliseconds.
// For ModeQuote and ModeCode, limit is ignored (0).
func New(target string, mode mode.Mode, limit int) *Engine {
	return NewStrict(target, mode, limit, false)
}

// NewStrict creates an Engine with optional strict (stop-on-error letter) mode.
func NewStrict(target string, mode mode.Mode, limit int, strict bool) *Engine {
	return &Engine{
		target: []rune(target),
		mode:   mode,
		limit:  limit,
		strict: strict,
	}
}

// StartMs returns the timestamp of the first Apply call, or 0 if no key has
// been pressed yet. Callers use this to compute test duration.
func (e *Engine) StartMs() int64 { return e.startMs }

// Apply records a printable rune keystroke at the given monotonic millisecond
// timestamp. The first call sets startMs. Extra runes past the target length
// are appended and classified as Extra by States().
//
// In strict mode no wrong rune advances the cursor — including one typed past
// the end of the target, which is wrong by definition since there is nothing
// left to match. Such a keystroke is logged as Blocked so replays agree with
// the buffer this engine actually holds.
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

	if e.strict && !correct {
		e.log = append(e.log, Keystroke{
			TimeMs:  nowMs,
			Typed:   r,
			Target:  target,
			Correct: false,
			Blocked: true,
		})
		return
	}

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

// Log returns the full keystroke log in chronological order.
// Backspace events have Typed==0.
func (e *Engine) Log() []Keystroke {
	out := make([]Keystroke, len(e.log))
	copy(out, e.log)
	return out
}

// Typed returns the current typed buffer after applying backspaces.
func (e *Engine) Typed() []rune {
	out := make([]rune, len(e.typed))
	copy(out, e.typed)
	return out
}

// ForwardKeystrokes returns how many non-backspace events are in the log.
func (e *Engine) ForwardKeystrokes() int {
	n := 0
	for _, k := range e.log {
		if k.Typed != 0 {
			n++
		}
	}
	return n
}
