package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/theme"
)

// View renders the History screen. It places title, trend sparkline, table,
// meta line, and footer per mockups §5.
// Degraded mode (w<60 or h<20) is handled by the root View; this is only called
// when the terminal meets the safe minimum.
func (m HistoryModel) View() string {
	title := m.th.Style(theme.RoleAccent).Bold(true).Render("H I S T O R Y")
	footer := RenderFooter(historyFooterHints(), m.w, m.th)

	var body strings.Builder
	body.WriteString(title)
	body.WriteString("\n\n")

	rows := m.newestFirst()
	n := len(rows)

	if n == 0 {
		// Empty state: friendly centered message.
		msg := m.th.Style(theme.RoleTextMuted).Render("no tests yet — press 1 to start")
		body.WriteString(msg)
	} else {
		// Trend sparkline from all records (newest last = left-to-right chronological).
		// We use the rows in oldest-first order (m.rows) for the sparkline so the
		// rightmost bar is the most recent result.
		sparkVals := make([]float64, len(m.rows))
		for i, r := range m.rows {
			sparkVals[i] = float64(r.WPM)
		}
		sparkLabel := m.th.Style(theme.RoleTextMuted).Render(
			"trend  " + sparklineInline(sparkVals, m.th) +
				"  last " + histItoa(n) + " tests",
		)
		body.WriteString(sparkLabel)
		body.WriteString("\n\n")
		body.WriteString(renderHistoryHeader(m.th))
		body.WriteString("\n")

		// Windowed rows.
		vis := m.visibleCount()
		bests := bestWPMPerBucket(m.rows)

		end := m.top + vis
		if end > n {
			end = n
		}
		for i := m.top; i < end; i++ {
			r := rows[i]
			key := histBucketKey(r.Mode, r.Length)
			isBestRow := bests[key] == r.WPM
			body.WriteString(renderHistoryRow(r, i == m.sel, isBestRow, m.th))
			body.WriteString("\n")
		}

		// Bottom border rule.
		sep := m.th.Style(theme.RoleBorder).Render(strings.Repeat("─", 62))
		body.WriteString(sep)
		body.WriteString("\n")
		body.WriteString(renderHistoryMeta(m.top, m.sel, n, m.th))
	}

	content := body.String()

	// Pin footer to bottom.
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

// sparklineInline renders a compact single-row sparkline string for the trend
// label. It uses only the bar characters (no axis) for inline display.
func sparklineInline(vals []float64, th theme.Theme) string {
	if len(vals) == 0 {
		return ""
	}
	minV, maxV := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		minV = 0
		if maxV == 0 {
			maxV = 1
		}
	}
	bars := make([]rune, len(vals))
	for i, v := range vals {
		ratio := (v - minV) / (maxV - minV)
		idx := int(ratio*float64(len(sparkBars)-1) + 0.5)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBars) {
			idx = len(sparkBars) - 1
		}
		bars[i] = sparkBars[idx]
	}
	return th.Style(theme.RoleAccent).Render(string(bars))
}

// itoa is re-declared in the ui package for use in screen_history_view.go.
// It converts a non-negative int to a decimal string without importing fmt.
func histItoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
