package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/words"
)

// helpers

func newTestHome() HomeModel {
	return NewHome(config.Defaults(), theme.Default(), config.DefaultKeymap(), "", "")
}

func pressKey(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: mod})
}

func pressTab() tea.KeyPressMsg      { return pressKey(tea.KeyTab, 0) }
func pressShiftTab() tea.KeyPressMsg { return pressKey(tea.KeyTab, tea.ModShift) }
func pressRight() tea.KeyPressMsg    { return pressKey(tea.KeyRight, 0) }
func pressLeft() tea.KeyPressMsg     { return pressKey(tea.KeyLeft, 0) }
func pressEnter() tea.KeyPressMsg    { return pressKey(tea.KeyEnter, 0) }

// TestNewHomeSeededFromSettings verifies that the initial mode and length index
// are derived from config.Settings (DefaultMode=Time, DefaultLength=30).
func TestNewHomeSeededFromSettings(t *testing.T) {
	h := newTestHome()
	if h.currentMode() != config.ModeTime {
		t.Fatalf("want ModeTime, got %v", h.currentMode())
	}
	// DefaultLength=30 is index 1 in [15,30,60,120].
	if h.lenIdx[config.ModeTime] != 1 {
		t.Fatalf("want lenIdx[Time]=1 (30s), got %d", h.lenIdx[config.ModeTime])
	}
}

// TestTabCyclesModeForward verifies tab → Time→Words→Quote→Code→Time.
func TestTabCyclesModeForward(t *testing.T) {
	h := newTestHome()

	h, _ = h.Update(pressTab())
	if h.currentMode() != config.ModeWords {
		t.Fatalf("after 1st tab: want ModeWords, got %v", h.currentMode())
	}

	h, _ = h.Update(pressTab())
	if h.currentMode() != config.ModeQuote {
		t.Fatalf("after 2nd tab: want ModeQuote, got %v", h.currentMode())
	}

	h, _ = h.Update(pressTab())
	if h.currentMode() != config.ModeCode {
		t.Fatalf("after 3rd tab: want ModeCode, got %v", h.currentMode())
	}

	h, _ = h.Update(pressTab())
	if h.currentMode() != config.ModeTime {
		t.Fatalf("after 4th tab (wrap): want ModeTime, got %v", h.currentMode())
	}
}

// TestShiftTabCyclesModeBackward verifies shift+tab reverses mode order.
// With four modes (Time, Words, Quote, Code), shift+tab from Time wraps to Code.
func TestShiftTabCyclesModeBackward(t *testing.T) {
	h := newTestHome()
	// Start on Time, shift+tab should wrap to Code (last mode).
	h, _ = h.Update(pressShiftTab())
	if h.currentMode() != config.ModeCode {
		t.Fatalf("shift+tab from Time: want ModeCode (last), got %v", h.currentMode())
	}

	h, _ = h.Update(pressShiftTab())
	if h.currentMode() != config.ModeQuote {
		t.Fatalf("shift+tab from Code: want ModeQuote, got %v", h.currentMode())
	}

	h, _ = h.Update(pressShiftTab())
	if h.currentMode() != config.ModeWords {
		t.Fatalf("shift+tab from Quote: want ModeWords, got %v", h.currentMode())
	}
}

// TestRightArrowChangesLength verifies → increments the length option index.
func TestRightArrowChangesLength(t *testing.T) {
	h := newTestHome()
	initial := h.lenIdx[config.ModeTime] // 1 (30s)
	h, _ = h.Update(pressRight())
	if h.lenIdx[config.ModeTime] != initial+1 {
		t.Fatalf("want lenIdx %d, got %d", initial+1, h.lenIdx[config.ModeTime])
	}
}

// TestLeftArrowChangesLength verifies ← decrements the length option index.
func TestLeftArrowChangesLength(t *testing.T) {
	h := newTestHome()
	initial := h.lenIdx[config.ModeTime] // 1
	h, _ = h.Update(pressLeft())
	if h.lenIdx[config.ModeTime] != initial-1 {
		t.Fatalf("want lenIdx %d, got %d", initial-1, h.lenIdx[config.ModeTime])
	}
}

// TestLengthClampedAtMin verifies ← does not go below 0.
func TestLengthClampedAtMin(t *testing.T) {
	h := newTestHome()
	// Move to index 0 first.
	for i := 0; i < 5; i++ {
		h, _ = h.Update(pressLeft())
	}
	if h.lenIdx[config.ModeTime] != 0 {
		t.Fatalf("want clamped to 0, got %d", h.lenIdx[config.ModeTime])
	}
}

