package ui

import (
	"strings"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// Hint is a single key+action pair rendered in the footer.
type Hint struct {
	Key    string
	Action string
}

// RenderFooter renders a key-hint footer bar per design §5.4.
//
// Full form: "tab restart · ctrl+r new · esc menu"
// Key glyphs: text-muted (scannable). Action words and separator: text-faint.
// When termW < 72, action words are dropped and only key glyphs are shown.
func RenderFooter(hints []Hint, termW int, th theme.Theme) string {
	keyStyle := th.Style(theme.RoleTextMuted)
	actionStyle := th.Style(theme.RoleTextFaint)
	sepStyle := th.Style(theme.RoleTextFaint)

	narrow := termW > 0 && termW < 72
	sep := sepStyle.Render(" · ")

	parts := make([]string, 0, len(hints))
	for _, h := range hints {
		key := keyStyle.Render(h.Key)
		if narrow || h.Action == "" {
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
