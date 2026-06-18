package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// FrameInterval is the cadence of the self-stopping animation frame loop. It is
// independent of the 100ms timer tick (timer.go): the frame loop only runs while
// at least one screen — or a root-owned transition — reports a live animation,
// and self-stops back to zero scheduled ticks once everything settles.
//
// It lives in package ui (not app) so both the root driver's re-arm and a screen
// sub-model's bootstrap-on-edge can schedule the same single-fire tick.
const FrameInterval = 33 * time.Millisecond

// FrameTickCmd schedules a single animation frame tick producing a FrameTickMsg.
// The loop stays alive only by re-arming this in Update while an animation is
// live (mirrors timer.go's single-fire re-arm), so an idle app schedules none.
func FrameTickCmd() tea.Cmd {
	return tea.Tick(FrameInterval, func(t time.Time) tea.Msg {
		return FrameTickMsg{T: t}
	})
}
