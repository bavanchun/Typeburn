package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// historyColWidths defines the fixed character widths for each table column.
// DATE(16) + MODE(12) + WPM(6) + ACC(7) + CONS(7) + STAR(2) = 50 inner chars + separating spaces.
const (
	colDateW = 16
	colModeW = 12
	colWPMW  = 6
	colAccW  = 7
	colConsW = 7
)

// effWPM returns the effective WPM for new-best comparison as a float64.
// Records written before the NetWPM field was added unmarshal as 0.0; the
// fallback to float64(WPM) keeps the same integer scale so an old record's
// best is not unfairly overridden by any new run with a lower rounded WPM.
func effWPM(r storage.Record) float64 {
	if r.NetWPM == 0 {
		return float64(r.WPM)
	}
	return r.NetWPM
}

// bestWPMPerBucket returns the highest effective WPM for each mode+length bucket
// across all provided records. The key format matches storage.IsNewBest scoping.
// Effective WPM uses the persisted NetWPM float when present, falling back to
// float64(WPM) for legacy records so the scale comparison is consistent.
func bestWPMPerBucket(rows []storage.Record) map[string]float64 {
	bests := make(map[string]float64)
	for _, r := range rows {
		key := histBucketKey(r.Mode, r.Length)
		eff := effWPM(r)
		if prev, ok := bests[key]; !ok || eff > prev {
			bests[key] = eff
		}
	}
	return bests
}

// histBucketKey mirrors storage.modeKey logic for UI use.
func histBucketKey(mode string, length int) string {
	switch mode {
	case "time", "words":
		return fmt.Sprintf("%s/%d", mode, length)
	default:
		return mode
	}
}

// renderHistoryHeader renders the UPPERCASE header row with border rules above
// and below, styled in RoleTextMuted per mockups §5.
func renderHistoryHeader(th theme.Theme) string {
	sep := th.Style(theme.RoleBorder).Render(strings.Repeat("─", 62))
	mutedStyle := th.Style(theme.RoleTextMuted)

	header := fmt.Sprintf("  %-*s %-*s %-*s %-*s %-*s",
		colDateW, "DATE",
		colModeW, "MODE",
		colWPMW, "WPM",
		colAccW, "ACC",
		colConsW, "CONS",
	)
	return sep + "\n" + mutedStyle.Render(header) + "\n" + sep
}

// renderHistoryRow renders a single history table row. Selected rows get the
// ▎ accent bar and RoleSurfaceAlt background. Per-mode best rows get a ★ badge.
func renderHistoryRow(r storage.Record, selected bool, isBestRow bool, th theme.Theme) string {
	// Format each column value.
	date := r.Time.Format("2006-01-02 15:04")
	modeLabel := modeLabel(r.Mode, r.Length)
	wpm := fmt.Sprintf("%d", r.WPM)
	acc := fmt.Sprintf("%.0f%%", r.Accuracy)
	cons := fmt.Sprintf("%.0f%%", r.Consistency)

	// Determine accuracy color: success if ≥95, else muted.
	accRole := theme.RoleTextMuted
	if r.Accuracy >= 95 {
		accRole = theme.RoleSuccess
	}

	star := "  "
	if isBestRow {
		star = th.Style(theme.RoleSuccess).Render("★") + " "
	}

	if selected {
		bar := th.Style(theme.RoleAccent).Bold(true).Render("▎")
		bgStyle := lipgloss.NewStyle().Background(th.Color(theme.RoleSurfaceAlt))
		wpmStyled := bgStyle.Render(th.Style(theme.RoleTextPrimary).Bold(true).Render(fmt.Sprintf("%-*s", colWPMW, wpm)))
		accStyled := bgStyle.Render(th.Style(accRole).Render(fmt.Sprintf("%-*s", colAccW, acc)))
		consStyled := bgStyle.Render(th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colConsW, cons)))
		dateStyled := bgStyle.Render(th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colDateW, date)))
		modeStyled := bgStyle.Render(th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colModeW, modeLabel)))
		return bar + " " + dateStyled + " " + modeStyled + " " + wpmStyled + " " + accStyled + " " + consStyled + star
	}

	// Unselected row styling.
	wpmStyled := th.Style(theme.RoleTextPrimary).Bold(true).Render(fmt.Sprintf("%-*s", colWPMW, wpm))
	accStyled := th.Style(accRole).Render(fmt.Sprintf("%-*s", colAccW, acc))
	consStyled := th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colConsW, cons))
	dateStyled := th.Style(theme.RoleTextMuted).Render(fmt.Sprintf("%-*s", colDateW, date))
	modeStyled := th.Style(theme.RoleTextMuted).Render(fmt.Sprintf("%-*s", colModeW, modeLabel))
	return "   " + dateStyled + " " + modeStyled + " " + wpmStyled + " " + accStyled + " " + consStyled + star
}

// modeLabel formats a mode+length label for display (e.g. "time 30", "words 50", "quote").
func modeLabel(mode string, length int) string {
	switch mode {
	case "time":
		return fmt.Sprintf("time %d", length)
	case "words":
		return fmt.Sprintf("words %d", length)
	default:
		return "quote"
	}
}

// renderHistoryMeta renders the "showing X–Y of N" meta line in RoleTextFaint.
func renderHistoryMeta(top, sel, total int, th theme.Theme) string {
	from := top + 1
	to := sel + 1
	if to < from {
		to = from
	}
	return th.Style(theme.RoleTextFaint).Render(
		fmt.Sprintf("showing %d–%d of %d", from, to, total),
	)
}

// historyFooterHints returns the footer hint set for the History screen per §8.6.
func historyFooterHints() []Hint {
	return []Hint{
		{Key: "↑↓", Action: "scroll"},
		{Key: "g/G", Action: "top/bottom"},
		{Key: "esc", Action: "back"},
		{Key: "ctrl+c", Action: "quit"},
	}
}
