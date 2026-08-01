package ui

import (
	"fmt"
	"math"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// heroBlockGap separates the WPM big-digit block from the acc block.
const heroBlockGap = "     "

// renderHero renders the two-big-number hero: the ASCII big-digit WPM block
// (left) beside a prominent acc block (right), with raw + consistency as a
// secondary card row underneath. During the reveal the WPM counts up in a
// fixed-width digit slot (no jitter), the acc block and secondary cards
// stagger-fade in; once settled it is byte-identical to the static hero.
// The hero takes no width: the panel is capped (see resultMaxContentW), so the
// hero's fixed-width blocks always fit. It previously accepted an innerW it
// discarded, which is how it drifted out of the layout system in the first
// place — if it ever needs to adapt, take the width then and use it.
func (m ResultModel) renderHero() string {
	finalWPM := int(math.Round(m.res.NetWPM))
	displayWPM := countUpValue(finalWPM, m.revealStartMs, m.nowMs)
	bigWPM := BigDigits(finalWPM, m.th)
	if !revealDone(m.revealStartMs, m.nowMs) {
		bigWPM = BigDigitsFixed(displayWPM, finalWPM, m.th)
	}

	wpmLabel := m.th.Style(theme.RoleTextMuted).Render("wpm")
	if m.isBest {
		wpmLabel += m.th.Style(theme.RoleSuccess).Render(" ★ new best")
	}

	accVal := m.res.Accuracy
	if m.strict {
		accVal = m.res.KeystrokeAccuracy
	}
	accP := cardProgress(0, m.revealStartMs, m.nowMs)
	accBlock := []string{
		revealLine(m.th.Style(theme.RoleTextMuted).Render("acc"), accP, m.th),
		revealLine(m.th.Style(accColorRole(accVal)).Bold(true).Render(fmt.Sprintf("%.0f%%", accVal)), accP, m.th),
	}

	// Manual side-by-side join: acc block vertically centered against the digit
	// rows. Digit rows share one width; only rows that carry acc content get
	// padded, so blank right-hand space never trails other rows.
	bigLines := strings.Split(bigWPM+"\n"+wpmLabel, "\n")
	bigW := maxLineWidth(bigWPM)
	offset := (len(bigLines) - len(accBlock)) / 2
	if offset < 0 {
		offset = 0
	}

	rows := make([]string, len(bigLines))
	for i, line := range bigLines {
		rows[i] = line
		if i >= offset && i-offset < len(accBlock) {
			pad := bigW - lipgloss.Width(line)
			if pad < 0 {
				pad = 0
			}
			rows[i] += strings.Repeat(" ", pad) + heroBlockGap + accBlock[i-offset]
		}
	}

	rawCard := revealLine(
		StatCard("raw", fmt.Sprintf("%.0f wpm", m.res.RawWPM), theme.RoleTextPrimary, m.th),
		cardProgress(1, m.revealStartMs, m.nowMs), m.th,
	)
	consCard := revealLine(
		StatCard("consistency", fmt.Sprintf("%.0f%%", m.res.Consistency), theme.RoleTextPrimary, m.th),
		cardProgress(2, m.revealStartMs, m.nowMs), m.th,
	)

	return strings.Join(rows, "\n") + "\n\n" + rawCard + "   " + consCard
}
