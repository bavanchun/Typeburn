package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// runeAtIndex returns the single display rune at index i: a target rune, then an
// extra typed rune past the target, then a blank for the trailing cursor cell.
func runeAtIndex(i int, target, typed []rune) rune {
	if i < len(target) {
		return target[i]
	}
	if i < len(typed) {
		return typed[i]
	}
	return ' '
}

// buildWordTokens renders one styled string per rune from its CharState. This is
// the single source of base styling shared by the static and animated renderers,
// so a settled animated frame is byte-identical to the static stream.
func buildWordTokens(states []typing.CharState, target, typed []rune, th theme.Theme) []string {
	stUntyped := th.Style(theme.RoleTextFaint)
	stCorrect := th.Style(theme.RoleTextMuted)
	stIncorrect := th.Style(theme.RoleError) // .Underline already in theme
	stIncorrectSpace := th.Style(theme.RoleErrorBg)
	stExtra := th.Style(theme.RoleError).Faint(true)
	stCurrent := th.Style(theme.RoleCursorBg)

	tokens := make([]string, len(states))
	for i := range states {
		r := runeAtIndex(i, target, typed)
		ch := string(r)

		var st lipgloss.Style
		switch states[i] {
		case typing.Correct:
			st = stCorrect
		case typing.Incorrect:
			st = stIncorrect
		case typing.IncorrectSpace:
			// Wrong char where a space was expected or vice-versa; show as a
			// background-highlighted block so the missed boundary is visible.
			if r == ' ' {
				ch = " "
			}
			st = stIncorrectSpace
		case typing.Extra:
			st = stExtra
		case typing.Current:
			// Block cursor: show a space if current position is at end of target.
			if r == ' ' {
				ch = " "
			}
			st = stCurrent
		default: // Untyped
			st = stUntyped
		}
		tokens[i] = st.Render(ch)
	}
	return tokens
}

// RenderWordStream renders the typing word-stream as a multi-line string.
//
// Each rune in target is styled according to its CharState. Extra typed runes
// (past the target length) are appended at the end. Rows are wrapped on word
// boundaries at `width` terminal cells (see wrapTokens). This is the static,
// animation-free, unwindowed render used by non-typing callers and golden
// tests; the typing screen goes through renderWordStreamAnim, which windows the
// rows to the height it has.
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
	tokens := buildWordTokens(states, target, typed, th)
	rows, _ := wrapTokens(tokens, states, target, typed, width)
	return strings.Join(rows, "\n")
}
