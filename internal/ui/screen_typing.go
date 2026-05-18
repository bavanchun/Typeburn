package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
	"monkeytype-tui/internal/typing"
	"monkeytype-tui/internal/words"
)

// TypingModel is the sub-model for the typing test screen. The root app.Model
// holds one TypingModel and routes messages to it when screen == ScreenTyping.
type TypingModel struct {
	eng    *typing.Engine
	mode   config.Mode
	length int // seconds (Time) or word count (Words); 0 for Quote
	ql     words.QuoteLen
	target string

	w, h int // terminal dimensions

	startMs     int64   // epoch-ms of first keystroke; 0 = not started
	nowMs       int64   // epoch-ms of last tick
	headerWPM   float64 // last computed live WPM for header
	lastPaintMs int64   // throttle: last time header WPM was recomputed

	blink bool // wired from settings (Phase 7); steady block default = false

	th   theme.Theme
	keys config.Keymap

	// seed used when ctrl+r generates a new test (0 = time-based random)
	seed int64
}

// AbortMsg is emitted when the user presses esc on the typing screen.
// The root model interprets it as "return to Home".
type AbortMsg struct{}

// NewTyping constructs a ready-to-use TypingModel. Target text is generated
// immediately so the screen can render before the first keystroke.
func NewTyping(
	mode config.Mode,
	length int,
	ql words.QuoteLen,
	th theme.Theme,
	km config.Keymap,
	blink bool,
) TypingModel {
	return newTypingWithSeed(mode, length, ql, th, km, blink, 0)
}

// newTypingWithSeed is the internal constructor used by NewTyping and ctrl+r.
// seed==0 uses a random time-based seed (production); non-zero is deterministic
// (tests).
func newTypingWithSeed(
	mode config.Mode,
	length int,
	ql words.QuoteLen,
	th theme.Theme,
	km config.Keymap,
	blink bool,
	seed int64,
) TypingModel {
	g := words.NewGenerator(seed)
	target := words.ForMode(g, mode, length, ql)

	// For ModeTime, wordTarget encodes the time limit in ms.
	wordTarget := length
	if mode == config.ModeTime {
		wordTarget = length * 1000
	}
	eng := typing.New(target, mode, wordTarget)

	return TypingModel{
		eng: eng, mode: mode, length: length, ql: ql,
		target: target, th: th, keys: km, blink: blink, seed: seed,
	}
}

// SetSize stores the terminal dimensions. Called by the root on WindowSizeMsg.
func (m TypingModel) SetSize(w, h int) TypingModel {
	m.w, m.h = w, h
	return m
}

// Update handles all messages while the typing screen is active.
func (m TypingModel) Update(msg tea.Msg) (TypingModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tickMsg:
		return m.handleTick(msg)
	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())
	case tea.PasteMsg:
		// Feed paste content as sequential runes through the typing engine.
		// This is the project-decided paste policy (see charm-v2-api-cheatsheet.md).
		return m.applyText(msg.Content)
	}
	return m, nil
}

// handleTick processes wall-clock ticks: recomputes WPM and checks time-mode
// completion.
func (m TypingModel) handleTick(msg tickMsg) (TypingModel, tea.Cmd) {
	m.nowMs = msg.t.UnixMilli()
	elapsed := elapsedMs(m.startMs, msg.t)

	// Recompute live WPM at ~250ms cadence to limit style recomputation.
	if m.nowMs-m.lastPaintMs >= 250 {
		m.headerWPM = liveWPM(m.eng.Log(), elapsed)
		m.lastPaintMs = m.nowMs
	}

	// Time-mode completion: when elapsed ≥ limit, end the test.
	if m.mode == config.ModeTime && m.startMs > 0 {
		if elapsed >= limitMs(m.length) {
			return m, m.completeCmd(limitMs(m.length) + m.startMs)
		}
	}
	return m, tickCmd()
}

// handleKey dispatches a key event to the appropriate action.
func (m TypingModel) handleKey(k tea.Key) (TypingModel, tea.Cmd) {
	switch {
	case m.keys.Quit.Matches(k):
		return m, tea.Quit
	case m.keys.Back.Matches(k):
		return m, func() tea.Msg { return AbortMsg{} }
	case m.keys.RestartSame.Matches(k):
		return m.restartSame(), nil
	case m.keys.NewTest.Matches(k):
		return m.newTest(), tickCmd()
	}

	// Backspace — no-op before test starts.
	if k.Code == tea.KeyBackspace {
		if m.startMs != 0 {
			m.eng.Backspace(m.nowMs)
		}
		return m, nil
	}

	// Printable characters.
	if k.Text != "" {
		return m.applyText(k.Text)
	}
	return m, nil
}

// applyText feeds printable runes into the engine and starts the timer on the
// first keystroke.
func (m TypingModel) applyText(text string) (TypingModel, tea.Cmd) {
	nowMs := time.Now().UnixMilli()
	firstKey := m.startMs == 0
	if firstKey {
		m.startMs = nowMs
		m.nowMs = nowMs
	}
	for _, r := range text {
		m.eng.Apply(r, nowMs)
	}
	// Words/Quote: check completion after each keystroke.
	if m.mode != config.ModeTime && m.eng.Complete(nowMs) {
		return m, m.completeCmd(nowMs)
	}
	if firstKey {
		return m, tickCmd()
	}
	return m, nil
}

// Test lifecycle actions (restartSame, newTest, completeCmd) live in
// screen_typing_actions.go to keep this file focused on Update/key handling.
