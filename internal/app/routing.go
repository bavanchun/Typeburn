package app

import (
	"strings"

	"monkeytype-tui/internal/theme"
)

// screenTitle is the human label for a screen, used by the placeholder views
// (and a convenient single place to keep names consistent).
func screenTitle(s Screen) string {
	switch s {
	case ScreenTyping:
		return "Typing"
	case ScreenResult:
		return "Result"
	case ScreenSettings:
		return "Settings"
	case ScreenHistory:
		return "History"
	default:
		return "Home"
	}
}

// placeholderView renders a minimal, themed stand-in for a screen. Each real
// screen replaces this in its own phase; the skeleton only proves routing,
// theming, and the global key handling work.
func placeholderView(s Screen, th theme.Theme) string {
	title := th.Style(theme.RoleAccent).Bold(true).Render("monkeytype-tui")
	screen := th.Style(theme.RoleTextPrimary).Render("[ " + screenTitle(s) + " ]")
	hint := th.Style(theme.RoleTextFaint).Render(
		"1 home · 2 settings · 3 history · esc back · ctrl+c quit")

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(screen)
	b.WriteString("\n\n")
	b.WriteString(hint)
	return b.String()
}
