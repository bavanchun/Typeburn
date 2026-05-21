package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/runner"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// restartSame resets engine and timer but keeps the same target text.
func (m TypingModel) restartSame() TypingModel {
	m.eng = runner.RebuildEngine(m.target, m.mode, m.length)
	m.startMs, m.nowMs, m.lastPaintMs, m.headerWPM = 0, 0, 0, 0
	return m
}

// newTest regenerates target text and resets everything.
// For ModeCode, the same code text is reused (no random re-generation).
func (m TypingModel) newTest() TypingModel {
	var fresh TypingModel
	if m.mode == config.ModeCode {
		fresh = NewTypingCode(m.target, m.th, m.keys, m.blink)
	} else {
		fresh = newTypingWithSeed(m.mode, m.length, m.ql, m.th, m.keys, m.blink, 0)
	}
	fresh.w, fresh.h = m.w, m.h
	return fresh
}

// completeCmd returns a Cmd that emits a ResultMsg carrying computed metrics.
// QuoteLen is forwarded so ResultModel can restart the same quote bucket.
// CodeText is forwarded so the root can persist rune count and allow restart.
func (m TypingModel) completeCmd(endMs int64) tea.Cmd {
	log := m.eng.Log()
	result := metrics.Compute(log, m.mode, endMs)
	mode, length, ql, ct := m.mode, m.length, m.ql, m.target
	return func() tea.Msg {
		return ResultMsg{Result: result, Mode: mode, Length: length, QuoteLen: ql, CodeText: ct}
	}
}

// TargetText returns the current target string for the typing test.
// Used by external callers (e.g. integration tests in package app) that need
// to replay the exact character sequence to drive the engine to completion.
func (m TypingModel) TargetText() string { return m.target }

// ApplySettings updates the blink flag and theme from new settings without
// restarting the test. Used by the root settings-change handler for live
// propagation.
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
