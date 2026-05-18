// Package typing implements the pure, UI-free typing engine and keystroke log.
// It has zero UI or Bubble Tea dependencies — all state is derived from rune
// slices and monotonic millisecond timestamps supplied by the caller.
package typing

// CharState classifies the display state of each position in the target text.
//
// NOTE: "current-error" is NOT used in v1. The engine operates in allow-continue
// mode only: the user may keep typing past an error without being blocked.
// Stop-on-error would require a separate state; that is deferred to a future version.
type CharState int

const (
	// Untyped means the position has not yet been reached by the cursor.
	Untyped CharState = iota
	// Correct means the typed rune matches the target rune at this position.
	Correct
	// Incorrect means a non-space rune was typed where a non-space was expected,
	// and it does not match.
	Incorrect
	// IncorrectSpace means a non-space rune was typed where the target is a space,
	// or a space was typed where the target is a non-space. The visual distinction
	// lets the UI highlight word-boundary errors differently.
	IncorrectSpace
	// Extra means a rune was typed past the end of the target text (overflow).
	Extra
	// Current marks the position the cursor is currently at (next to be typed).
	Current
)
