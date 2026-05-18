package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

// View renders the Home screen as a centered block string. HomeModel does its
// own placement (mirrors the typing screen); the root returns it directly.
func (m HomeModel) View() string {
	logo := RenderLogo(m.w, m.th)
	tabs := m.renderTabs()
	opts := m.renderOptions()
	hint := m.th.Style(theme.RoleTextPrimary).Bold(true).Render("press enter to start")
	footer := RenderFooter(homeHints(), m.w, m.th)

	// Spacer between the content block and the pinned footer.
	used := strings.Count(logo, "\n") + 1 // logo lines
	used += 2                             // blank after logo
	used++                                // tabs
	used++                                // blank after tabs
	used++                                // options
	used += 2                             // blanks before hint
	used++                                // hint
	used++                                // footer
	spacerH := m.h - used
	if spacerH < 1 {
		spacerH = 1
	}

	var b strings.Builder
	b.WriteString(logo)
	b.WriteString("\n\n")
	b.WriteString(tabs)
	b.WriteString("\n\n")
	b.WriteString(opts)
	b.WriteString("\n\n")
	b.WriteString(hint)
	b.WriteString("\n")
	b.WriteString(strings.Repeat("\n", spacerH-1))
	b.WriteString(footer)

	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, b.String())
	}
	return b.String()
}

// renderTabs builds the mode-selector tab row.
func (m HomeModel) renderTabs() string {
	return RenderTabs(modeLabels, m.modeIdx, m.th)
}

// renderOptions builds the length-option row for the current mode.
func (m HomeModel) renderOptions() string {
	mode := m.currentMode()
	chosen := m.lenIdx[mode]
	if mode == config.ModeQuote {
		return RenderOptions(quoteBucketLabels, chosen, m.th)
	}
	lens := config.LengthsFor(mode)
	labels := make([]string, len(lens))
	for i, v := range lens {
		labels[i] = fmt.Sprintf("%d", v)
	}
	return RenderOptions(labels, chosen, m.th)
}

// homeHints returns the footer hint set for the Home screen per mockups §1.
func homeHints() []Hint {
	return []Hint{
		{Key: "tab", Action: "mode"},
		{Key: "←→", Action: "length"},
		{Key: "enter", Action: "start"},
		{Key: "2", Action: "settings"},
		{Key: "3", Action: "history"},
		{Key: "ctrl+c", Action: "quit"},
	}
}
