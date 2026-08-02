package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// railGap is the smallest space allowed between a rail label and its value.
// The rail is rendered at its own natural width and then right-flushed, so this
// gap plus the widest label and value is all the horizontal travel the eye ever
// has to make — the reason the panel's inner width is capped rather than
// stretched to a 200-column terminal.
const railGap = 2

// railRow is one label/value pair in the comparison rail. A row with an empty
// value spans the block on its label alone, which is how the first-run state
// says so without inventing a number.
type railRow struct {
	label string
	short string
	value string
	role  theme.Role
}

// deltaValue renders a signed difference as a glyph plus a magnitude. The glyph
// is the information: a bare tinted number says nothing under NO_COLOR, and the
// mono theme renders error and primary text nearly identically.
func deltaValue(delta float64) (string, theme.Role) {
	switch {
	case delta > 0.5:
		return fmt.Sprintf("▲ %.0f wpm", delta), theme.RoleSuccess
	case delta < -0.5:
		return fmt.Sprintf("▼ %.0f wpm", -delta), theme.RoleWarning
	default:
		return "= 0 wpm", theme.RoleTextPrimary
	}
}

// railRows builds the comparison rail's six rows — one per big-digit row, so
// the rail and the hero always end on the same line.
//
// With no comparable history the rail says so and promotes this run's own
// secondary figures, which is what a brand-new profile sees on its first ever
// result.
func (m ResultModel) railRows() []railRow {
	raw := railRow{label: "raw", short: "raw",
		value: fmt.Sprintf("%.0f wpm", m.res.RawWPM), role: theme.RoleTextPrimary}
	cons := railRow{label: "consistency", short: "cons",
		value: fmt.Sprintf("%.0f%%", m.res.Consistency), role: theme.RoleTextPrimary}

	if !m.ctx.HasHistory {
		return []railRow{
			{label: "first run", short: "first run", role: theme.RoleTextMuted},
			{label: "no history yet", short: "no history", role: theme.RoleTextFaint},
			{},
			raw,
			cons,
			{},
		}
	}

	delta, deltaRole := deltaValue(m.res.NetWPM - m.ctx.PB)
	return []railRow{
		{label: "personal best", short: "pb",
			value: fmt.Sprintf("%.0f wpm", m.ctx.PB), role: theme.RoleTextPrimary},
		{label: "this run", short: "vs pb", value: delta, role: deltaRole},
		{label: "avg last 10", short: "avg10",
			value: fmt.Sprintf("%.0f wpm", m.ctx.Avg10), role: theme.RoleTextPrimary},
		raw,
		cons,
		{label: "rank", short: "rank", value: m.rankValue(), role: theme.RoleTextPrimary},
	}
}

// rankValue is this run's place in its bucket, or a plain statement that it has
// none — a run withheld from history never took a place, and printing one would
// be the same lie the withholding exists to prevent.
func (m ResultModel) rankValue() string {
	if m.ctx.Total <= 0 {
		return "not ranked"
	}
	return fmt.Sprintf("#%d of %d", m.ctx.Rank, m.ctx.Total)
}

// railLabel picks the full or abbreviated label for a row.
func railLabel(r railRow, short bool) string {
	if short {
		return r.short
	}
	return r.label
}

// railNaturalW measures the width the rail actually needs. Nothing here is a
// constant: the number of digits in a personal best and the length of a rank
// decide it, and a hardcoded guess is what makes a layout break on the first
// three-digit run.
func railNaturalW(rows []railRow, short bool) int {
	widest := 0
	for _, r := range rows {
		label := railLabel(r, short)
		if label == "" {
			continue
		}
		w := lipgloss.Width(label)
		if r.value != "" {
			w += railGap + lipgloss.Width(r.value)
		}
		if w > widest {
			widest = w
		}
	}
	return widest
}

// railLabelValueSpan reports the widest gap the eye has to cross between a
// label and the value it belongs to.
func railLabelValueSpan(rows []railRow, short bool) int {
	block := railNaturalW(rows, short)
	widest := 0
	for _, r := range rows {
		if r.value == "" {
			continue
		}
		if gap := block - lipgloss.Width(railLabel(r, short)) - lipgloss.Width(r.value); gap > widest {
			widest = gap
		}
	}
	return widest
}

// renderRail returns exactly len(rows) lines, each exactly colW cells wide, with
// the rail block right-flushed inside colW. Right-flushing is what puts ink at
// the panel's right edge; rendering the block at its natural width rather than
// stretching it is what keeps a label and its value readable as one pair.
//
// p is the reveal progress for the whole rail; a partially revealed line keeps
// its exact width, so the band never reflows mid-animation.
func renderRail(rows []railRow, colW int, short bool, p float64, th theme.Theme) []string {
	block := railNaturalW(rows, short)
	if block > colW {
		block = colW
	}
	pad := strings.Repeat(" ", colW-block)

	labelStyle := th.Style(theme.RoleTextMuted)
	out := make([]string, len(rows))
	for i, r := range rows {
		label := railLabel(r, short)
		if label == "" {
			out[i] = strings.Repeat(" ", colW)
			continue
		}
		styled := th.Style(r.role)
		if r.value == "" {
			label = cutCells(label, block)
			out[i] = pad + revealLine(styled.Render(label)+
				strings.Repeat(" ", block-lipgloss.Width(label)), p, th)
			continue
		}
		// The resolved rail column is never narrower than railNaturalW, so this
		// only bites for a caller that ignored the ladder. Cutting the label
		// keeps the band's per-line width exact instead of wrapping the border.
		value := cutCells(r.value, block)
		label = cutCells(label, block-1-lipgloss.Width(value))
		gap := block - lipgloss.Width(label) - lipgloss.Width(value)
		out[i] = pad + revealLine(
			labelStyle.Render(label)+strings.Repeat(" ", gap)+styled.Bold(true).Render(value), p, th)
	}
	return out
}

// padCell, cutCells and maxInt live in result_render_helpers.go beside the
// panel's other cell arithmetic.
