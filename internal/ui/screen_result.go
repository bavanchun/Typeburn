package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/update"
	"github.com/bavanchun/Typeburn/internal/words"
)

// ResultModel is the sub-model for the post-test result screen. It renders
// big-digit WPM, accuracy/raw/consistency stats, a WPM-over-time sparkline,
// char counts, and test metadata — all from a completed metrics.Result.
type ResultModel struct {
	res      metrics.Result
	mode     config.Mode
	length   int
	quoteLen words.QuoteLen
	codeText string // ModeCode snippet, so restart-same re-runs it (not "")
	isBest   bool   // set by the root after new-best detection; false = hidden

	// updateHint is set when an opportunistic check found a newer release.
	// Nil means no footer hint. Set via WithUpdateHint after NewResult.
	updateHint *update.Result

	w, h int
	th   theme.Theme
	km   config.Keymap

	// revealStartMs and nowMs drive the Result reveal. A zero revealStartMs
	// means static/settled rendering, which keeps direct NewResult callers stable.
	revealStartMs int64
	nowMs         int64
}

// NewResult constructs a ResultModel from a completed ResultMsg. The isBest
// field defaults to false; the root sets it after a history lookup.
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
func (m ResultModel) WithBest(best bool) ResultModel {
	m.isBest = best
	return m
}

// UpdateHint returns the current update hint (may be nil).
func (m ResultModel) UpdateHint() *update.Result { return m.updateHint }

// WithUpdateHint attaches an update hint to the result model.
// The hint renders as a single muted footer line if non-nil and the version
// string passes semver validation (injection guard applied at render time too).
func (m ResultModel) WithUpdateHint(hint *update.Result) ResultModel {
	m.updateHint = hint
	return m
}

// Update handles key events for the result screen.
//
//   - tab / enter  → restart SAME test (same mode, length, quoteLen)
//   - ctrl+r       → new test (fresh pick via Home)
//   - esc / 1      → Home
//   - 3            → History
//   - ctrl+c       → quit (handled globally by root; forwarded here for completeness)
func (m ResultModel) Update(msg tea.Msg) (ResultModel, tea.Cmd) {
	if ft, ok := msg.(FrameTickMsg); ok {
		m.nowMs = ft.T.UnixMilli()
		return m, nil
	}
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

// HasActiveAnim reports whether the result reveal OR a new-best celebration is
// still running at nowMs. The celebration window (celebrateMs) is longer than
// the reveal, so an isBest result keeps the frame loop alive until the burst
// settles; an ordinary result self-stops when the reveal finishes.
func (m ResultModel) HasActiveAnim(nowMs int64) bool {
	if !revealDone(m.revealStartMs, nowMs) {
		return true
	}
	return m.isBest && m.revealStartMs > 0 && nowMs < m.revealStartMs+celebrateMs
}

// NavHistoryMsg is emitted when the user navigates to the History screen from
// the Result screen. The root routes it to ScreenHistory.
type NavHistoryMsg struct{}