// TestLengthClampedAtMax verifies → does not exceed the last option.
func TestLengthClampedAtMax(t *testing.T) {
	h := newTestHome()
	lens := config.LengthsFor(config.ModeTime) // [15,30,60,120]
	max := len(lens) - 1
	for i := 0; i < 10; i++ {
		h, _ = h.Update(pressRight())
	}
	if h.lenIdx[config.ModeTime] != max {
		t.Fatalf("want clamped to %d, got %d", max, h.lenIdx[config.ModeTime])
	}
}

// TestSwitchModePreservesPerModeLengthIndex verifies per-mode selection is
// preserved across tab switches (no cross-mode corruption).
func TestSwitchModePreservesPerModeLengthIndex(t *testing.T) {
	h := newTestHome()

	// Move Time to index 3 (120s) — start at 1, need 2 more presses.
	h, _ = h.Update(pressRight()) // → 2
	h, _ = h.Update(pressRight()) // → 3
	if h.lenIdx[config.ModeTime] != 3 {
		t.Fatalf("pre-switch Time index: want 3, got %d", h.lenIdx[config.ModeTime])
	}

	// Switch to Words (initial lenIdx depends on default seed; just record it
	// then move one step right and verify the change).
	h, _ = h.Update(pressTab())
	wordsInitial := h.lenIdx[config.ModeWords]
	h, _ = h.Update(pressRight())
	wantWords := wordsInitial + 1
	if h.lenIdx[config.ModeWords] != wantWords {
		t.Fatalf("Words index: want %d, got %d", wantWords, h.lenIdx[config.ModeWords])
	}

	// Switch back to Time — its index must still be 3.
	h, _ = h.Update(pressShiftTab())
	if h.lenIdx[config.ModeTime] != 3 {
		t.Fatalf("post-switch Time index: want 3, got %d", h.lenIdx[config.ModeTime])
	}
}

// TestLengthIndexInRangeAfterModeSwitch verifies the per-mode index is always
// within the bounds of the new mode's options (no out-of-range panic).
// ModeCode is explicitly excluded: it has no length cycler (optionCount==0)
// so the usual [0,count) check does not apply.
func TestLengthIndexInRangeAfterModeSwitch(t *testing.T) {
	h := newTestHome()
	// Cycle through all modes and check each index is valid.
	for i := 0; i < len(modeOrder)*2; i++ {
		h, _ = h.Update(pressTab())
		mode := h.currentMode()
		if mode == config.ModeCode {
			continue // Code has no length cycler; skip range check
		}
		idx := h.lenIdx[mode]
		count := h.optionCount()
		if idx < 0 || idx >= count {
			t.Fatalf("mode %v: index %d out of range [0,%d)", mode, idx, count)
		}
	}
}

// TestEnterEmitsStartTestMsg verifies that pressing enter returns a Cmd that
// produces a StartTestMsg with the correct mode and length.
func TestEnterEmitsStartTestMsg(t *testing.T) {
	h := newTestHome()
	// Defaults: ModeTime, lenIdx=1 → length=30.
	_, cmd := h.Update(pressEnter())
	if cmd == nil {
		t.Fatal("enter should return a non-nil Cmd")
	}
	msg := cmd()
	sm, ok := msg.(StartTestMsg)
	if !ok {
		t.Fatalf("want StartTestMsg, got %T", msg)
	}
	if sm.Mode != config.ModeTime {
		t.Errorf("want Mode=ModeTime, got %v", sm.Mode)
	}
	if sm.Length != 30 {
		t.Errorf("want Length=30, got %d", sm.Length)
	}
}

// TestEnterEmitsStartTestMsgForWords verifies StartTestMsg for Words mode.
func TestEnterEmitsStartTestMsgForWords(t *testing.T) {
	h := newTestHome()
	h, _ = h.Update(pressTab()) // → Words
	_, cmd := h.Update(pressEnter())
	if cmd == nil {
		t.Fatal("enter should return a non-nil Cmd")
	}
	sm, ok := cmd().(StartTestMsg)
	if !ok {
		t.Fatalf("want StartTestMsg, got %T", cmd())
	}
	if sm.Mode != config.ModeWords {
		t.Errorf("want Mode=ModeWords, got %v", sm.Mode)
	}
	lens := config.LengthsFor(config.ModeWords) // [10,25,50,100]
	wantLen := lens[h.lenIdx[config.ModeWords]]
	if sm.Length != wantLen {
		t.Errorf("want Length=%d, got %d", wantLen, sm.Length)
	}
}

