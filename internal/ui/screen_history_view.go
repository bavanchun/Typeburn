package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
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
		body.WriteString(trendLine(sparkVals, n, m.w, m.th))
		body.WriteString("\n\n")
		body.WriteString(renderHistoryHeader(m.th, historyRuleW(m.w)))
		body.WriteString("\n")

		// Windowed rows.
		vis := m.visibleCount()
		bests := storage.BestWPMPerBucket(m.rows)

		end := m.top + vis
		if end > n {
			end = n
		}
		for i := m.top; i < end; i++ {
			r := rows[i]
			key := storage.BestBucketKey(r.Mode, r.Length)
			// Compare effective WPM against the persisted bucket best.
			// Float equality is safe here: we compare the same stored value
			// (EffectiveWPM(r)) against what BestWPMPerBucket already derived from
			// the same records — no recomputation drift possible.
			isBestRow := storage.EligibleForBest(r) && storage.EffectiveWPM(r) == bests[key]
			body.WriteString(renderHistoryRow(r, i == m.sel, isBestRow, m.th))
			body.WriteString("\n")
		}

		// Bottom border rule.
		sep := m.th.Style(theme.RoleBorder).Render(strings.Repeat("─", historyRuleW(m.w)))
		body.WriteString(sep)
		body.WriteString("\n")
		body.WriteString(renderHistoryMeta(m.top, end, n, m.th))
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
