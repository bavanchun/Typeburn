// Package ui contains the Bubble Tea sub-models for each screen.
// Each sub-model implements Update(tea.Msg)(SubModel, tea.Cmd) and View() string;
// the root app.Model composes the View output and routes messages.
package ui

import (
	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/metrics"
)

// ResultMsg is emitted by TypingModel when a test finishes. The root model
// receives it and transitions to ScreenResult (Phase 6). Until Phase 6 lands
// the root may route to a placeholder.
type ResultMsg struct {
	Result metrics.Result
	Mode   config.Mode
	Length int
}
