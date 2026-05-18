package ui

import (
	"strings"

	"github.com/bavanchun/Typeburn/internal/theme"
)

// logoLines is the block-art ("ANSI Shadow" style) wordmark for "TYPEBURN",
// rendered in the accent color. Each row is a fixed 69 cells wide so the
// glyphs always align regardless of terminal color support.
var logoLines = []string{
	`████████╗██╗   ██╗██████╗ ███████╗██████╗ ██╗   ██╗██████╗ ███╗   ██╗`,
	`╚══██╔══╝╚██╗ ██╔╝██╔══██╗██╔════╝██╔══██╗██║   ██║██╔══██╗████╗  ██║`,
	`   ██║    ╚████╔╝ ██████╔╝█████╗  ██████╔╝██║   ██║██████╔╝██╔██╗ ██║`,
	`   ██║     ╚██╔╝  ██╔═══╝ ██╔══╝  ██╔══██╗██║   ██║██╔══██╗██║╚██╗██║`,
	`   ██║      ██║   ██║     ███████╗██████╔╝╚██████╔╝██║  ██║██║ ╚████║`,
	`   ╚═╝      ╚═╝   ╚═╝     ╚══════╝╚═════╝  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝`,
}

// logoWidth is the fixed cell width of every logoLines row. Below this many
// usable columns the wordmark is replaced by a plain text fallback.
const logoWidth = 69

// RenderLogo returns the ASCII block-art wordmark styled per design §3.
//
// Wide form (width >= logoWidth+3): full "TYPEBURN" block art in accent.
// Narrow form: plain Bold accent "typeburn" on a single line, per the
// Responsive/Degraded rule (threshold is logoWidth+3 since this 8-letter
// wordmark is wider than the design's original 64-col assumption).
func RenderLogo(width int, th theme.Theme) string {
	accentStyle := th.Style(theme.RoleAccent).Bold(true)

	if width < logoWidth+3 {
		return accentStyle.Render("typeburn")
	}

	lines := make([]string, len(logoLines))
	for i, line := range logoLines {
		lines[i] = accentStyle.Render(line)
	}
	return strings.Join(lines, "\n")
}
