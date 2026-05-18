package ui

import (
	"fmt"
	"strings"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

// accColorRole picks the appropriate theme role for accuracy display.
// ≥97 → RoleSuccess, <90 → RoleWarning, else → RoleTextPrimary.
func accColorRole(acc float64) theme.Role {
	switch {
	case acc >= 97:
		return theme.RoleSuccess
	case acc < 90:
		return theme.RoleWarning
	default:
		return theme.RoleTextPrimary
	}
}

// modeMetaLabel formats the mode+length string for the result meta line.
// Quote mode has no numeric length so it returns "quote".
func modeMetaLabel(mode config.Mode, length int) string {
	switch mode {
	case config.ModeTime:
		return fmt.Sprintf("time %d", length)
	case config.ModeWords:
		return fmt.Sprintf("words %d", length)
	default:
		return "quote"
	}
}

// injectBorderTitle replaces the visual centre of the top border line with a
// styled title string. It operates on the raw string representation of a
// lipgloss-rendered panel and modifies only the first line (the top border).
//
// ANSI codes are stripped for width measurement then the plain top border is
// rebuilt with the title runes copied in at the centre position. Because we
// rebuild from the stripped text, the result loses border ANSI styling on the
// top line — acceptable since the top-border colour is subtle (RoleBorder) and
// the title label is independently styled before passing in.
func injectBorderTitle(panel, title string) string {
	lines := strings.SplitN(panel, "\n", 2)
	if len(lines) == 0 {
		return panel
	}

	raw := []rune(stripANSI(lines[0]))
	titleRunes := []rune(stripANSI(title))

	midStart := (len(raw) - len(titleRunes)) / 2
	if midStart < 1 {
		midStart = 1
	}
	if midStart+len(titleRunes) > len(raw) {
		// Title too wide — skip injection and return panel unchanged.
		return panel
	}

	newTop := make([]rune, len(raw))
	copy(newTop, raw)
	copy(newTop[midStart:], titleRunes)

	if len(lines) > 1 {
		return string(newTop) + "\n" + lines[1]
	}
	return string(newTop)
}

// stripANSI removes ANSI SGR escape sequences (ESC [ ... m) from s so the
// visual character width can be measured accurately for border title injection.
func stripANSI(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEsc = true
		case inEsc:
			if r == 'm' {
				inEsc = false
			}
		default:
			out.WriteRune(r)
		}
	}
	return out.String()
}
