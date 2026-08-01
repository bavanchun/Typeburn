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
	// Geometry comes from layoutFor so the panel, hero, graph, and stats grid
	// cannot disagree about how much room they have.
	lay := layoutFor(m.w)
	panelW, innerW := lay.PanelW, lay.InnerW

	var inner strings.Builder
	inner.WriteString(m.renderHero(innerW))
	inner.WriteString("\n\n")
	inner.WriteString(m.renderGraph(innerW))
	inner.WriteString("\n\n")
	inner.WriteString(m.renderStatsGrid(innerW))
	inner.WriteString("\n\n")
	inner.WriteString(m.renderKeyHeatmap(innerW))

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

// renderGraph renders the "wpm over time" dual-axis line graph section. The
// reveal draw-in reuses the sparkVisibleBars progress source (one source of
// truth, so the graph settles exactly with the rest of the reveal).
func (m ResultModel) renderGraph(innerW int) string {
	header := m.th.Style(theme.RoleTextMuted).Render("wpm over time")
	visible := sparkVisibleBars(len(m.res.PerSecond), m.revealStartMs, m.nowMs)
	graph := RenderResultGraph(m.res.PerSecond, innerW, 5, visible, m.th)
	if graph == "" {
		graph = m.th.Style(theme.RoleTextFaint).Render("(no data)")
	}
	return header + "\n" + graph
}

// renderStatsGrid renders the 2-column stats grid: test type / raw /
// characters on the left, consistency / time on the right. Below TierMid the
// two columns stack vertically so 60-col terminals never overflow.
func (m ResultModel) renderStatsGrid(innerW int) string {
	label := m.th.Style(theme.RoleTextMuted)
	value := m.th.Style(theme.RoleTextPrimary).Bold(true)

	incVal := value.Render(fmt.Sprintf("%d", m.res.IncorrectChars))
	if m.res.IncorrectChars > 0 {
		incVal = m.th.Style(theme.RoleError).Bold(true).Render(fmt.Sprintf("%d", m.res.IncorrectChars))
	}
	chars := value.Render(fmt.Sprintf("%d", m.res.CorrectChars)) +
		label.Render("/") + incVal +
		label.Render("/") + value.Render(fmt.Sprintf("%d", m.res.ExtraChars))

	left := []string{
		label.Render("test type") + "  " + value.Render(displayModeLabel(string(m.mode), m.length)+" · english"),
		label.Render("raw") + "        " + value.Render(fmt.Sprintf("%.0f wpm", m.res.RawWPM)),
		label.Render("characters") + " " + chars,
	}
	right := []string{
		label.Render("consistency") + " " + value.Render(fmt.Sprintf("%.0f%%", m.res.Consistency)),
		label.Render("time") + "        " + value.Render(fmt.Sprintf("%.0fs", float64(m.res.DurationMs)/1000.0)),
	}

	leftCol := strings.Join(left, "\n")
	rightCol := strings.Join(right, "\n")
	// Two columns need colW ≥ 30 so the longest left line ("test type
	// words 100 · english", 30 chars) never wraps inside its column block.
	if innerW < 60 {
		return leftCol + "\n" + rightCol
	}
	colW := innerW / 2
	leftBlock := lipgloss.NewStyle().Width(colW).Render(leftCol)
	return lipgloss.JoinHorizontal(lipgloss.Top, leftBlock, rightCol)
}
