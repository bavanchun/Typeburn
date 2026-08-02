package ui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// bestBadge and bestBadgeShort mark a new personal best. The short form is used
// when the WPM zone is too narrow for the full phrase — a one-digit run leaves
// four columns, and a badge that overlapped the accuracy label would break the
// band's column arithmetic.
const (
	bestBadge      = " ★ new best"
	bestBadgeShort = " ★"
)

// heroBand renders the label row plus the six big-digit rows as one block of
// lines, each exactly lay.InnerW cells wide.
//
// The band is three columns: the big-digit WPM, the accuracy zone, and the
// right-flushed comparison rail. Which of them get block art is decided by
// resolveHeroZones from widths measured off the very strings about to be
// rendered, because the digit count is what actually varies — a 100 wpm run is
// seven cells wider than an 87 wpm run, and the difference is the whole rail.
func (m ResultModel) heroBand(lay resultLayout) []string {
	finalWPM := int(math.Round(m.res.NetWPM))
	accVal := m.accForDisplay()
	accText := fmt.Sprintf("%.0f%%", accVal)

	wpmLines := m.wpmBlock(finalWPM)
	accBig := accBigBlock(int(math.Round(accVal)), m.th)
	rows := m.railRows()
	zones := m.heroZonesFor(lay.InnerW)

	if !zones.WPMBig {
		wpmLines = plainBlock(fmt.Sprintf("%d wpm", finalWPM), m.th.Style(theme.RoleAccent).Bold(true))
	}

	accP := cardProgress(0, m.revealStartMs, m.nowMs)
	accLines, accLabel := m.accZone(zones, accBig, accText, accP)

	railLines := make([]string, numRows)
	if zones.RailW > 0 {
		railLines = renderRail(rows, zones.RailW, zones.RailShort,
			cardProgress(1, m.revealStartMs, m.nowMs), m.th)
	}

	out := make([]string, 0, numRows+1)
	out = append(out, m.heroLabelRow(zones, accLabel, lay.InnerW))
	for i := 0; i < numRows; i++ {
		out = append(out, joinHeroRow(zones, lay.InnerW, wpmLines[i], accLines[i], railLines[i]))
	}
	return out
}

// heroZonesFor resolves the band's columns for an inner width.
//
// Every measurement is taken from the final values, never the ones mid-count-up:
// the zone widths have to be the same on the first animation frame as on the
// last, or the rail's column would slide sideways while the number climbs.
func (m ResultModel) heroZonesFor(innerW int) heroZones {
	finalWPM := int(math.Round(m.res.NetWPM))
	accVal := m.accForDisplay()
	rows := m.railRows()
	return resolveHeroZones(innerW, heroDemand{
		WPMW:       maxLineWidth(BigDigits(finalWPM, m.th)),
		WPMTextW:   lipgloss.Width(fmt.Sprintf("%d wpm", finalWPM)),
		AccBigW:    maxLineWidth(strings.Join(accBigBlock(int(math.Round(accVal)), m.th), "\n")),
		AccTextW:   maxInt(lipgloss.Width("acc"), lipgloss.Width(fmt.Sprintf("%.0f%%", accVal))),
		RailFullW:  railNaturalW(rows, false),
		RailShortW: railNaturalW(rows, true),
	})
}

// wpmBlock renders the WPM value as block art. During the count-up it is padded
// to the final value's width so the zone — and therefore the rail's column —
// never moves while the number climbs.
func (m ResultModel) wpmBlock(finalWPM int) []string {
	big := BigDigits(finalWPM, m.th)
	if !revealDone(m.revealStartMs, m.nowMs) {
		big = BigDigitsFixed(countUpValue(finalWPM, m.revealStartMs, m.nowMs), finalWPM, m.th)
	}
	return strings.Split(big, "\n")
}

// accBigBlock renders accuracy as block art with the percent sign parked on the
// glyphs' baseline row, so the unit is never carried by position alone.
func accBigBlock(acc int, th theme.Theme) []string {
	lines := strings.Split(BigDigits(acc, th), "\n")
	w := maxLineWidth(strings.Join(lines, "\n"))
	out := make([]string, len(lines))
	for i, ln := range lines {
		out[i] = ln + strings.Repeat(" ", w-lipgloss.Width(ln)) + "  "
		if i == len(lines)-1 {
			out[i] = ln + strings.Repeat(" ", w-lipgloss.Width(ln)) + " " +
				th.Style(theme.RoleTextMuted).Render("%")
		}
	}
	return out
}

// accZone returns the accuracy column's six rows and the label that belongs on
// the band's label row (empty when the text form carries its own label).
func (m ResultModel) accZone(z heroZones, big []string, text string, p float64) ([]string, string) {
	lines := make([]string, numRows)
	if z.AccBig {
		for i := range lines {
			lines[i] = revealLine(big[i], p, m.th)
		}
		return lines, "acc"
	}
	// Two lines centred against six: the label sits with its value instead of
	// five rows above it.
	const offset = (numRows - 2) / 2
	lines[offset] = revealLine(m.th.Style(theme.RoleTextMuted).Render("acc"), p, m.th)
	lines[offset+1] = revealLine(
		m.th.Style(accColorRole(m.accForDisplay())).Bold(true).Render(text), p, m.th)
	return lines, ""
}

// accForDisplay is the accuracy figure the screen shows: the keystroke-log form
// for a letter-strict run, the standard one otherwise.
func (m ResultModel) accForDisplay() float64 {
	if m.strict {
		return m.res.KeystrokeAccuracy
	}
	return m.res.Accuracy
}

// heroLabelRow places the zone labels over their columns. The new-best badge
// rides the WPM label and shortens rather than spilling into the next zone.
func (m ResultModel) heroLabelRow(z heroZones, accLabel string, innerW int) string {
	// The accuracy label owns the first column of its own zone, so the badge may
	// run into the gutter but must stop one cell short of it.
	accCol := z.WPMW + z.Gutter
	badge := ""
	if m.isBest {
		switch {
		case lipgloss.Width("wpm"+bestBadge) < accCol:
			badge = bestBadge
		case lipgloss.Width("wpm"+bestBadgeShort) < accCol:
			badge = bestBadgeShort
		}
	}

	left := m.th.Style(theme.RoleTextMuted).Render("wpm")
	if badge != "" {
		left += m.th.Style(theme.RoleSuccess).Render(badge)
	}
	if accLabel == "" {
		return padCell(left, innerW)
	}
	gap := maxInt(accCol-lipgloss.Width("wpm"+badge), 1)
	return padCell(left+strings.Repeat(" ", gap)+
		m.th.Style(theme.RoleTextMuted).Render(accLabel), innerW)
}

// joinHeroRow lays one band row out across the resolved zones and pads the
// result to exactly innerW, so every panel line is the width the layout claims.
func joinHeroRow(z heroZones, innerW int, wpm, acc, rail string) string {
	var b strings.Builder
	b.WriteString(padCell(wpm, z.WPMW))
	if z.AccW > 0 {
		b.WriteString(strings.Repeat(" ", z.Gutter))
		b.WriteString(padCell(acc, z.AccW))
	}
	if z.RailW > 0 {
		b.WriteString(strings.Repeat(" ", z.Gutter))
		b.WriteString(rail)
	}
	return padCell(b.String(), innerW)
}

// plainBlock renders s as a numRows-tall block so a value too wide for block
// art still occupies the hero's shape.
func plainBlock(s string, style lipgloss.Style) []string {
	out := make([]string, numRows)
	for i := range out {
		out[i] = ""
	}
	out[numRows/2] = style.Render(s)
	return out
}
