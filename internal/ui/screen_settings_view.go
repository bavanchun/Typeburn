package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/theme"
)

// View renders the Settings screen as a centered block string.
// Layout per mockups §4: title, separator, 4 rows, separator, help line, footer.
// Degraded mode (w<60 or h<20) is handled by the root View; this is only called
// when the terminal meets the safe minimum.
func (m SettingsModel) View() string {
	title := m.th.Style(theme.RoleAccent).Bold(true).Render("S E T T I N G S")
	sep := m.th.Style(theme.RoleBorder).Render(strings.Repeat("─", 44))

	rows := m.renderRows()
	help := m.th.Style(theme.RoleTextFaint).Render(m.rows[m.sel].help)
	footer := RenderFooter(settingsHints(), m.w, m.th)

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n\n")
	b.WriteString(rows)
	b.WriteString("\n")
	b.WriteString(sep)
	b.WriteString("\n")
	b.WriteString(help)

	content := b.String()

	// Pin footer to the bottom.
	contentLines := strings.Count(content, "\n") + 1
	used := contentLines + 1 + 1 // content + blank + footer
	spacer := m.h - used
	if spacer < 1 {
		spacer = 1
	}

	var full strings.Builder
	full.WriteString(content)
	full.WriteString(strings.Repeat("\n", spacer))
	full.WriteString(footer)

	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, full.String())
	}
	return full.String()
}

// renderRows renders all 4 settings rows joined by newlines.
func (m SettingsModel) renderRows() string {
	parts := make([]string, len(m.rows))
	for i, row := range m.rows {
		parts[i] = m.renderRow(i, row)
	}
	return strings.Join(parts, "\n")
}

// renderRow renders a single settings row. Selected rows get the ▎ accent bar,
// surface-alt background, bold primary label and accent bold value.
// Unselected rows use text-muted for both label and value.
func (m SettingsModel) renderRow(idx int, row settingRow) string {
	val := row.values[row.idx]
	display := "‹ " + val + " ›"

	if idx == m.sel {
		// Selected: ▎ bar + surface-alt bg + bold label + accent bold value.
		bar := m.th.Style(theme.RoleAccent).Bold(true).Render("▎")
		labelStyle := m.th.Style(theme.RoleTextPrimary).Bold(true)
		valStyle := m.th.Style(theme.RoleAccent).Bold(true)
		bgStyle := lipgloss.NewStyle().Background(m.th.Color(theme.RoleSurfaceAlt))

		label := bgStyle.Render(labelStyle.Render(row.label))
		value := bgStyle.Render(valStyle.Render(display))

		// Pad the gap between label and value.
		gap := 44 - len(row.label) - len(display)
		if gap < 2 {
			gap = 2
		}
		return bar + " " + label + strings.Repeat(" ", gap) + value
	}

	// Unselected: plain text-muted for both parts.
	labelStyle := m.th.Style(theme.RoleTextMuted)
	valStyle := m.th.Style(theme.RoleTextMuted)

	gap := 44 - len(row.label) - len(display)
	if gap < 2 {
		gap = 2
	}
	return "  " + labelStyle.Render(row.label) + strings.Repeat(" ", gap) + valStyle.Render(display)
}

// settingsHints returns the footer hint set for the Settings screen per §8.5.
func settingsHints() []Hint {
	return []Hint{
		{Key: "↑↓", Action: "move"},
		{Key: "←→", Action: "change"},
		{Key: "esc", Action: "save & back"},
		{Key: "ctrl+c", Action: "quit"},
	}
}
