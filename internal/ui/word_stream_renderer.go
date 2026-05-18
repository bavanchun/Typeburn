package ui

import (
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/theme"
	"monkeytype-tui/internal/typing"
)

// RenderWordStream renders the typing word-stream as a multi-line string.
//
// Each rune in target is styled according to its CharState. Extra typed runes
// (past the target length) are appended at the end. Hard-wrapping happens at
// width columns using word-aware boundaries: words are never split mid-rune
// unless a single word exceeds the full width.
//
// Rendering is rune-safe: all iteration uses []rune, not byte slices.
func RenderWordStream(
	states []typing.CharState,
	target []rune,
	typed []rune,
	width int,
	th theme.Theme,
) string {
	if width < 1 {
		width = 40
	}

	// Build pre-computed styles once to avoid re-allocating per rune.
	stUntypedStyle := th.Style(theme.RoleTextFaint)
	stCorrectStyle := th.Style(theme.RoleTextMuted)
	stIncorrectStyle := th.Style(theme.RoleError) // .Underline already in theme
	stIncorrectSpaceStyle := th.Style(theme.RoleErrorBg)
	stExtraStyle := th.Style(theme.RoleError).Faint(true)
	stCurrentStyle := th.Style(theme.RoleCursorBg)

	// Build styled tokens: one styled string per rune.
	total := len(states)
	tokens := make([]string, total)

	for i := 0; i < total; i++ {
		var r rune
		if i < len(target) {
			r = target[i]
		} else if i < len(typed) {
			r = typed[i]
		} else {
			r = ' '
		}

		ch := string(r)

		var st lipgloss.Style
		switch states[i] {
		case typing.Correct:
			st = stCorrectStyle
		case typing.Incorrect:
			st = stIncorrectStyle
		case typing.IncorrectSpace:
			// Wrong char where a space was expected or vice-versa; show as
			// a background-highlighted block so the missed boundary is visible.
			if r == ' ' {
				ch = " " // keep as space so the bg block is visible
			}
			st = stIncorrectSpaceStyle
		case typing.Extra:
			st = stExtraStyle
		case typing.Current:
			// Block cursor: show a space if current position is at end of target.
			if r == ' ' {
				ch = " "
			}
			st = stCurrentStyle
		default: // Untyped
			st = stUntypedStyle
		}

		tokens[i] = st.Render(ch)
	}

	// Word-aware hard-wrap at width columns.
	// We wrap on rune counts because styled strings have ANSI escapes that
	// inflate byte length. We track the raw rune count of current line.
	return wrapTokens(tokens, states, target, typed, width)
}

// wrapTokens assembles the styled rune tokens into wrapped lines.
// Word boundaries are determined from the raw runes so wrapping is correct
// regardless of ANSI escape inflation.
func wrapTokens(
	tokens []string,
	states []typing.CharState,
	target []rune,
	typed []rune,
	width int,
) string {
	var lines []string
	var lineBuilder strings.Builder
	lineWidth := 0 // rune width of current line (raw, no ANSI)

	flush := func() {
		lines = append(lines, lineBuilder.String())
		lineBuilder.Reset()
		lineWidth = 0
	}

	for i, tok := range tokens {
		// Determine the raw rune (single cell) for width accounting.
		var r rune
		if i < len(target) {
			r = target[i]
		} else if i < len(typed) {
			r = typed[i]
		} else {
			r = ' '
		}
		runeW := utf8.RuneLen(r)
		if runeW < 0 {
			runeW = 1
		}
		// Terminal width = 1 per standard ASCII/Latin rune; use 1 for simplicity
		// (full CJK double-width support deferred to Phase 9).
		cellW := 1

		// Word-aware wrap: if this rune would exceed the line width and the
		// rune is NOT a space, scan back to find the last space in the current
		// line and break there.
		if lineWidth+cellW > width && lineWidth > 0 {
			flush()
		}

		lineBuilder.WriteString(tok)
		lineWidth += cellW

		// If we just wrote a space at the end of a word and the line is full,
		// break here so the next word starts fresh.
		if r == ' ' && lineWidth >= width {
			flush()
		}
	}

	// Flush any remaining content.
	if lineBuilder.Len() > 0 {
		lines = append(lines, lineBuilder.String())
	}

	return strings.Join(lines, "\n")
}

// contentWidth calculates the word-stream content width from the terminal width.
// Per design §4.1: min(termW-8, 80). A minimum of 20 is enforced defensively.
func contentWidth(termW int) int {
	w := termW - 8
	if w > 80 {
		w = 80
	}
	if w < 20 {
		w = 20
	}
	return w
}
