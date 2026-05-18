package ui

import (
	"strings"

	"monkeytype-tui/internal/theme"
)

// logoLines is the block-art representation of "MONKEY" in accent color.
// The trailing "type" word is rendered separately in text-muted.
// Width: each line is ~56 chars (7 chars × 8 wide) for the MONKEY part.
var logoLines = []string{
	`███╗   ███╗ ██████╗ ███╗   ██╗██╗  ██╗███████╗██╗   ██╗`,
	`████╗ ████║██╔═══██╗████╗  ██║██║ ██╔╝██╔════╝╚██╗ ██╔╝`,
	`██╔████╔██║██║   ██║██╔██╗ ██║█████╔╝ █████╗   ╚████╔╝ `,
	`██║╚██╔╝██║██║   ██║██║╚██╗██║██╔═██╗ ██╔══╝    ╚██╔╝  `,
	`██║ ╚═╝ ██║╚██████╔╝██║ ╚████║██║  ██╗███████╗   ██║   `,
	`╚═╝     ╚═╝ ╚═════╝ ╚═╝  ╚═══╝╚═╝  ╚═╝╚══════╝   ╚═╝   `,
}

// typeLabel appended to the last logo line, in text-muted.
const typeLabel = "  t y p e"

// RenderLogo returns the ASCII block-art logo styled per design §3 + mockups §1.
//
// Wide form (width >= 64): full block art in accent with "type" trailing in
// text-muted on the last line. Narrow form (width < 64): plain Bold accent
// "monkeytype" (single line) per the Responsive/Degraded rule.
func RenderLogo(width int, th theme.Theme) string {
	accentStyle := th.Style(theme.RoleAccent).Bold(true)
	mutedStyle := th.Style(theme.RoleTextMuted)

	if width < 64 {
		// Narrow fallback: plain bold accent text.
		return accentStyle.Render("monkeytype")
	}

	// Full block-art logo.
	lines := make([]string, len(logoLines))
	for i, line := range logoLines {
		if i == len(logoLines)-1 {
			// Last line: MONKEY art + "type" in muted.
			lines[i] = accentStyle.Render(line) + mutedStyle.Render(typeLabel)
		} else {
			lines[i] = accentStyle.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}
