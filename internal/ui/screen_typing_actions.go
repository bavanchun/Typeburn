package ui

import (
	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/metrics"
	"monkeytype-tui/internal/theme"
	"monkeytype-tui/internal/typing"
)

// restartSame resets engine and timer but keeps the same target text.
func (m TypingModel) restartSame() TypingModel {
	wordTarget := m.length
	if m.mode == config.ModeTime {
		wordTarget = m.length * 1000
	}
	m.eng = typing.New(m.target, m.mode, wordTarget)
	m.startMs, m.nowMs, m.lastPaintMs, m.headerWPM = 0, 0, 0, 0
	return m
}

// newTest regenerates target text and resets everything.
func (m TypingModel) newTest() TypingModel {
	fresh := newTypingWithSeed(m.mode, m.length, m.ql, m.th, m.keys, m.blink, 0)
	fresh.w, fresh.h = m.w, m.h
	return fresh
}

// completeCmd returns a Cmd that emits a ResultMsg carrying computed metrics.
// QuoteLen is forwarded so ResultModel can restart the same quote bucket.
func (m TypingModel) completeCmd(endMs int64) tea.Cmd {
	log := m.eng.Log()
	result := metrics.Compute(log, m.mode, endMs)
	mode, length, ql := m.mode, m.length, m.ql
	return func() tea.Msg {
		return ResultMsg{Result: result, Mode: mode, Length: length, QuoteLen: ql}
	}
}

// TargetText returns the current target string for the typing test.
// Used by external callers (e.g. integration tests in package app) that need
// to replay the exact character sequence to drive the engine to completion.
func (m TypingModel) TargetText() string { return m.target }

// ApplySettings updates the blink flag and theme from new settings without
// restarting the test. Used by the root onChange callback for live propagation.
func (m TypingModel) ApplySettings(s config.Settings, th theme.Theme) TypingModel {
	m.blink = s.BlinkCursor
	m.th = th
	return m
}

// ApplyTheme swaps the theme on the result model for live theme propagation.
func (m ResultModel) ApplyTheme(th theme.Theme) ResultModel {
	m.th = th
	return m
}
