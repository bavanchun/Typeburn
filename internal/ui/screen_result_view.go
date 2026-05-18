package ui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
)

// resultHints returns the footer hint set for the result screen per mockups §3.
func resultHints() []Hint {
	return []Hint{
		{Key: "tab", Action: "restart"},
		{Key: "ctrl+r", Action: "new"},
		{Key: "esc", Action: "menu"},
		{Key: "3", Action: "history"},
	}
}

// View renders the result screen. It places a single rounded-border panel
// (RoleBorder, surface bg) with title "result" on the top border edge.
// Layout mirrors mockups §3.
func (m ResultModel) View() string {
	footer := RenderFooter(resultHints(), m.w, m.th)

	panel := m.renderPanel()

	// Vertical padding: pin footer to bottom.
	panelLines := strings.Count(panel, "\n") + 1
	used := panelLines + 1 + 1 // panel + blank + footer
	spacer := m.h - used
	if spacer < 1 {
		spacer = 1
	}

	var b strings.Builder
	b.WriteString(panel)
	b.WriteString(strings.Repeat("\n", spacer))
	b.WriteString(footer)

	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, b.String())
	}
	return b.String()
}

// renderPanel builds the rounded-border result panel with all content sections.
func (m ResultModel) renderPanel() string {
	// Panel content width: terminal width minus outer margins and border chars.
	panelW := m.w - 8
	if panelW < 40 {
		panelW = 40
	}
	innerW := panelW - 4 // account for "│  " left and "  │" right padding

	var inner strings.Builder
	inner.WriteString(m.renderHero(innerW))
	inner.WriteString("\n\n")
	inner.WriteString(m.renderSparkline(innerW))
	inner.WriteString("\n\n")
	inner.WriteString(m.renderCharStats())
	inner.WriteString("\n")
	inner.WriteString(m.renderMeta())

	// Build bordered panel, then inject "result" title on the top border line.
	borderColor := m.th.Color(theme.RoleBorder)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(panelW)

	panel := borderStyle.Render(inner.String())
	titleStyled := m.th.Style(theme.RoleTextMuted).Render(" result ")
	return injectBorderTitle(panel, titleStyled)
}

// renderHero renders the big-digit WPM block alongside acc/raw/consistency.
func (m ResultModel) renderHero(innerW int) string {
	bigWPM := BigDigits(int(math.Round(m.res.NetWPM)), m.th)

	wpmLabel := m.th.Style(theme.RoleTextMuted).Render("wpm")
	if m.isBest {
		wpmLabel += m.th.Style(theme.RoleSuccess).Render(" ★ new best")
	}

	accLine := StatCard("acc", fmt.Sprintf("%.0f%%", m.res.Accuracy), accColorRole(m.res.Accuracy), m.th)
	rawLine := StatCard("raw", fmt.Sprintf("%.0f wpm", m.res.RawWPM), theme.RoleTextPrimary, m.th)
	consLine := StatCard("consistency", fmt.Sprintf("%.0f%%", m.res.Consistency), theme.RoleTextPrimary, m.th)
	secondaryCol := strings.Join([]string{accLine, rawLine, consLine}, "\n")

	bigLines := strings.Split(bigWPM+"\n"+wpmLabel, "\n")
	secLines := strings.Split(secondaryCol, "\n")
	for len(bigLines) < len(secLines) {
		bigLines = append(bigLines, "")
	}
	for len(secLines) < len(bigLines) {
		secLines = append(secLines, "")
	}

	rows := make([]string, len(bigLines))
	for i := range bigLines {
		sec := ""
		if i < len(secLines) {
			sec = secLines[i]
		}
		rows[i] = bigLines[i] + "   " + sec
	}
	_ = innerW // reserved for future width-aware truncation
	return strings.Join(rows, "\n")
}

// renderSparkline renders the "wpm over time" sparkline section.
func (m ResultModel) renderSparkline(innerW int) string {
	header := m.th.Style(theme.RoleTextMuted).Render("wpm over time")

	vals := make([]float64, len(m.res.PerSecond))
	for i, ps := range m.res.PerSecond {
		vals[i] = ps.RawWPM
	}

	graph := Sparkline(vals, innerW, 3, m.th)
	if graph == "" {
		graph = m.th.Style(theme.RoleTextFaint).Render("(no data)")
	}
	return header + "\n" + graph
}

// renderCharStats renders the correct/incorrect/extra/missed counts line.
func (m ResultModel) renderCharStats() string {
	labelStyle := m.th.Style(theme.RoleTextMuted)
	valueStyle := m.th.Style(theme.RoleTextPrimary).Bold(true)
	errStyle := m.th.Style(theme.RoleError).Bold(true)

	incVal := valueStyle.Render(fmt.Sprintf("%d", m.res.IncorrectChars))
	if m.res.IncorrectChars > 0 {
		incVal = errStyle.Render(fmt.Sprintf("%d", m.res.IncorrectChars))
	}

	parts := []string{
		labelStyle.Render("correct") + " " + valueStyle.Render(fmt.Sprintf("%d", m.res.CorrectChars)),
		labelStyle.Render("incorrect") + " " + incVal,
		labelStyle.Render("extra") + " " + valueStyle.Render(fmt.Sprintf("%d", m.res.ExtraChars)),
		labelStyle.Render("missed") + " " + valueStyle.Render(fmt.Sprintf("%d", m.res.MissedChars)),
	}
	return strings.Join(parts, "   ")
}

// renderMeta renders the duration · mode length · english line.
func (m ResultModel) renderMeta() string {
	style := m.th.Style(theme.RoleTextFaint)
	dur := fmt.Sprintf("%.0fs", float64(m.res.DurationMs)/1000.0)
	return style.Render(dur + " · " + modeMetaLabel(m.mode, m.length) + " · english")
}
