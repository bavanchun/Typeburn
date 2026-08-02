package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/runner"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
	"github.com/bavanchun/Typeburn/v2/internal/words"
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

	blink       bool // wired from settings; false = steady block, true = blinking
	strict      bool
	punctuation bool
	numbers     bool

	th   theme.Theme
	keys config.Keymap

	// seed used when ctrl+r generates a new test (0 = time-based random)
	seed int64

	// Caret animation state. lastKeyMs is the time of the most recent keystroke
	// (drives the new-cell fade + trail window). frameLoopArmed guards the
	// idle→active edge so exactly one 33ms frame loop runs regardless of typing
	// speed. nowFn is the injectable clock seam (defaults to time.Now) so caret
	// goldens are deterministic. wordCache holds the static prefix of styled
	// word-stream tokens so only the ≤3 animated cells re-Render per frame.
	lastKeyMs      int64
	frameLoopArmed bool
	nowFn          func() int64
	wordCache      *streamTokenCache

	// tickLoopEnded records that the 100ms timer chain has stopped. The root
	// starts one chain with InitCmd as soon as it builds the model and every
	// tickMsg re-arms it, so a fresh model always has exactly one running —
	// hence the zero value means "running". Restart paths consult this instead
	// of starting another chain: tea.Tick chains never merge, so a second start
	// leaves two loops ticking forever and both racing to end the same test.
	tickLoopEnded bool
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
	strict bool,
	punctuation bool,
	numbers bool,
) TypingModel {
	return newTypingWithSeed(mode, length, ql, th, km, blink, strict, punctuation, numbers, 0)
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
	strict bool,
	punctuation bool,
	numbers bool,
	seed int64,
) TypingModel {
	s := runner.NewSession(mode, length, ql, seed, strict, punctuation, numbers)
	return TypingModel{
		eng: s.Engine, mode: s.Mode, length: s.Length, ql: s.QuoteLen,
		target: s.Target, th: th, keys: km, blink: blink, strict: strict,
		punctuation: punctuation, numbers: numbers, seed: seed,
		nowFn: defaultNowFn, wordCache: &streamTokenCache{},
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
	case FrameTickMsg:
		// Animation frame: store the shared clock so View-side caret tweens can
		// advance. Never touches WPM/completion — that is the timer tick's job.
		// When the fade window has closed, disarm so the root's re-arm check stops
		// the loop and the next keystroke can bootstrap a fresh one.
		m.nowMs = msg.T.UnixMilli()
		if !m.HasActiveAnim(m.nowMs) {
			m.frameLoopArmed = false
		}
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())
	case tea.PasteMsg:
		// Feed paste content as sequential runes through the typing engine.
		// This is the project-decided paste policy (see charm-v2-api-cheatsheet.md).
		return m.applyText(msg.Content)
	}
	return m, nil
}

// handleKey dispatches a key event to the appropriate action.
func (m TypingModel) handleKey(k tea.Key) (TypingModel, tea.Cmd) {
	switch {
	case m.keys.Quit.Matches(k):
		return m, tea.Quit
	case m.keys.Back.Matches(k):
		return m, func() tea.Msg { return AbortMsg{} }
	case m.keys.RestartSame.Matches(k):
		// Keep the Time-mode header/timer live across a restart. The running
		// chain already does that — it idles harmlessly until the first
		// keystroke sets startMs (elapsedMs/completion are guarded on startMs).
		return m.restartSame().armTickLoop()
	case m.keys.NewTest.Matches(k):
		fresh := m.newTest()
		fresh.tickLoopEnded = m.tickLoopEnded // a new target, the same chain
		return fresh.armTickLoop()
	}

	// Backspace — no-op before test starts.
	if k.Code == tea.KeyBackspace {
		if m.startMs != 0 {
			m.eng.Backspace(m.nowMs)
			m.wordCache.invalidate() // engine state changed; rebuild base tokens
		}
		return m, nil
	}

	// Printable characters.
	if k.Text != "" {
		return m.applyText(k.Text)
	}
	return m, nil
}

// The 100ms timer chain (handleTick, armTickLoop) lives in
// screen_typing_tick.go; keystroke application (applyText) and the caret clock
// seam live in screen_typing_input.go; caret animation hooks (HasActiveAnim,
// InitCmd, caretAnimState) in screen_typing_caret.go; test lifecycle actions
// (restartSame, newTest, completeCmd) in screen_typing_actions.go — keeping
// this file focused on Update/message routing and key dispatch.
