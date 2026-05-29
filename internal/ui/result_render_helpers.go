package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// heatmapMaxEntries caps how many missed keys the Result line shows.
const heatmapMaxEntries = 8

// renderKeyHeatmap renders the "most missed" key line from m.res.KeyMisses,
// capped to heatmapMaxEntries and to innerW visible columns (never overflowing
// the panel). A clean run (no misses) renders a faint "no missed keys".
//
// Theme roles only — the plain text is identical under NO_COLOR/mono (attributes
// differ, layout does not).
func (m ResultModel) renderKeyHeatmap(innerW int) string {
	if len(m.res.KeyMisses) == 0 {
		return m.th.Style(theme.RoleTextFaint).Render("no missed keys")
	}

	labelStyle := m.th.Style(theme.RoleTextMuted)
	keyStyle := m.th.Style(theme.RoleTextPrimary)
	countStyle := m.th.Style(theme.RoleError)

	const prefix = "most missed:  "
	width := lipgloss.Width(prefix)
	if width > innerW {
		// Pathologically narrow panel — show the faint fallback rather than overflow.
		return m.th.Style(theme.RoleTextFaint).Render("no missed keys")
	}

	var b strings.Builder
	b.WriteString(labelStyle.Render(prefix))

	added := 0
	for _, km := range m.res.KeyMisses {
		if added >= heatmapMaxEntries {
			break
		}
		sep := ""
		if added > 0 {
			sep = "   "
		}
		entry := fmt.Sprintf("%s ×%d", km.Label, km.Misses)
		if width+lipgloss.Width(sep+entry) > innerW {
			break
		}
		width += lipgloss.Width(sep + entry)

		b.WriteString(sep)
		b.WriteString(keyStyle.Render(km.Label))
		b.WriteString(" ")
		b.WriteString(countStyle.Render(fmt.Sprintf("×%d", km.Misses)))
		added++
	}
	return b.String()
}

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

// stripANSI removes ANSI CSI escape sequences (ESC [ ... finalByte 0x40-0x7E)
// from s so the visual character width can be measured accurately for border
// title injection. Only CSI (ESC[) is handled; OSC/SS3 are not emitted by
// lipgloss panel borders and are out of scope. A non-[ introducer (e.g. ESC M)
// is treated as a 2-byte sequence and drops exactly one byte.
func stripANSI(s string) string {
	var out strings.Builder
	const (
		stNorm = iota
		stEsc  // saw ESC, expecting introducer
		stCSI  // inside CSI params/intermediates, waiting for final byte
	)
	state := stNorm
	for _, r := range s {
		switch state {
		case stNorm:
			if r == '\x1b' {
				state = stEsc
			} else {
				out.WriteRune(r)
			}
		case stEsc:
			if r == '[' {
				state = stCSI
			} else {
				state = stNorm // non-[ introducer: drop it, done with this escape
			}
		case stCSI:
			if r >= '@' && r <= '~' { // CSI final byte 0x40..0x7E
				state = stNorm
			}
			// else: param/intermediate byte — keep dropping
		}
	}
	return out.String()
}
