package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/words"
)

// ResultModel is the sub-model for the post-test result screen. It renders
// big-digit WPM, accuracy/raw/consistency stats, a WPM-over-time sparkline,
// char counts, and test metadata — all from a completed metrics.Result.
//
// isBest is set by Phase 8 (history persistence) and gates the ★ new best
// badge. It defaults false.
type ResultModel struct {
	res      metrics.Result
	mode     config.Mode
	length   int
	quoteLen words.QuoteLen
	codeText string // ModeCode snippet, so restart-same re-runs it (not "")
	isBest   bool   // set externally by Phase 8; false = badge hidden

	w, h int
	th   theme.Theme
	km   config.Keymap
}

// NewResult constructs a ResultModel from a completed ResultMsg. The isBest
// field defaults to false; Phase 8 will set it after a history lookup.
func NewResult(msg ResultMsg, th theme.Theme, km config.Keymap) ResultModel {
	return ResultModel{
		res:      msg.Result,
		mode:     msg.Mode,
		length:   msg.Length,
		quoteLen: msg.QuoteLen,
		codeText: msg.CodeText,
		th:       th,
		km:       km,
	}
}

// SetSize stores terminal dimensions. Called by the root on WindowSizeMsg.
func (m ResultModel) SetSize(w, h int) ResultModel {
	m.w, m.h = w, h
	return m
}

// WithBest sets the isBest flag, enabling the ★ new best badge in the View.
// Called by the root after history persistence (Phase 8).
func (m ResultModel) WithBest(best bool) ResultModel {
	m.isBest = best
	return m
}

// Update handles key events for the result screen per design §8.4.
//
//   - tab / enter  → restart SAME test (same mode, length, quoteLen)
//   - ctrl+r       → new test (fresh pick via Home)
//   - esc / 1      → Home
//   - 3            → History (placeholder until Phase 8)
//   - ctrl+c       → quit (handled globally by root; forwarded here for completeness)
func (m ResultModel) Update(msg tea.Msg) (ResultModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	k := kp.Key()

	switch {
	case m.km.RestartSame.Matches(k), m.km.Start.Matches(k):
		// Restart same: re-emit StartTestMsg with identical params.
		return m, m.restartSameCmd()

	case m.km.NewTest.Matches(k):
		// New test: return to Home so user can pick new params.
		return m, func() tea.Msg { return AbortMsg{} }

	case m.km.Back.Matches(k), m.km.NavHome.Matches(k):
		return m, func() tea.Msg { return AbortMsg{} }

	case m.km.NavHistory.Matches(k):
		return m, func() tea.Msg { return NavHistoryMsg{} }
	}

	return m, nil
}

// restartSameCmd emits a StartTestMsg that re-creates an identical typing test.
func (m ResultModel) restartSameCmd() tea.Cmd {
	mode, length, ql, ct := m.mode, m.length, m.quoteLen, m.codeText
	return func() tea.Msg {
		return StartTestMsg{Mode: mode, Length: length, QuoteLen: ql, CodeText: ct}
	}
}

// NavHistoryMsg is emitted when the user navigates to the History screen from
// the Result screen. The root routes it to ScreenHistory.
type NavHistoryMsg struct{}
