package ui

import (
	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/metrics"
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
