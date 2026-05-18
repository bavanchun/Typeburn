package ui

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
)

// View renders the full typing screen content as a string. The root app.Model
// wraps this in lipgloss.Place for centering — View itself does not center.
//
// Degraded mode (w<60 or h<20) is handled by the root View before delegation;
// this method is only called when the terminal is large enough.
func (m TypingModel) View() string {
	tier := WidthTier(m.w, m.h)
	cw := ContentWidth(m.w, tier)

	elapsed := elapsedMs(m.startMs, time.Now())
	done, total := m.eng.Progress()

	header := ModeHeader(
		m.mode,
		m.headerWPM,
		done, total,
		float64(elapsed)/1000.0,
		m.length,
		m.th,
	)

	stream := RenderWordStream(
		m.eng.States(),
		[]rune(m.target),
		typedFromLog(m.eng.Log()),
		cw,
		m.th,
	)

	footer := RenderFooter(TypingHints(), m.w, m.th)

	// Spacer fills remaining vertical space so footer pins to the last row.
	streamLines := strings.Count(stream, "\n") + 1
	used := 1 + // header
		1 + // blank line after header
		streamLines +
		1 + // blank line before footer
		1 // footer
	spacerLines := m.h - used
	if spacerLines < 0 {
		spacerLines = 0
	}
	spacer := strings.Repeat("\n", spacerLines)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		stream,
		spacer,
		footer,
	)
}
