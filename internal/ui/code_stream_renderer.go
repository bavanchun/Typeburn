package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// tabVisualWidth is how many columns a literal '\t' occupies on screen. The
// user still presses a single Tab key to match the single '\t' target rune;
// this is display-only (locked decision: 2 columns).
const tabVisualWidth = 2

// RenderCodeStream renders the Code-mode buffer with FULL-LITERAL layout:
// literal '\n' starts a new row, '\t' shows as tabVisualWidth columns, and
// long rows hard-wrap at width (no horizontal scroll). A vertical viewport of
// at most height rows is returned, always containing the caret row
// (scroll-follow). Per-rune styling reuses the same CharState→Role mapping as
// the word stream; that tiny switch is duplicated here deliberately so the
// golden-tested word_stream_renderer.go stays untouched (KISS over forced
// DRY — same call as the v1.1.0 renderer decision).
//
// It is rune-safe and NO_COLOR-safe (the no-color theme only swaps attributes
// for colors, so row/line structure is identical).
func RenderCodeStream(
	states []typing.CharState,
	target []rune,
	typed []rune,
	width, height int,
	th theme.Theme,
) string {
	return renderCodeStreamAnim(states, target, typed, width, height, th, disabledCaret())
}

// renderCodeStreamAnim is RenderCodeStream with caret animation applied. The
// cursor blink and the ≤2 cells behind it get their animated styles; with a
// disabled caret (cursorIdx < 0) the output equals RenderCodeStream exactly, so
// existing goldens hold. Code mode has no token cache (its viewport recompute is
// the pre-existing cost); only the ≤3 animated cells differ per frame.
func renderCodeStreamAnim(
	states []typing.CharState,
	target []rune,
	typed []rune,
	width, height int,
	th theme.Theme,
	ca caretAnim,
) string {
	if width < 1 {
		width = 40
	}
	if height < 1 {
		height = 20
	}

	stUntyped := th.Style(theme.RoleTextFaint)
	stCorrect := th.Style(theme.RoleTextMuted)
	stIncorrect := th.Style(theme.RoleError)
	stIncorrectSpace := th.Style(theme.RoleErrorBg)
	stExtra := th.Style(theme.RoleError).Faint(true)
	stCurrent := th.Style(theme.RoleCursorBg)

	styleFor := func(s typing.CharState) lipgloss.Style {
		switch s {
		case typing.Correct:
			return stCorrect
		case typing.Incorrect:
			return stIncorrect
		case typing.IncorrectSpace:
			return stIncorrectSpace
		case typing.Extra:
			return stExtra
		case typing.Current:
			return stCurrent
		default:
			return stUntyped
		}
	}

	var rows []string
	var row strings.Builder
	rowW := 0
	caretRow := -1

	flush := func() {
		rows = append(rows, row.String())
		row.Reset()
		rowW = 0
	}

	for i := 0; i < len(states); i++ {
		var r rune
		switch {
		case i < len(target):
			r = target[i]
		case i < len(typed):
			r = typed[i]
		default:
			r = ' '
		}
		st := styleFor(states[i])
		// Caret animation: override the cursor cell and the ≤2 cells behind it.
		if ca.cursorIdx >= 0 {
			if d := ca.cursorIdx - i; d >= 0 && d <= 2 {
				if ast, ok := ca.caretCellStyle(d, states[i], th); ok {
					st = ast
				}
			}
		}
		if states[i] == typing.Current {
			caretRow = len(rows) // caret lives in the row being built now
		}

		if r == '\n' {
			if states[i] == typing.Current {
				row.WriteString(st.Render(" ")) // caret visible at line end
			}
			flush()
			continue
		}

		cellW := 1
		text := string(r)
		if r == '\t' {
			cellW = tabVisualWidth
			text = strings.Repeat(" ", tabVisualWidth)
		}
		// Hard-wrap continuation (not a logical newline) to avoid horizontal
		// overflow; a token wider than an empty row is still placed.
		if rowW > 0 && rowW+cellW > width {
			flush()
			if states[i] == typing.Current {
				caretRow = len(rows)
			}
		}
		row.WriteString(st.Render(text))
		rowW += cellW
	}
	// Always flush the final (possibly empty) in-progress row, and guarantee
	// at least one row for non-empty input.
	if row.Len() > 0 || len(rows) == 0 {
		flush()
	}

	return joinViewport(rows, caretRow, height)
}

// joinViewport returns at most height rows, scrolled so caretRow is visible.
// Fewer rows than height → all rows, unscrolled.
func joinViewport(rows []string, caretRow, height int) string {
	if len(rows) <= height {
		return strings.Join(rows, "\n")
	}
	if caretRow < 0 {
		caretRow = len(rows) - 1
	}
	start := 0
	if caretRow >= height {
		start = caretRow - height + 1
	}
	if max := len(rows) - height; start > max {
		start = max
	}
	if start < 0 {
		start = 0
	}
	return strings.Join(rows[start:start+height], "\n")
}
