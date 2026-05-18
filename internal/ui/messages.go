// Package ui contains the Bubble Tea sub-models for each screen.
// Each sub-model implements Update(tea.Msg)(SubModel, tea.Cmd) and View() string;
// the root app.Model composes the View output and routes messages.
package ui

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/words"
)

// ResultMsg is emitted by TypingModel when a test finishes. The root model
// receives it and transitions to ScreenResult.
// QuoteLen carries the quote bucket so ResultModel can re-emit StartTestMsg
// with the correct bucket when the user restarts the same test.
type ResultMsg struct {
	Result   metrics.Result
	Mode     config.Mode
	Length   int
	QuoteLen words.QuoteLen
}
