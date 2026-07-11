package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// pressSettKey builds a KeyPressMsg using a rune constant (e.g. tea.KeyDown).
// Reuses the package-level pressKey helper from screen_home_test.go.
func pressSettKey(code rune) tea.KeyPressMsg {
	return pressKey(code, 0)
}

// newTestSettings builds a SettingsModel with a fresh copy of Defaults.
func newTestSettings() SettingsModel {
	m := NewSettings(config.Defaults(), theme.Default(), config.DefaultKeymap())
	return m.SetSize(100, 40)
}

// settChangedFrom runs the Cmd returned by Update and returns the emitted
// SettingsChangedMsg (or fails). A nil cmd means no value change was emitted.
func settChangedFrom(t *testing.T, cmd tea.Cmd) SettingsChangedMsg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected a Cmd emitting SettingsChangedMsg, got nil")
	}
	sc, ok := cmd().(SettingsChangedMsg)
	if !ok {
		t.Fatalf("expected SettingsChangedMsg, got %T", cmd())
	}
	return sc
}

// TestNewSettingsExactly7Rows verifies the constructor yields exactly 7 rows
// (5 original + Punctuation + Numbers).
func TestNewSettingsExactly7Rows(t *testing.T) {
	m := newTestSettings()
	if len(m.rows) != 7 {
		t.Fatalf("want 7 rows, got %d", len(m.rows))
	}
}

func TestCodeDefaultRendersSettings(t *testing.T) {
	s := config.Defaults()
	s.DefaultMode = config.ModeCode
	m := NewSettings(s, theme.Default(), config.DefaultKeymap()).SetSize(100, 40)

	if got := m.rows[rowDefaultMode].values[m.rows[rowDefaultMode].idx]; got != "code" {
		t.Fatalf("persisted Code default should render as code, got %q", got)
	}
	if got := m.View(); !strings.Contains(got, "n/a") {
		t.Fatalf("Code default length row should render n/a, got %q", got)
	}

	for range rowDefaultLength {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if m.s.DefaultLength != s.DefaultLength {
		t.Fatalf("Code default should leave length unchanged: want %d, got %d", s.DefaultLength, m.s.DefaultLength)
	}
}

func TestCyclingPersistedCodeDefaultUsesTUIChoices(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  rune
		want config.Mode
	}{
		{name: "right selects time", key: tea.KeyRight, want: config.ModeTime},
		{name: "left selects quote", key: tea.KeyLeft, want: config.ModeQuote},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := config.Defaults()
			s.DefaultMode = config.ModeCode
			m := NewSettings(s, theme.Default(), config.DefaultKeymap())
			m, _ = m.Update(pressSettKey(tea.KeyDown))
			m, cmd := m.Update(pressSettKey(tc.key))

			if m.s.DefaultMode != tc.want {
				t.Fatalf("selected default mode = %q, want %q", m.s.DefaultMode, tc.want)
			}
			if got := m.rows[rowDefaultMode].values; strings.Join(got, ",") != "time,words,quote" {
				t.Fatalf("TUI default-mode choices = %v, want time/words/quote", got)
			}
			if sc := settChangedFrom(t, cmd); sc.Settings.DefaultMode != tc.want {
				t.Fatalf("emitted default mode = %q, want %q", sc.Settings.DefaultMode, tc.want)
			}
		})
	}
}

// TestDownMovesSelection verifies ↓ increments selection.
func TestDownMovesSelection(t *testing.T) {
	m := newTestSettings()
	m, _ = m.Update(pressSettKey(tea.KeyDown))
	if m.sel != 1 {
		t.Fatalf("want sel=1 after ↓, got %d", m.sel)
	}
}

// TestUpDoesNotGoNegative verifies ↑ is clamped at row 0.
func TestUpDoesNotGoNegative(t *testing.T) {
	m := newTestSettings()
	m, _ = m.Update(pressSettKey(tea.KeyUp))
	if m.sel != 0 {
		t.Fatalf("want sel=0 after ↑ from top, got %d", m.sel)
	}
}