// TestEnterEmitsStartTestMsgForQuote verifies StartTestMsg for Quote mode
// carries the correct QuoteLen bucket.
func TestEnterEmitsStartTestMsgForQuote(t *testing.T) {
	h := newTestHome()
	h, _ = h.Update(pressTab()) // → Words
	h, _ = h.Update(pressTab()) // → Quote
	// Default bucket index is 1 → QuoteMedium.
	_, cmd := h.Update(pressEnter())
	if cmd == nil {
		t.Fatal("enter should return a non-nil Cmd")
	}
	sm, ok := cmd().(StartTestMsg)
	if !ok {
		t.Fatalf("want StartTestMsg, got %T", cmd())
	}
	if sm.Mode != config.ModeQuote {
		t.Errorf("want Mode=ModeQuote, got %v", sm.Mode)
	}
	if sm.QuoteLen != words.QuoteMedium {
		t.Errorf("want QuoteLen=QuoteMedium, got %v", sm.QuoteLen)
	}
}

// TestViewContainsLogo verifies View() includes the logo. The view
// contains ANSI escape codes, so we look for box-drawing characters that
// appear in the block-art logo lines (never inside an escape sequence) or for
// the plain "typeburn" text used in the narrow fallback.
func TestViewContainsLogo(t *testing.T) {
	h := newTestHome()
	h = h.SetSize(80, 24)
	v := h.View()
	// Box-drawing ╗ appears in every wide logo line; plain text used narrow.
	hasWide := strings.Contains(v, "╗")
	hasNarrow := strings.Contains(v, "typeburn")
	if !hasWide && !hasNarrow {
		t.Fatalf("view should contain logo (wide: '╗', narrow: 'typeburn'), got:\n%s", v)
	}
}

// TestViewContainsModeLabel verifies the current mode label appears in View().
// ANSI codes wrap styled segments but the text bytes for "Time", "Words",
// "Quote" are present as literal UTF-8 within the ANSI sequences.
func TestViewContainsModeLabel(t *testing.T) {
	h := newTestHome()
	h = h.SetSize(80, 24)
	v := h.View()
	// All three mode labels must be present; they are plain ASCII embedded in
	// ANSI escape sequences (the escape codes only surround, not replace them).
	for _, label := range modeLabels {
		if !strings.Contains(v, label) {
			t.Fatalf("view should contain mode label %q, got:\n%s", label, v)
		}
	}
}

// TestNarrowLogoFallback verifies RenderLogo returns a single line under 64 cols.
func TestNarrowLogoFallback(t *testing.T) {
	th := theme.Default()
	logo := RenderLogo(60, th)
	if strings.Contains(logo, "\n") {
		t.Fatalf("narrow logo (<64) should be single line, got multi-line:\n%s", logo)
	}
	if !strings.Contains(logo, "typeburn") {
		t.Fatalf("narrow logo should contain 'typeburn', got: %q", logo)
	}
}

// TestWideLogo verifies RenderLogo returns multi-line art at wide widths.
func TestWideLogo(t *testing.T) {
	th := theme.Default()
	logo := RenderLogo(80, th)
	if !strings.Contains(logo, "\n") {
		t.Fatalf("wide logo (>=64) should be multi-line block art")
	}
}

// TestSpaceAlsoStarts verifies space key emits StartTestMsg same as enter.
func TestSpaceAlsoStarts(t *testing.T) {
	h := newTestHome()
	_, cmd := h.Update(pressKey(tea.KeySpace, 0))
	if cmd == nil {
		t.Fatal("space should return a non-nil Cmd")
	}
	if _, ok := cmd().(StartTestMsg); !ok {
		t.Fatalf("space should emit StartTestMsg, got %T", cmd())
	}
}

// TestVimKeysChangeLengthOption verifies 'h'/'l' work as left/right.
func TestVimKeysChangeLengthOption(t *testing.T) {
	h := newTestHome()
	initial := h.lenIdx[config.ModeTime]

	h, _ = h.Update(pressKey('l', 0)) // vim right
	if h.lenIdx[config.ModeTime] != initial+1 {
		t.Fatalf("'l' should increment: want %d, got %d", initial+1, h.lenIdx[config.ModeTime])
	}

	h, _ = h.Update(pressKey('h', 0)) // vim left
	if h.lenIdx[config.ModeTime] != initial {
		t.Fatalf("'h' should decrement: want %d, got %d", initial, h.lenIdx[config.ModeTime])
	}
}
