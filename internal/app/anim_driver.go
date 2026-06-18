package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/ui"
)

// The frame-tick command and cadence live in package ui (ui.FrameTickCmd /
// ui.FrameInterval) so screen sub-models can also bootstrap the loop on the
// idle→active edge; the root only decides whether to re-arm via maybeFrameCmd.

// animActive reports whether any animation is live at nowMs: the active screen's
// own animation OR a root-owned transition. This is the signal that decides
// whether the frame loop re-arms — the basis of the self-stop behavior.
func (m Model) animActive(nowMs int64) bool {
	switch m.screen {
	case ScreenTyping:
		if m.typing.HasActiveAnim(nowMs) {
			return true
		}
	case ScreenResult:
		if m.result.HasActiveAnim(nowMs) {
			return true
		}
	}
	return m.transitionActive(nowMs)
}

// maybeFrameCmd returns a frame re-arm command iff an animation is still live at
// the current animNowMs, else nil. Returning nil is exactly what stops the loop.
func (m Model) maybeFrameCmd() tea.Cmd {
	if m.animActive(m.animNowMs) {
		return ui.FrameTickCmd()
	}
	return nil
}

// handleFrameTick stamps the shared animation clock (animNowMs), forwards the
// tick to the active screen so it can advance its own tween state, then re-arms
// the loop only while an animation remains live.
func (m Model) handleFrameTick(msg ui.FrameTickMsg) (tea.Model, tea.Cmd) {
	m.animNowMs = msg.T.UnixMilli()

	var subCmd tea.Cmd
	switch m.screen {
	case ScreenTyping:
		m.typing, subCmd = m.typing.Update(msg)
	case ScreenResult:
		m.result, subCmd = m.result.Update(msg)
	}
	return m, tea.Batch(subCmd, m.maybeFrameCmd())
}

// transitionActive reports whether a root-owned screen transition is mid-flight
// at nowMs. Transitions span two screens, so only the root can track them.
func (m Model) transitionActive(nowMs int64) bool {
	return m.transition != nil && nowMs < m.transition.startMs+m.transition.durMs
}
