package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// startCmd builds a Cmd that emits a StartTestMsg with the current selection.
// For ModeCode with no text loaded, it returns nil (no-op — stay on Home).
func (m HomeModel) startCmd() tea.Cmd {
	mode := m.currentMode()

	// Code mode: no snippet → open in-app paste; snippet (--text) → start.
	if mode == config.ModeCode {
		if m.codeText == "" {
			return func() tea.Msg { return NavCodePasteMsg{} }
		}
		ct := m.codeText
		return func() tea.Msg {
			return StartTestMsg{Mode: config.ModeCode, CodeText: ct}
		}
	}

	idx := m.lenIdx[mode]

	var length int
	var ql words.QuoteLen
	if mode == config.ModeQuote {
		ql = quoteBuckets[idx]
	} else {
		lens := config.LengthsFor(mode)
		length = lens[idx]
	}

	return func() tea.Msg {
		return StartTestMsg{Mode: mode, Length: length, QuoteLen: ql}
	}
}
