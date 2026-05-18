package ui

import (
	"strings"

	"monkeytype-tui/internal/theme"
)

// RenderTabs renders a horizontal row of mode-selector tabs per design §5.5/§6.
//
// Active tab: accent + Bold + "▎" left bar. Inactive tabs: text-muted.
// Tabs are space-separated; suitable for the Home mode row.
func RenderTabs(opts []string, active int, th theme.Theme) string {
	accentStyle := th.Style(theme.RoleAccent).Bold(true)
	mutedStyle := th.Style(theme.RoleTextMuted)
	bar := accentStyle.Render("▎")

	parts := make([]string, len(opts))
	for i, opt := range opts {
		if i == active {
			parts[i] = bar + " " + accentStyle.Render(opt)
		} else {
			parts[i] = "  " + mutedStyle.Render(opt)
		}
	}
	return strings.Join(parts, "   ")
}

// RenderOptions renders a horizontal row of length/bucket option values per
// design §6 (Mode option row).
//
// Chosen option: accent Bold with [brackets]. Siblings: text-muted.
// Items are space-separated with padding to align evenly.
func RenderOptions(opts []string, chosen int, th theme.Theme) string {
	accentStyle := th.Style(theme.RoleAccent).Bold(true)
	mutedStyle := th.Style(theme.RoleTextMuted)

	parts := make([]string, len(opts))
	for i, opt := range opts {
		if i == chosen {
			parts[i] = accentStyle.Render("[" + opt + "]")
		} else {
			parts[i] = mutedStyle.Render(opt)
		}
	}
	return strings.Join(parts, "    ")
}
