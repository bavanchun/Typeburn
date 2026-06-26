package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/bavanchun/Typeburn/internal/theme"
)

// renderHero renders the big-digit WPM block beside the acc/raw/consistency stat
// cards. During the reveal the WPM counts up in a fixed-width digit slot (no
// jitter) and each stat card stagger-fades in; once settled it is byte-identical
// to the static hero.
func (m ResultModel) renderHero(innerW int) string {
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
	accLine := revealLine(
		StatCard("acc", fmt.Sprintf("%.0f%%", accVal), accColorRole(accVal), m.th),
		cardProgress(0, m.revealStartMs, m.nowMs), m.th,
	)
	rawLine := revealLine(
		StatCard("raw", fmt.Sprintf("%.0f wpm", m.res.RawWPM), theme.RoleTextPrimary, m.th),
		cardProgress(1, m.revealStartMs, m.nowMs), m.th,
	)
	consLine := revealLine(
		StatCard("consistency", fmt.Sprintf("%.0f%%", m.res.Consistency), theme.RoleTextPrimary, m.th),
		cardProgress(2, m.revealStartMs, m.nowMs), m.th,
	)
	secondaryCol := strings.Join([]string{accLine, rawLine, consLine}, "\n")

	bigLines := strings.Split(bigWPM+"\n"+wpmLabel, "\n")
	secLines := strings.Split(secondaryCol, "\n")
	for len(bigLines) < len(secLines) {
		bigLines = append(bigLines, "")
	}
	for len(secLines) < len(bigLines) {
		secLines = append(secLines, "")
	}

	rows := make([]string, len(bigLines))
	for i := range bigLines {
		sec := ""
		if i < len(secLines) {
			sec = secLines[i]
		}
		rows[i] = bigLines[i] + "   " + sec
	}
	return strings.Join(rows, "\n")
}
