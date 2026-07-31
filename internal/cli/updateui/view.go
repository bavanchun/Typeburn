package updateui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// boxWidth is the frame's total cell width. Fixed rather than terminal-derived:
// a block that reflowed mid-download would be worse than one that never fits,
// so a terminal narrower than this falls back to plain output instead (the
// caller gates on it).
const boxWidth = 50

// BoxWidth exposes the frame width so the caller can decide whether the
// terminal is wide enough to render it at all.
const BoxWidth = boxWidth

// stages lists the run in order. Row count is constant for the whole run, so
// the block never grows or shrinks while it is on screen.
var stages = []update.Stage{
	update.StageChecksums,
	update.StageDownloading,
	update.StageVerifying,
	update.StageInstalling,
}

// Frame renders the current frame as a plain string. Kept separate from View so
// tests can assert on layout without running a program.
func (m Model) Frame() string {
	var body strings.Builder

	body.WriteString(m.headerLine() + "\n\n")
	for _, s := range stages {
		body.WriteString(m.stageRow(s) + "\n")
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(boxWidth)
	if !m.noColor {
		box = box.BorderForeground(m.theme.Color(theme.RoleBorder))
	}

	rendered := box.Render(strings.TrimRight(body.String(), "\n"))
	title := m.style(theme.RoleAccent).Render(" typeburn update ")
	return "\n" + injectTitle(rendered, title) + "\n"
}

// headerLine is the version transition plus the archive size, which lives here
// rather than on the download row: inline it overflowed boxWidth and wrapped,
// breaking the frame.
func (m Model) headerLine() string {
	head := m.style(theme.RoleTextFaint).Render(m.from) +
		m.style(theme.RoleTextFaint).Render("   →   ") +
		m.style(theme.RoleAccent).Render(m.to)
	if m.cur.Total > 0 {
		head += "     " + m.style(theme.RoleTextFaint).Render(humanBytes(m.cur.Total))
	}
	return head
}

// stageRow renders one checklist line. The download row additionally carries
// the progress bar while it is the active stage.
func (m Model) stageRow(s update.Stage) string {
	row := m.glyph(s) + "  " + m.label(s)
	if s == update.StageDownloading && m.cur.Stage == update.StageDownloading {
		row += "  " + m.barView()
	}
	return row
}

// glyph is the leading status marker: a settled check, the live spinner, or a
// faint pending dot. Stages are declared in run order, so a stage numerically
// below the current one is finished.
func (m Model) glyph(s update.Stage) string {
	switch {
	case m.done || m.cur.Stage > s:
		return m.style(theme.RoleSuccess).Render("✓")
	case m.cur.Stage == s:
		return m.style(theme.RoleAccent).Render(m.spin.View())
	default:
		return m.style(theme.RoleTextFaint).Render("·")
	}
}

func (m Model) label(s update.Stage) string {
	switch {
	case m.done || m.cur.Stage > s:
		return m.style(theme.RoleTextMuted).Render(s.String())
	case m.cur.Stage == s:
		return m.style(theme.RoleTextPrimary).Bold(true).Render(s.String())
	default:
		return m.style(theme.RoleTextFaint).Render(s.String())
	}
}

// style resolves a Role, always through the theme so no color literal appears
// in this package.
func (m Model) style(r theme.Role) lipgloss.Style { return m.theme.Style(r) }

// injectTitle overwrites part of the box's top border with a label. Lip Gloss
// v2 has no bordered-title API, so the line is rebuilt from its own measured
// cell count — slicing at fixed offsets goes off by one as soon as the border
// glyphs are multi-byte.
func injectTitle(box, title string) string {
	lines := strings.Split(box, "\n")
	if len(lines) == 0 {
		return box
	}
	runes := []rune(stripANSI(lines[0]))
	if len(runes) < 6 {
		return box
	}

	// runes is: corner + N dashes + corner. Keep the total cell count identical
	// so the top border still matches the sides.
	const leadDashes = 2
	dash := string(runes[1])
	trailDashes := len(runes) - 2 - leadDashes - visLen(title)
	if trailDashes < 1 {
		return box
	}
	lines[0] = string(runes[0]) +
		strings.Repeat(dash, leadDashes) + title + strings.Repeat(dash, trailDashes) +
		string(runes[len(runes)-1])
	return strings.Join(lines, "\n")
}

// humanBytes formats a byte count at the scale release archives actually land
// in (~2-5 MB), falling back to KB for the small checksums file.
func humanBytes(n int64) string {
	const mb = 1 << 20
	if n < mb {
		return fmt.Sprintf("%d KB", n>>10)
	}
	return fmt.Sprintf("%.1f MB", float64(n)/mb)
}
