package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/anim"
	"github.com/bavanchun/Typeburn/internal/theme"
)

const (
	countUpMs   int64 = 600
	drawInMs    int64 = 500
	staggerMs   int64 = 120
	cardFadeMs  int64 = 220
	resultCards       = 3
)

// WithRevealStart arms the Result entrance reveal at startMs.
func (m ResultModel) WithRevealStart(startMs int64) ResultModel {
	m.revealStartMs = startMs
	m.nowMs = startMs
	return m
}

func resultRevealTotalMs() int64 {
	lastCardEnd := int64(resultCards-1)*staggerMs + cardFadeMs
	total := countUpMs
	if drawInMs > total {
		total = drawInMs
	}
	if lastCardEnd > total {
		total = lastCardEnd
	}
	return total
}

func revealProgress(startMs, nowMs, durMs int64) float64 {
	if startMs <= 0 {
		return 1
	}
	return anim.Tween{StartMs: startMs, DurMs: durMs}.Progress(nowMs)
}

func revealDone(startMs, nowMs int64) bool {
	return startMs <= 0 || nowMs >= startMs+resultRevealTotalMs()
}

func countUpValue(final int, startMs, nowMs int64) int {
	p := revealProgress(startMs, nowMs, countUpMs)
	return anim.LerpInt(0, final, anim.EaseOutQuad(p))
}

func sparkVisibleBars(total int, startMs, nowMs int64) int {
	if total <= 0 {
		return 0
	}
	p := revealProgress(startMs, nowMs, drawInMs)
	n := anim.LerpInt(0, total, anim.EaseOutQuad(p))
	if n > total {
		return total
	}
	return n
}

func cardProgress(idx int, startMs, nowMs int64) float64 {
	if startMs <= 0 {
		return 1
	}
	cardStart := startMs + int64(idx)*staggerMs
	return revealProgress(cardStart, nowMs, cardFadeMs)
}

func revealLine(s string, p float64, th theme.Theme) string {
	if p >= 1 {
		return s
	}
	plain := stripANSI(s)
	w := lipgloss.Width(plain)
	if p <= 0 {
		return strings.Repeat(" ", w)
	}
	return th.Style(theme.RoleTextFaint).Render(plain)
}

// BigDigitsFixed renders n with the same visual width as `final` so the count-up
// never shifts the Result hero block as the digit count grows (9 → 10 → 100).
func BigDigitsFixed(n, final int, th theme.Theme) string {
	out := BigDigits(n, th)
	finalW := maxLineWidth(BigDigits(final, th))
	if finalW <= 0 {
		return out
	}
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		if lipgloss.Width(line) < finalW {
			lines[i] = lipgloss.PlaceHorizontal(finalW, lipgloss.Right, line)
		}
	}
	return strings.Join(lines, "\n")
}

// maxLineWidth returns the widest rendered line width in s.
func maxLineWidth(s string) int {
	maxW := 0
	for _, line := range strings.Split(s, "\n") {
		if w := lipgloss.Width(line); w > maxW {
			maxW = w
		}
	}
	return maxW
}
