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
// CodeText carries the snippet text for ModeCode (used to store rune count and
// to allow ctrl+r restart with the same text).
type ResultMsg struct {
	Result   metrics.Result
	Mode     config.Mode
	Length   int
	QuoteLen words.QuoteLen
	CodeText string
}

// NavCodePasteMsg is emitted by HomeModel when the user presses enter/space
// on the Code row with no snippet loaded. The root model receives it and
// opens ScreenCodePaste. (When a --text snippet IS loaded, Code is enabled
// and the same keypress emits StartTestMsg instead — paste is not offered.)
type NavCodePasteMsg struct{}

// CodePastedMsg is emitted by CodePasteModel when a bracketed paste passes
// codetext.Normalize. Text is the normalized snippet, ready to use as a Code
// target verbatim. The root model receives it, stores the snippet, and
// returns to Home with Code enabled. No message is emitted on a failed paste
// (the sub-model stays and shows the reason) or on cancel (esc is handled by
// the global Back handler, not via a message).
type CodePastedMsg struct {
	Text string
}
