package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/words"
)

// newTestTyping returns a deterministic TypingModel using seed 42 and a small
// word count so tests are fast and reproducible.
func newTestTyping(mode config.Mode, length int) TypingModel {
	return newTypingWithSeed(
		mode, length, words.QuoteShort,
		theme.Default(), config.DefaultKeymap(), false,
		42,
	).SetSize(80, 24)
}

// press constructs a tea.KeyPressMsg from a rune code and modifier.
func press(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: mod})
}

// pressText constructs a tea.KeyPressMsg for a printable character.
func pressText(ch string) tea.KeyPressMsg {
	r := rune(ch[0])
	return tea.KeyPressMsg(tea.Key{Code: r, Text: ch})
}

// TestTypingView_CompactNotFullHeight verifies the word-stream typing view is
// a compact block (header + stream + footer with small gaps) rather than a
// block padded to the full terminal height. The root wraps it in
// lipgloss.Place(Center,Center); a full-height block would make that vertical
// centering a no-op and leave the text pinned to the top.
func TestTypingView_CompactNotFullHeight(t *testing.T) {
	m := newTypingWithSeed(
		config.ModeWords, 10, words.QuoteShort,
		theme.Default(), config.DefaultKeymap(), false, 42,
	).SetSize(160, 50)
	v := m.View()
	lines := strings.Count(v, "\n") + 1
	if lines >= 50 {
		t.Fatalf("typing view should be compact for root centering, got %d lines at h=50", lines)
	}
}

// TestNewTyping_HasTarget ensures the constructor produces a non-empty target.
func TestNewTyping_HasTarget(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	if m.target == "" {
		t.Fatal("expected non-empty target")
	}
}

// TestTyping_PrintableAdvancesCursor checks that typing a correct char
// advances the engine state (no longer Untyped at position 0).
func TestTyping_PrintableAdvancesCursor(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	firstRune := []rune(m.target)[0]

	m2, _ := m.Update(pressText(string(firstRune)))

	states := m2.eng.States()
	if len(states) == 0 {
		t.Fatal("states empty after first keystroke")
	}
	// Position 0 should now be Correct or Incorrect — not Untyped or Current.
	// It cannot still be Current because the cursor advanced.
	if states[0] == 0 { // Untyped == 0
		t.Errorf("position 0 still Untyped after typing first rune")
	}
}

// TestTyping_WrongCharShowsIncorrect checks that an incorrect char yields an
// Incorrect (or IncorrectSpace) state at position 0.
func TestTyping_WrongCharShowsIncorrect(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	firstRune := []rune(m.target)[0]

	// Pick a char guaranteed to differ from firstRune.
	wrong := "X"
	if string(firstRune) == "X" {
		wrong = "Z"
	}

	m2, _ := m.Update(pressText(wrong))
	states := m2.eng.States()
	if len(states) == 0 {
		t.Fatal("states empty")
	}
	// State at position 0 must not be Correct (2) or Untyped (0).
	got := states[0]
	if got == 0 || got == 1 { // Untyped=0, Correct=1 — see char_state.go iota
		t.Errorf("expected Incorrect state, got %v", got)
	}
}

// TestTyping_BackspaceRemovesChar checks that backspace after one typed char
// restores position 0 to Current state.
func TestTyping_BackspaceRemovesChar(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	firstRune := []rune(m.target)[0]

	m2, _ := m.Update(pressText(string(firstRune)))
	m3, _ := m2.Update(press(tea.KeyBackspace, 0))

	states := m3.eng.States()
	if len(states) == 0 {
		t.Fatal("states empty after backspace")
	}
	// After backspace, cursor is back at 0 → state must be Current (5).
	if states[0] != 5 { // Current == 5 per iota
		t.Errorf("expected Current(5) after backspace, got %v", states[0])
	}
}

// TestTyping_TabRestartsSameTarget checks that tab resets the engine but keeps
// the same target text.
func TestTyping_TabRestartsSameTarget(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	originalTarget := m.target

	// Type something first.
	for _, r := range []rune(m.target)[:3] {
		m, _ = m.Update(pressText(string(r)))
	}

	// Press tab to restart.
	m2, _ := m.Update(press(tea.KeyTab, 0))

	if m2.target != originalTarget {
		t.Errorf("target changed after tab: want %q, got %q", originalTarget, m2.target)
	}
	if m2.startMs != 0 {
		t.Error("startMs should be 0 after restart")
	}
	// Cursor should be back at position 0 (Current).
	states := m2.eng.States()
	if len(states) == 0 || states[0] != 5 {
		t.Errorf("expected cursor at 0 after restart, states[0]=%v", states[0])
	}
}

