package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// dismissHint is the trailing "how do I get rid of this" text. It is the first
// thing dropped when the notice does not fit, because dismissal is discoverable
// by pressing anything, while the reason is not discoverable at all.
const dismissHint = "  ·  press any key to dismiss"

// PersistenceNotice renders a single-line, dismissible toast shown when a run
// was not saved — a failed disk write, or a result deliberately withheld from
// history. RoleWarning message + RoleTextFaint dismiss hint, so it is legible
// under NO_COLOR (attribute-only) without shifting any layout.
//
// termW is the terminal width, not a suggestion. The caller places this line
// into a real frame, so a notice wider than the terminal widens every row
// around it. The full form runs to 78 cells, which spills at 60, 61 and 72
// columns; it is trimmed here rather than left to overflow. A termW of 0 means
// "unknown", and the notice is returned untrimmed.
//
// Empty msg yields an empty string (caller should not invoke it in that case,
// but guard anyway).
func PersistenceNotice(msg string, termW int, th theme.Theme) string {
	if msg == "" {
		return ""
	}

	body := "⚠ " + msg
	hint := dismissHint

	if termW > 0 {
		if lipgloss.Width(body)+lipgloss.Width(hint) > termW {
			hint = ""
		}
		body = truncateToCells(body, termW)
	}

	out := th.Style(theme.RoleWarning).Render(body)
	if hint != "" {
		out += th.Style(theme.RoleTextFaint).Render(hint)
	}
	return out
}

// truncateToCells shortens s to at most cells display cells, marking the cut
// with an ellipsis so a clipped reason does not read as the whole reason.
//
// Measurement is per rune in cells, never bytes: a message can carry a file
// path with wide characters in it.
func truncateToCells(s string, cells int) string {
	if cells <= 0 || lipgloss.Width(s) <= cells {
		return s
	}
	if cells == 1 {
		return "…"
	}

	budget := cells - 1 // one cell for the ellipsis
	used := 0
	var end int
	for i, r := range s {
		w := lipgloss.Width(string(r))
		if used+w > budget {
			end = i
			break
		}
		used += w
		end = i + len(string(r))
	}
	return s[:end] + "…"
}
