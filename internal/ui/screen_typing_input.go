package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// defaultNowFn is the production caret clock: wall-clock epoch milliseconds.
func defaultNowFn() int64 { return time.Now().UnixMilli() }

// nowMillis returns the current epoch-ms via the injectable clock seam, falling
// back to wall-clock when nowFn is unset (e.g. a zero-value model in a test).
func (m TypingModel) nowMillis() int64 {
	if m.nowFn != nil {
		return m.nowFn()
	}
	return time.Now().UnixMilli()
}

// applyText feeds printable runes into the engine and starts the timer on the
// first keystroke. It also records the keystroke time (driving the caret fade +
// trail), invalidates the token cache, and bootstraps the 33ms frame loop on the
// idle→active edge only.
func (m TypingModel) applyText(text string) (TypingModel, tea.Cmd) {
	nowMs := m.nowMillis()
	firstKey := m.startMs == 0
	if firstKey {
		m.startMs = nowMs
		m.nowMs = nowMs
	}
	for _, r := range text {
		m.eng.Apply(r, nowMs)
	}
	m.lastKeyMs = nowMs
	m.wordCache.invalidate() // engine state changed; rebuild base tokens

	// Words/Quote: check completion after each keystroke.
	if m.mode != config.ModeTime && m.eng.Complete(nowMs) {
		return m, m.completeCmd(nowMs)
	}

	var cmds []tea.Cmd
	if firstKey {
		cmds = append(cmds, tickCmd())
	}
	// Bootstrap the frame loop ONLY on the idle→active edge: returning it per
	// keystroke would multiply overlapping tea.Tick timers that never self-stop.
	if !m.frameLoopArmed {
		m.frameLoopArmed = true
		cmds = append(cmds, FrameTickCmd())
	}
	return m, tea.Batch(cmds...)
}
