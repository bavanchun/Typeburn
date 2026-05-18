package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// RenderWordStream renders the typing word-stream as a multi-line string.
//
// Each rune in target is styled according to its CharState. Extra typed runes
// (past the target length) are appended at the end. Wrapping is a hard
// character-cell wrap at `width`: when the next rune would overflow the line
// it starts a new line, so a word longer than `width` IS split between runes
// (never within a multi-byte rune). A space landing at/after the boundary
// also flushes so the following word begins on a fresh line. This is not a
// word-aware (scan-back) wrap.
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

	// Hard character-cell wrap at width columns. We wrap on rune counts (not
	// byte length) because styled tokens carry ANSI escapes that inflate bytes;
	// the raw rune count of the current line is tracked instead.
	return wrapTokens(tokens, states, target, typed, width)
}

// wrapTokens assembles the styled rune tokens into wrapped lines using a hard
// per-cell wrap. Width is accounted from the raw runes (one cell each) so the
// ANSI escapes in the styled tokens do not distort line length.
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
		// One terminal cell per rune (ASCII/Latin). CJK double-width is not
		// handled — deferred (roadmap m5, "CJK width support if quotes added").
		cellW := 1

		// Hard wrap: if this rune would overflow the line, break before it.
		// There is no scan-back to the last space, so a word wider than the
		// line is split between runes here.
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
