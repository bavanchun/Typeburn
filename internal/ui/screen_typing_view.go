package ui

import (
	"fmt"
	"strings"
	"time"

	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/theme"
)

// View renders the full typing screen content as a string. The root app.Model
// wraps this in lipgloss.Place for centering — View itself does not center.
func (m TypingModel) View() string {
	if m.w < 60 || m.h < 20 {
		return m.degradedView()
	}

	cw := contentWidth(m.w)
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
	// Count actual lines used by the stream (newline-separated).
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

// degradedView renders the "terminal too small" notice per design §4.3.
// Never crashes, never partially paints — always a safe centered stub.
func (m TypingModel) degradedView() string {
	warn := m.th.Style(theme.RoleWarning).Render("Terminal too small")
	info := m.th.Style(theme.RoleTextMuted).Render(
		fmt.Sprintf("Need at least 60×20 (now %d×%d)", m.w, m.h),
	)
	hint := m.th.Style(theme.RoleTextFaint).Render("Resize to continue · ctrl+c quit")
	return lipgloss.JoinVertical(lipgloss.Center, warn, info, hint)
}
