package ui

import (
	"fmt"
	"strings"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// progressBarWidth is the number of cells in the Quote mode progress bar.
const progressBarWidth = 5

// ModeHeader renders the ultra-minimal header for the typing screen.
//
// Layout per design §5.4 / mockups §2:
//   - Time mode:  "WPM   elapsed / total"     (e.g. "87 wpm   0:23 / 0:30")
//   - Words mode: "WPM   done / total"         (e.g. "87 wpm   12 / 25")
//   - Quote mode: "WPM   pct% ▰▰▰▱▱"          (e.g. "87 wpm   42% ▰▰▰▱▱")
//
// WPM number is accent+bold; "wpm" and remainder are text-muted.
// No border; left-aligned.
func ModeHeader(
	mode config.Mode,
	wpm float64,
	done, total int,
	elapsedSec float64,
	limitSec int,
	th theme.Theme,
) string {
	accentBold := th.Style(theme.RoleAccent).Bold(true)
	muted := th.Style(theme.RoleTextMuted)

	wpmStr := accentBold.Render(fmt.Sprintf("%.0f", wpm))
	label := muted.Render(" wpm   ")

	var right string
	switch mode {
	case config.ModeTime:
		elapsed := formatSeconds(int(elapsedSec))
		limit := formatSeconds(limitSec)
		right = muted.Render(elapsed + " / " + limit)

	case config.ModeWords:
		right = muted.Render(fmt.Sprintf("%d / %d", done, total))

	case config.ModeQuote:
		var pct float64
		if total > 0 {
			pct = float64(done) / float64(total) * 100
		}
		bar := renderProgressBar(pct, progressBarWidth, th)
		right = muted.Render(fmt.Sprintf("%.0f%%", pct)) + " " + bar
	}

	return wpmStr + label + right
}

// formatSeconds converts a total-seconds value to "m:ss" display format.
func formatSeconds(s int) string {
	m := s / 60
	sec := s % 60
	return fmt.Sprintf("%d:%02d", m, sec)
}

// renderProgressBar returns a filled/empty block progress bar.
// Filled cells use ▰ in accent-dim; empty cells use ▱ in text-faint.
func renderProgressBar(pct float64, width int, th theme.Theme) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	accentDim := th.Style(theme.RoleAccentDim)
	faint := th.Style(theme.RoleTextFaint)

	var sb strings.Builder
	for i := 0; i < width; i++ {
		if i < filled {
			sb.WriteString(accentDim.Render("▰"))
		} else {
			sb.WriteString(faint.Render("▱"))
		}
	}
	return sb.String()
}
