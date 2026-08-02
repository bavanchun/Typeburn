package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
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

// History table geometry.
//
// historyIndentW is the gutter the selection bar occupies ("▎ "). The header
// and both row forms use the same gutter, so a selected row's columns line up
// with the header's instead of shifting a cell left.
//
// historyRowW is the width a rendered row occupies: the gutter, the five padded
// columns with a space between each, and the two-cell star slot.
// TestHistoryRow_MatchesTheDeclaredWidth holds it to the renderer.
const (
	historyRuleMaxW = 62
	historyIndentW  = 2
	historyRowW     = historyIndentW + colDateW + 1 + colModeW + 1 + colWPMW + 1 + colAccW + 1 + colConsW + 2
)

// historyRuleW sizes the rules above and below the table to the terminal.
//
// A constant 62 overflowed the 60-column minimum the product advertises, and
// because History is centred with lipgloss.Place — which sizes the block from
// its widest line — an over-wide rule does not just spill, it decentres every
// other line on the screen. Two cells are held back so the rule never runs into
// the terminal edge, and the rule never shrinks below the rows it brackets.
func historyRuleW(termW int) int {
	if termW <= 0 {
		return historyRuleMaxW
	}
	w := termW - 2
	if w > historyRuleMaxW {
		w = historyRuleMaxW
	}
	if w < historyRowW {
		w = historyRowW
	}
	return w
}

// renderHistoryHeader renders the UPPERCASE header row with border rules above
// and below, styled in RoleTextMuted per mockups §5.
func renderHistoryHeader(th theme.Theme, ruleW int) string {
	sep := th.Style(theme.RoleBorder).Render(strings.Repeat("─", ruleW))
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
// ▎ accent bar and RoleSurfaceAlt background; unselected rows leave the same
// gutter blank so the columns do not move when the cursor lands on them.
// Per-mode best rows get a ★ badge.
func renderHistoryRow(r storage.Record, selected bool, isBestRow bool, th theme.Theme) string {
	// Format each column value.
	date := r.Time.Format("2006-01-02 15:04")
	label := displayModeLabel(r.Mode, r.Length)
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
		modeStyled := bgStyle.Render(th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colModeW, label)))
		return bar + " " + dateStyled + " " + modeStyled + " " + wpmStyled + " " + accStyled + " " + consStyled + star
	}

	// Unselected row styling.
	wpmStyled := th.Style(theme.RoleTextPrimary).Bold(true).Render(fmt.Sprintf("%-*s", colWPMW, wpm))
	accStyled := th.Style(accRole).Render(fmt.Sprintf("%-*s", colAccW, acc))
	consStyled := th.Style(theme.RoleTextPrimary).Render(fmt.Sprintf("%-*s", colConsW, cons))
	dateStyled := th.Style(theme.RoleTextMuted).Render(fmt.Sprintf("%-*s", colDateW, date))
	modeStyled := th.Style(theme.RoleTextMuted).Render(fmt.Sprintf("%-*s", colModeW, label))
	return "  " + dateStyled + " " + modeStyled + " " + wpmStyled + " " + accStyled + " " + consStyled + star
}

// renderHistoryMeta renders the "showing X–Y of N" meta line in RoleTextFaint.
//
// end is the exclusive end of the visible window, not the cursor. Reporting the
// cursor made a screen showing fourteen rows say "showing 1–1 of 120", which
// describes neither the selection nor the page.
func renderHistoryMeta(top, end, total int, th theme.Theme) string {
	from := top + 1
	to := end
	if to > total {
		to = total
	}
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
