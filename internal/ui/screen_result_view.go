package ui

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// validSemverFooter rejects any version string that could carry ANSI/control
// sequences before it reaches the TUI. Belt-and-suspenders on top of the cache
// load-time guard in internal/update.
var validSemverFooter = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+.][\w.-]+)?$`)

// resultHints returns the footer actions for the result screen.
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
func (m ResultModel) View() string {
	footer := RenderFooter(resultHints(), m.w, m.th)
	updateLine := m.renderUpdateHint()

	panel := m.renderPanel()

	// Vertical padding: pin footer to bottom.
	panelLines := strings.Count(panel, "\n") + 1
	updateLineCount := 0
	if updateLine != "" {
		updateLineCount = 1
	}
	used := panelLines + 1 + updateLineCount + 1 // panel + blank + [update hint +] footer
	spacer := m.h - used
	if spacer < 1 {
		spacer = 1
	}

	var b strings.Builder
	b.WriteString(panel)
	b.WriteString(strings.Repeat("\n", spacer))
	if updateLine != "" {
		b.WriteString(updateLine)
		b.WriteString("\n")
	}
	b.WriteString(footer)

	frame := b.String()
	if m.w > 0 && m.h > 0 {
		frame = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, frame)
	}
	// New-best celebration: one-shot sparkle burst overlaid onto blank margin
	// rows of the placed frame. Triggers only on a new best, never on ordinary
	// results, and only while the burst window is open.
	if m.isBest {
		frame = applyCelebration(frame, m.revealStartMs, m.nowMs, m.th)
	}
	return frame
}

// renderUpdateHint returns the update-available footer line, or "" if no hint
// or if the version string fails semver validation (injection guard).
func (m ResultModel) renderUpdateHint() string {
	if m.updateHint == nil {
		return ""
	}
	latest := m.updateHint.Latest
	if !validSemverFooter.MatchString(latest) {
		return ""
	}
	hint := fmt.Sprintf("↑ %s available — run \"typeburn update\"", latest)
	return m.th.Style(theme.RoleTextMuted).Render(hint)
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
	inner.WriteString(m.renderKeyHeatmap(innerW))
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

// renderSparkline renders the "wpm over time" sparkline section.
func (m ResultModel) renderSparkline(innerW int) string {
	header := m.th.Style(theme.RoleTextMuted).Render("wpm over time")

	vals := make([]float64, len(m.res.PerSecond))
	for i, ps := range m.res.PerSecond {
		vals[i] = ps.RawWPM
	}

	visible := sparkVisibleBars(len(vals), m.revealStartMs, m.nowMs)
	graph := sparklineVisible(vals, innerW, 3, visible, m.th)
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
	}
	return strings.Join(parts, "   ")
}

// renderMeta renders the duration · mode length · english line.
func (m ResultModel) renderMeta() string {
	style := m.th.Style(theme.RoleTextFaint)
	dur := fmt.Sprintf("%.0fs", float64(m.res.DurationMs)/1000.0)
	return style.Render(dur + " · " + displayModeLabel(string(m.mode), m.length) + " · english")
}
