package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// StartTestMsg is emitted by HomeModel when the user presses enter/space.
// The root app.Model receives it, constructs a TypingModel, and transitions
// to ScreenTyping.
type StartTestMsg struct {
	Mode     config.Mode
	Length   int // seconds (Time) or word count (Words); 0 for Quote/Code
	QuoteLen words.QuoteLen
	// CodeText carries the user-supplied snippet when Mode==ModeCode.
	// Empty for all other modes.
	CodeText string
}

// modeLabels maps mode constants to display names shown in the tab row.
var modeLabels = []string{"Time", "Words", "Quote", "Code"}

// modeOrder is the cycle order for tab switching.
var modeOrder = []config.Mode{config.ModeTime, config.ModeWords, config.ModeQuote, config.ModeCode}

// quoteBucketLabels are the display labels for Quote sub-options.
var quoteBucketLabels = []string{"short", "medium", "long"}

// quoteBuckets maps option index to words.QuoteLen.
var quoteBuckets = []words.QuoteLen{words.QuoteShort, words.QuoteMedium, words.QuoteLong}

// HomeModel is the sub-model for the Home / Welcome screen. It holds the
// currently-selected mode and a per-mode length index so that switching tabs
// preserves each mode's choice.
type HomeModel struct {
	modeIdx  int                 // index into modeOrder / modeLabels
	lenIdx   map[config.Mode]int // selected option index per mode
	w, h     int
	th       theme.Theme
	km       config.Keymap
	codeText string // user-supplied snippet; empty = Code row disabled
	codeHint string // load-failure reason shown on Code row; empty = no error
}

// NewHome constructs a HomeModel seeded from s. The initial mode and length
// index are derived from s.DefaultMode and s.DefaultLength.
// codeText is the loaded snippet (empty = Code disabled); codeHint is the
// load-failure reason string (empty = loaded OK or no --text flag).
func NewHome(s config.Settings, th theme.Theme, km config.Keymap, codeText, codeHint string) HomeModel {
	// Resolve initial mode index.
	modeIdx := 0
	for i, m := range modeOrder {
		if m == s.DefaultMode {
			modeIdx = i
			break
		}
	}

	// Build per-mode length index map seeded to the middle option by default.
	// Quote defaults to index 1 (medium bucket). Code has no cycler; index is unused.
	lenIdx := make(map[config.Mode]int)
	for _, m := range modeOrder {
		lens := config.LengthsFor(m)
		if lens == nil {
			if m == config.ModeQuote {
				lenIdx[m] = 1 // default to "medium" bucket
			} else {
				lenIdx[m] = 0 // Code: no-op index, unused
			}
			continue
		}
		// Find the index of DefaultLength within this mode's option list.
		idx := len(lens) / 2 // fallback to mid
		for j, v := range lens {
			if v == s.DefaultLength && m == s.DefaultMode {
				idx = j
				break
			}
		}
		lenIdx[m] = idx
	}

	return HomeModel{
		modeIdx:  modeIdx,
		lenIdx:   lenIdx,
		th:       th,
		km:       km,
		codeText: codeText,
		codeHint: codeHint,
	}
}

// SetSize stores terminal dimensions for layout. Called by the root on
// tea.WindowSizeMsg.
func (m HomeModel) SetSize(w, h int) HomeModel {
	m.w, m.h = w, h
	return m
}

// WithCodeText returns a copy with ONLY codeText/codeHint replaced; modeIdx,
// lenIdx and sizes are preserved so the Code row stays selected after an
// in-app paste (a NewHome rebuild would snap modeIdx back to DefaultMode).
func (m HomeModel) WithCodeText(text, hint string) HomeModel {
	m.codeText = text
	m.codeHint = hint
	return m
}

// WithSettings returns a copy with theme and keymap updated, preserving
// modeIdx and lenIdx so the user's current selection is not reset when a
// setting changes (same preservation pattern as WithCodeText).
func (m HomeModel) WithSettings(th theme.Theme, km config.Keymap) HomeModel {
	m.th = th
	m.km = km
	return m
}

// currentMode returns the currently selected config.Mode.
func (m HomeModel) currentMode() config.Mode { return modeOrder[m.modeIdx] }

// optionCount returns the number of selectable options for the current mode.
// Code and Quote have no numeric length selector; Code returns 0, Quote returns
// the bucket count.
func (m HomeModel) optionCount() int {
	switch m.currentMode() {
	case config.ModeQuote:
		return len(quoteBucketLabels)
	case config.ModeCode:
		return 0 // no cycler; OptLeft/OptRight are no-ops
	default:
		return len(config.LengthsFor(m.currentMode()))
	}
}

// Update handles key events for the Home screen and returns an optional Cmd.
// It only processes messages when the screen is active (root delegates here).
func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd) {
	kp, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	k := kp.Key()

	switch {
	// Cycle mode forward.
	case m.km.NextMode.Matches(k):
		m.modeIdx = (m.modeIdx + 1) % len(modeOrder)

	// Cycle mode backward.
	case m.km.PrevMode.Matches(k):
		m.modeIdx = (m.modeIdx - 1 + len(modeOrder)) % len(modeOrder)

	// Decrease length option (clamped at 0, no-op for Code which has no cycler).
	case m.km.OptLeft.Matches(k):
		mode := m.currentMode()
		if mode != config.ModeCode && m.lenIdx[mode] > 0 {
			m.lenIdx[mode]--
		}

	// Increase length option (clamped at max, no-op for Code which has no cycler).
	case m.km.OptRight.Matches(k):
		mode := m.currentMode()
		if mode != config.ModeCode && m.lenIdx[mode] < m.optionCount()-1 {
			m.lenIdx[mode]++
		}

	// Start test.
	case m.km.Start.Matches(k):
		return m, m.startCmd()
	}

	return m, nil
}

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
