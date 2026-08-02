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
//
// The spacer is never zero. That blank row is what the new-best celebration
// bursts into: it only ever rewrites all-blank rows, so a layout with no slack
// would silently switch the feature off rather than fail.
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

// renderPanel builds the rounded-border result panel. Every row is produced at
// exactly lay.InnerW cells and the row count is fixed by lay.ContentRows, so the
// panel's height is a property of the terminal size rather than of whichever
// values this particular run produced.
func (m ResultModel) renderPanel() string {
	lay := layoutFor(m.w, m.h)

	lines := make([]string, 0, lay.ContentRows())
	lines = append(lines, m.heroBand(lay)...)
	lines = append(lines, m.chartLines(lay)...)
	lines = append(lines, m.metaLines(lay)...)

	borderColor := m.th.Color(theme.RoleBorder)
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(lay.VPad, 2).
		Width(lay.PanelW)

	panel := borderStyle.Render(strings.Join(lines, "\n"))
	titleStyled := m.th.Style(theme.RoleTextMuted).Render(" result ")
	return injectBorderTitle(panel, titleStyled)
}

// chartLines renders the chart header plus the plot, always as 1+ChartH+2 rows.
// A run with no per-second samples still occupies the same rows, so an empty
// data set cannot change the panel's height.
func (m ResultModel) chartLines(lay resultLayout) []string {
	header := metaRow(
		m.th.Style(theme.RoleTextMuted).Render("wpm over time"),
		m.th.Style(theme.RoleTextMuted).Render(m.modeMeta()),
		lay.InnerW)

	out := make([]string, 0, lay.ChartH+3)
	out = append(out, header)

	visible := sparkVisibleBars(len(m.res.PerSecond), m.revealStartMs, m.nowMs)
	graph := RenderResultGraph(m.res.PerSecond, lay.InnerW, lay.ChartH, visible, m.th)
	if graph == "" {
		blank := make([]string, lay.ChartH+2)
		blank[lay.ChartH/2] = m.th.Style(theme.RoleTextFaint).Render("(no data)")
		for _, ln := range blank {
			out = append(out, padCell(ln, lay.InnerW))
		}
		return out
	}
	for _, ln := range strings.Split(graph, "\n") {
		out = append(out, padCell(ln, lay.InnerW))
	}
	return out
}

// modeMeta is the run's identity line: what test it was, in what language, for
// how long.
func (m ResultModel) modeMeta() string {
	return fmt.Sprintf("%s · english · %.0fs",
		displayModeLabel(string(m.mode), m.length), float64(m.res.DurationMs)/1000.0)
}

// metaLines renders the closing rows. The last row carries ink at both edges:
// the character breakdown on the left and this run's standing on the right. A
// short terminal keeps that row and drops the missed-key line, because a
// breakdown of what you typed outranks a list of what you fumbled.
func (m ResultModel) metaLines(lay resultLayout) []string {
	p := cardProgress(2, m.revealStartMs, m.nowMs)
	out := make([]string, 0, 2)
	if !lay.Compact {
		out = append(out, revealLine(padCell(m.renderKeyHeatmap(lay.InnerW), lay.InnerW), p, m.th))
	}
	out = append(out, revealLine(metaRow(m.charBreakdown(), m.standing(), lay.InnerW), p, m.th))
	return out
}

// charBreakdown labels every number in the character triple. The middle figure
// used to be distinguished by colour alone, which is invisible under NO_COLOR
// and nearly invisible in the attribute-only mono theme.
func (m ResultModel) charBreakdown() string {
	label := m.th.Style(theme.RoleTextMuted)
	value := m.th.Style(theme.RoleTextPrimary).Bold(true)
	wrong := value
	if m.res.IncorrectChars > 0 {
		wrong = m.th.Style(theme.RoleError).Bold(true)
	}
	return value.Render(fmt.Sprintf("%d", m.res.CorrectChars)) + label.Render(" correct · ") +
		wrong.Render(fmt.Sprintf("%d", m.res.IncorrectChars)) + label.Render(" wrong · ") +
		value.Render(fmt.Sprintf("%d", m.res.ExtraChars)) + label.Render(" extra")
}

// standing is where this run sits in its own mode+length bucket. Scoping the
// rank to the bucket is deliberate: "#2 of 6" among comparable runs answers a
// question "#2 of 300" across every mode cannot.
func (m ResultModel) standing() string {
	switch {
	case !m.ctx.HasHistory:
		return m.th.Style(theme.RoleTextFaint).Render("first run")
	case m.ctx.Total <= 0:
		return m.th.Style(theme.RoleTextFaint).Render("not ranked")
	}
	return m.th.Style(theme.RoleTextMuted).Render(fmt.Sprintf("#%d of %d runs", m.ctx.Rank, m.ctx.Total))
}

// metaRow puts left at the row's left edge and right at its right edge, or
// drops right when the pair cannot share the row without colliding.
func metaRow(left, right string, innerW int) string {
	lw, rw := lipgloss.Width(left), lipgloss.Width(right)
	if right != "" && lw+2+rw <= innerW {
		return left + strings.Repeat(" ", innerW-lw-rw) + right
	}
	return padCell(left, innerW)
}