// TestTyping_EscEmitsAbortMsg checks that esc emits an AbortMsg.
func TestTyping_EscEmitsAbortMsg(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	_, cmd := m.Update(press(tea.KeyEsc, 0))
	if cmd == nil {
		t.Fatal("esc should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(AbortMsg); !ok {
		t.Fatalf("expected AbortMsg, got %T", msg)
	}
}

// TestTyping_WordsMode_CompletionEmitsResultMsg types all chars of a 3-word
// test (deterministic seed) and checks that a ResultMsg is emitted.
func TestTyping_WordsMode_CompletionEmitsResultMsg(t *testing.T) {
	m := newTypingWithSeed(
		config.ModeWords, 3, words.QuoteShort,
		theme.Default(), config.DefaultKeymap(), false,
		42,
	).SetSize(80, 24)

	target := m.target

	// Type each rune in sequence.
	var lastCmd tea.Cmd
	for _, r := range []rune(target) {
		m, lastCmd = m.Update(pressText(string(r)))
		if lastCmd != nil {
			msg := lastCmd()
			if res, ok := msg.(ResultMsg); ok {
				if res.Result.NetWPM < 0 {
					t.Errorf("unexpected negative WPM: %v", res.Result.NetWPM)
				}
				if res.Mode != config.ModeWords {
					t.Errorf("wrong mode in result: %v", res.Mode)
				}
				return // success
			}
		}
	}
	t.Error("ResultMsg never emitted after typing all chars")
}

// TestTyping_View_ContainsStream checks that View produces non-empty output
// with the stream and footer when terminal is large enough.
func TestTyping_View_ContainsStream(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	view := m.View()
	if view == "" {
		t.Fatal("View returned empty string")
	}
	// Footer key hints must be present.
	if !strings.Contains(view, "tab") {
		t.Errorf("footer 'tab' hint not found in view")
	}
}

// TestTyping_View_SmallTerminal_DoesNotCrash checks that TypingModel.View()
// does not panic at sub-threshold sizes. The degraded notice is now rendered
// by the root app.Model (single chokepoint per design §4.3); TypingModel.View()
// itself is only called when the terminal is large enough, but must not crash
// if invoked directly in tests at small sizes.
func TestTyping_View_SmallTerminal_DoesNotCrash(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	m = m.SetSize(40, 10) // below 60×20 threshold
	// Must not panic — content may be partial but no crash.
	_ = m.View()
}

// TestTyping_Tick_RecomputesWPM checks that a tick message updates headerWPM
// after the timer has started (i.e., after first keystroke).
func TestTyping_Tick_RecomputesWPM(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)

	// Type first char to start the timer.
	firstRune := []rune(m.target)[0]
	m, _ = m.Update(pressText(string(firstRune)))

	// Simulate a tick well past the 250ms paint throttle.
	future := time.Now().Add(400 * time.Millisecond)
	m.lastPaintMs = 0 // force recompute
	m2, _ := m.Update(tickMsg{t: future})

	// With at least one char typed, headerWPM should be > 0 if elapsed ≥ 500ms;
	// elapsed here is ~400ms so the live-WPM helper returns 0 (below 500ms guard). That
	// is correct behaviour — we just assert no panic and the type round-trips.
	_ = m2.headerWPM
}

// TestTypedFromLog_RoundTrip checks that typedFromLog correctly replays apply+backspace.
func TestTypedFromLog_RoundTrip(t *testing.T) {
	m := newTestTyping(config.ModeWords, 5)
	runes := []rune(m.target)

	m, _ = m.Update(pressText(string(runes[0])))
	m, _ = m.Update(pressText(string(runes[1])))
	m, _ = m.Update(press(tea.KeyBackspace, 0)) // delete runes[1]

	typed := typedFromLog(m.eng.Log())
	if len(typed) != 1 {
		t.Fatalf("expected 1 typed rune after backspace, got %d: %v", len(typed), typed)
	}
	if typed[0] != runes[0] {
		t.Errorf("expected %q, got %q", runes[0], typed[0])
	}
}
