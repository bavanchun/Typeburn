package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// The 100ms timer chain: one tea.Tick that re-arms itself for as long as the
// typing screen is live. It drives the header's live WPM and Time-mode
// completion. Chains never merge, so how many exist is a correctness property,
// not a detail — see tickLoopEnded on TypingModel.

// handleTick processes wall-clock ticks: recomputes WPM and checks time-mode
// completion.
func (m TypingModel) handleTick(msg tickMsg) (TypingModel, tea.Cmd) {
	m.nowMs = msg.t.UnixMilli()
	elapsed := elapsedMs(m.startMs, msg.t)

	// Recompute live WPM at ~250ms cadence to limit style recomputation.
	if m.nowMs-m.lastPaintMs >= 250 {
		m.headerWPM = liveWPMFromCount(m.eng.ForwardKeystrokes(), elapsed)
		m.lastPaintMs = m.nowMs
	}

	// Time-mode completion: when elapsed ≥ limit, end the test.
	if m.mode == config.ModeTime && m.startMs > 0 {
		if elapsed >= limitMs(m.length) {
			// This chain stops here — nothing re-arms it, so a later restart
			// on this model has to start a fresh one.
			m.tickLoopEnded = true
			return m, m.completeCmd(limitMs(m.length) + m.startMs)
		}
	}
	return m, tickCmd()
}

// armTickLoop starts the 100ms timer chain, or returns no command when one is
// already running. The tick drives the live WPM header and Time-mode
// completion; one chain does that, and every extra chain only burns wakeups.
func (m TypingModel) armTickLoop() (TypingModel, tea.Cmd) {
	if !m.tickLoopEnded {
		return m, nil
	}
	m.tickLoopEnded = false
	return m, tickCmd()
}
