package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// Hint is a single key+action pair rendered in the footer.
type Hint struct {
	Key    string
	Action string
}

// narrowFooterW is the width below which the footer is glyphs-only by design
// (TierNarrow in width_tier.go).
const narrowFooterW = 72

// RenderFooter renders a key-hint footer bar per design §5.4.
//
// Two forms: full ("tab restart · ctrl+r new · esc menu") and glyphs-only
// ("tab · ctrl+r · esc"). Key glyphs are text-muted (scannable); action words
// and the separator are text-faint.
//
// Below narrowFooterW the glyph form is the design. At or above it the full
// form is *measured* rather than assumed to fit: the six-hint Home footer is 73
// cells, which spills a 72-column terminal. That is not a cosmetic overflow —
// every screen is centred with lipgloss.Place, which sizes from the widest
// line, so one over-wide footer shifts the whole screen off centre.
func RenderFooter(hints []Hint, termW int, th theme.Theme) string {
	sep := th.Style(theme.RoleTextFaint).Render(" · ")

	if termW <= 0 || termW >= narrowFooterW {
		full := joinHints(hints, false, th, sep)
		if termW <= 0 || lipgloss.Width(full) <= termW {
			return full
		}
	}
	return joinHints(hints, true, th, sep)
}

// joinHints renders the hint list in one of the two forms. A hint with no
// action word renders as its key in both.
func joinHints(hints []Hint, glyphsOnly bool, th theme.Theme, sep string) string {
	keyStyle := th.Style(theme.RoleTextMuted)
	actionStyle := th.Style(theme.RoleTextFaint)

	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		key := keyStyle.Render(h.Key)
		if glyphsOnly || h.Action == "" {
			parts = append(parts, key)
		} else {
			parts = append(parts, key+" "+actionStyle.Render(h.Action))
		}
	}
	return strings.Join(parts, sep)
}

// TypingHints returns the standard hint set for the typing test screen.
func TypingHints() []Hint {
	return []Hint{
		{Key: "tab", Action: "restart"},
		{Key: "ctrl+r", Action: "new"},
		{Key: "esc", Action: "menu"},
	}
}