// TestDownClampsAtLastRow verifies ↓ is clamped at the last row.
func TestDownClampsAtLastRow(t *testing.T) {
	m := newTestSettings()
	for range 10 {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowNumbers {
		t.Fatalf("want sel=%d (clamped), got %d", rowNumbers, m.sel)
	}
}

// TestThemeCyclesRightAndWraps verifies → advances through every theme in
// theme.Available() order and wraps back to the first. Asserted generically
// so adding/removing a theme pack does not require touching this test.
func TestThemeCyclesRightAndWraps(t *testing.T) {
	m := newTestSettings()
	vals := m.rows[rowTheme].values
	if len(vals) < 2 {
		t.Fatalf("expected ≥2 themes, got %v", vals)
	}
	if vals[m.rows[rowTheme].idx] != vals[0] {
		t.Fatalf("initial theme should be the first entry %q", vals[0])
	}

	// → once → second theme.
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != vals[1] {
		t.Fatalf("after 1 →: want %q, got %q", vals[1], got)
	}

	// → the remaining len-1 times → wraps back to the first.
	for range len(vals) - 1 {
		m, _ = m.Update(pressSettKey(tea.KeyRight))
	}
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != vals[0] {
		t.Fatalf("after a full cycle →: want %q (wrap), got %q", vals[0], got)
	}
}

// TestThemeCyclesLeft verifies ← from the first entry wraps to the last.
func TestThemeCyclesLeft(t *testing.T) {
	m := newTestSettings()
	vals := m.rows[rowTheme].values
	last := vals[len(vals)-1]
	m, _ = m.Update(pressSettKey(tea.KeyLeft))
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != last {
		t.Fatalf("← from first: want %q (wrap to last), got %q", last, got)
	}
}

// TestCyclingDefaultModeRepairsDefaultLength verifies when mode changes the
// Default length row is rebuilt and index stays in bounds.
func TestCyclingDefaultModeRepairsDefaultLength(t *testing.T) {
	m := newTestSettings()

	// Navigate to Default mode row (sel=1).
	m, _ = m.Update(pressSettKey(tea.KeyDown))
	if m.sel != rowDefaultMode {
		t.Fatalf("expected sel=1, got %d", m.sel)
	}

	// Cycle mode to "words".
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	newMode := config.Mode(m.rows[rowDefaultMode].values[m.rows[rowDefaultMode].idx])

	// Verify the length row was rebuilt for the new mode.
	expectedLens := config.LengthsFor(newMode)
	if expectedLens != nil && len(m.rows[rowDefaultLength].values) != len(expectedLens) {
		t.Fatalf("length row not rebuilt: want %d options, got %d",
			len(expectedLens), len(m.rows[rowDefaultLength].values))
	}

	// Length index must be within the new option set.
	if idx := m.rows[rowDefaultLength].idx; idx < 0 || idx >= len(m.rows[rowDefaultLength].values) {
		t.Fatalf("length idx %d out of range for %d options", idx, len(m.rows[rowDefaultLength].values))
	}
}

// TestTogglingBlinkFlipsBool verifies cycling the Blink row toggles BlinkCursor.
func TestTogglingBlinkFlipsBool(t *testing.T) {
	m := newTestSettings()
	// Navigate to Blink cursor row (sel=3).
	for range 3 {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowBlinkCursor {
		t.Fatalf("expected sel=3 (blink row), got %d", m.sel)
	}

	initial := m.s.BlinkCursor
	m, cmd := m.Update(pressSettKey(tea.KeyRight))
	if m.s.BlinkCursor == initial {
		t.Fatalf("BlinkCursor should have toggled from %v", initial)
	}
	if sc := settChangedFrom(t, cmd); sc.Settings.BlinkCursor == initial {
		t.Fatalf("emitted msg BlinkCursor should be toggled from %v", initial)
	}
}

// TestTogglingStrictFlipsBool verifies cycling the Strict mode row toggles StrictMode.
func TestTogglingStrictFlipsBool(t *testing.T) {
	m := newTestSettings()
	// Navigate to Strict mode row (sel=4).
	for range 4 {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowStrictMode {
		t.Fatalf("expected sel=4 (strict row), got %d", m.sel)
	}

	initial := m.s.StrictMode
	m, cmd := m.Update(pressSettKey(tea.KeyRight))
	if m.s.StrictMode == initial {
		t.Fatalf("StrictMode should have toggled from %v", initial)
	}
	if sc := settChangedFrom(t, cmd); sc.Settings.StrictMode == initial {
		t.Fatalf("emitted msg StrictMode should be toggled from %v", initial)
	}
}

// TestTogglingPunctuationFlipsBool verifies cycling the Punctuation row
// toggles Punctuation.
func TestTogglingPunctuationFlipsBool(t *testing.T) {
	m := newTestSettings()
	for range rowPunctuation {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowPunctuation {
		t.Fatalf("expected sel=%d (punctuation row), got %d", rowPunctuation, m.sel)
	}

	initial := m.s.Punctuation
	m, cmd := m.Update(pressSettKey(tea.KeyRight))
	if m.s.Punctuation == initial {
		t.Fatalf("Punctuation should have toggled from %v", initial)
	}
	if sc := settChangedFrom(t, cmd); sc.Settings.Punctuation == initial {
		t.Fatalf("emitted msg Punctuation should be toggled from %v", initial)
	}
}

// TestTogglingNumbersFlipsBool verifies cycling the Numbers row toggles Numbers.
func TestTogglingNumbersFlipsBool(t *testing.T) {
	m := newTestSettings()
	for range rowNumbers {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowNumbers {
		t.Fatalf("expected sel=%d (numbers row), got %d", rowNumbers, m.sel)
	}

	initial := m.s.Numbers
	m, cmd := m.Update(pressSettKey(tea.KeyRight))
	if m.s.Numbers == initial {
		t.Fatalf("Numbers should have toggled from %v", initial)
	}
	if sc := settChangedFrom(t, cmd); sc.Settings.Numbers == initial {
		t.Fatalf("emitted msg Numbers should be toggled from %v", initial)
	}
}

// TestValueChangeEmitsSettingsChangedMsg verifies a value change emits a
// SettingsChangedMsg cmd while a pure selection move emits none.
func TestValueChangeEmitsSettingsChangedMsg(t *testing.T) {
	m := newTestSettings()

	m, cmd := m.Update(pressSettKey(tea.KeyRight)) // cycle theme
	settChangedFrom(t, cmd)                        // must emit

	m, cmd = m.Update(pressSettKey(tea.KeyDown)) // move selection only
	if cmd != nil {
		t.Fatalf("selection move must not emit a change cmd, got %T", cmd())
	}

	_, cmd = m.Update(pressSettKey(tea.KeyRight)) // cycle default mode
	settChangedFrom(t, cmd)                       // must emit
}

// TestSettingsChangedMsgCarriesUpdatedSettings verifies the emitted message
// carries the mutated settings value.
func TestSettingsChangedMsgCarriesUpdatedSettings(t *testing.T) {
	m := newTestSettings()

	// Cycle theme (sel=0 by default).
	_, cmd := m.Update(pressSettKey(tea.KeyRight))

	if sc := settChangedFrom(t, cmd); sc.Settings.Theme != "mono" {
		t.Fatalf("emitted msg wrong theme: want 'mono', got %q", sc.Settings.Theme)
	}
}

// TestEscReturnsAbortMsg verifies esc emits AbortMsg (return to Home).
func TestEscReturnsAbortMsg(t *testing.T) {
	m := newTestSettings()
	_, cmd := m.Update(pressSettKey(tea.KeyEsc))
	if cmd == nil {
		t.Fatal("esc: expected a Cmd, got nil")
	}
	if _, ok := cmd().(AbortMsg); !ok {
		t.Fatalf("esc: expected AbortMsg, got %T", cmd())
	}
}

// TestKey1ReturnsAbortMsg verifies '1' (NavHome) also returns to Home.
func TestKey1ReturnsAbortMsg(t *testing.T) {
	m := newTestSettings()
	_, cmd := m.Update(pressKey('1', 0))
	if cmd == nil {
		t.Fatal("key '1': expected a Cmd, got nil")
	}
	if _, ok := cmd().(AbortMsg); !ok {
		t.Fatalf("key '1': expected AbortMsg, got %T", cmd())
	}
}

// TestExcludedOptionsNeverRendered verifies the view has no excluded row labels.
func TestExcludedOptionsNeverRendered(t *testing.T) {
	m := newTestSettings()
	view := m.View()

	excluded := []string{
		"Error mode", "Smooth cursor", "Restart flash", "Sound",
	}
	for _, label := range excluded {
		if strings.Contains(view, label) {
			t.Errorf("view should not contain excluded option %q", label)
		}
	}
}

// TestViewContains5RowLabels verifies all 5 row labels appear in View().
func TestViewContains5RowLabels(t *testing.T) {
	m := newTestSettings()
	view := m.View()

	labels := []string{"Theme", "Default mode", "Default length", "Blink cursor", "Strict mode", "Punctuation", "Numbers"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("view missing row label %q", label)
		}
	}
}

// TestSettingsValueUpdatedOnCycle verifies the in-model settings value
// reflects changes after a cycle.
func TestSettingsValueUpdatedOnCycle(t *testing.T) {
	m := newTestSettings()

	// Cycle theme (row 0).
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if m.s.Theme != "mono" {
		t.Fatalf("settings value not updated: want theme='mono', got %q", m.s.Theme)
	}
}
