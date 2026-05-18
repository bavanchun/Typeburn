package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

// pressSettKey builds a KeyPressMsg using a rune constant (e.g. tea.KeyDown).
// Reuses the package-level pressKey helper from screen_home_test.go.
func pressSettKey(code rune) tea.KeyPressMsg {
	return pressKey(code, 0)
}

// newTestSettings builds a SettingsModel with a fresh copy of Defaults.
// If onChange is nil a no-op callback is used.
func newTestSettings(onChange func(config.Settings)) (SettingsModel, *config.Settings) {
	s := config.Defaults()
	if onChange == nil {
		onChange = func(config.Settings) {}
	}
	m := NewSettings(&s, theme.Default(), config.DefaultKeymap(), onChange)
	m = m.SetSize(100, 40)
	return m, &s
}

// TestNewSettingsExactly4Rows verifies the constructor yields exactly 4 rows.
func TestNewSettingsExactly4Rows(t *testing.T) {
	m, _ := newTestSettings(nil)
	if len(m.rows) != 4 {
		t.Fatalf("want 4 rows, got %d", len(m.rows))
	}
}

// TestDownMovesSelection verifies ↓ increments selection.
func TestDownMovesSelection(t *testing.T) {
	m, _ := newTestSettings(nil)
	m, _ = m.Update(pressSettKey(tea.KeyDown))
	if m.sel != 1 {
		t.Fatalf("want sel=1 after ↓, got %d", m.sel)
	}
}

// TestUpDoesNotGoNegative verifies ↑ is clamped at row 0.
func TestUpDoesNotGoNegative(t *testing.T) {
	m, _ := newTestSettings(nil)
	m, _ = m.Update(pressSettKey(tea.KeyUp))
	if m.sel != 0 {
		t.Fatalf("want sel=0 after ↑ from top, got %d", m.sel)
	}
}

// TestDownClampsAtLastRow verifies ↓ is clamped at the last row.
func TestDownClampsAtLastRow(t *testing.T) {
	m, _ := newTestSettings(nil)
	for range 10 {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != 3 {
		t.Fatalf("want sel=3 (clamped), got %d", m.sel)
	}
}

// TestThemeCyclesRightAndWraps verifies → cycles Theme {default→mono→default}.
func TestThemeCyclesRightAndWraps(t *testing.T) {
	m, _ := newTestSettings(nil)
	// sel=0 = Theme; initial = "default"
	if m.rows[rowTheme].values[m.rows[rowTheme].idx] != "default" {
		t.Fatalf("initial theme should be 'default'")
	}

	// → once → "mono"
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != "mono" {
		t.Fatalf("after 1 →: want 'mono', got %q", got)
	}

	// → again → wraps to "default"
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != "default" {
		t.Fatalf("after 2 →: want 'default' (wrap), got %q", got)
	}
}

// TestThemeCyclesLeft verifies ← wraps from first to last.
func TestThemeCyclesLeft(t *testing.T) {
	m, _ := newTestSettings(nil)
	// Default is index 0; ← should wrap to "mono".
	m, _ = m.Update(pressSettKey(tea.KeyLeft))
	if got := m.rows[rowTheme].values[m.rows[rowTheme].idx]; got != "mono" {
		t.Fatalf("← from 'default': want 'mono', got %q", got)
	}
}

// TestCyclingDefaultModeRepairsDefaultLength verifies when mode changes the
// Default length row is rebuilt and index stays in bounds.
func TestCyclingDefaultModeRepairsDefaultLength(t *testing.T) {
	m, _ := newTestSettings(nil)

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
	m, s := newTestSettings(nil)
	// Navigate to Blink cursor row (sel=3).
	for range 3 {
		m, _ = m.Update(pressSettKey(tea.KeyDown))
	}
	if m.sel != rowBlinkCursor {
		t.Fatalf("expected sel=3 (blink row), got %d", m.sel)
	}

	initial := s.BlinkCursor
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	if s.BlinkCursor == initial {
		t.Fatalf("BlinkCursor should have toggled from %v", initial)
	}
}

// TestOnChangeCalledOnValueChange verifies onChange is called for every value change.
func TestOnChangeCalledOnValueChange(t *testing.T) {
	calls := 0
	m, _ := newTestSettings(func(config.Settings) { calls++ })

	m, _ = m.Update(pressSettKey(tea.KeyRight)) // cycle theme
	if calls != 1 {
		t.Fatalf("onChange call count: want 1, got %d", calls)
	}

	m, _ = m.Update(pressSettKey(tea.KeyDown))  // move selection (no value change)
	m, _ = m.Update(pressSettKey(tea.KeyRight)) // cycle default mode
	if calls != 2 {
		t.Fatalf("onChange call count after 2nd change: want 2, got %d", calls)
	}
}

// TestOnChangeReceivesUpdatedSettings verifies onChange gets the mutated settings.
func TestOnChangeReceivesUpdatedSettings(t *testing.T) {
	var received config.Settings
	m, _ := newTestSettings(func(s config.Settings) { received = s })

	// Cycle theme (sel=0 by default).
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	_ = m

	if received.Theme != "mono" {
		t.Fatalf("onChange received wrong theme: want 'mono', got %q", received.Theme)
	}
}

// TestEscReturnsAbortMsg verifies esc emits AbortMsg (return to Home).
func TestEscReturnsAbortMsg(t *testing.T) {
	m, _ := newTestSettings(nil)
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
	m, _ := newTestSettings(nil)
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
	m, _ := newTestSettings(nil)
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

// TestViewContains4RowLabels verifies all 4 row labels appear in View().
func TestViewContains4RowLabels(t *testing.T) {
	m, _ := newTestSettings(nil)
	view := m.View()

	labels := []string{"Theme", "Default mode", "Default length", "Blink cursor"}
	for _, label := range labels {
		if !strings.Contains(view, label) {
			t.Errorf("view missing row label %q", label)
		}
	}
}

// TestSettingsPointerUpdatedOnCycle verifies the settings pointer reflects changes.
func TestSettingsPointerUpdatedOnCycle(t *testing.T) {
	m, s := newTestSettings(nil)

	// Cycle theme (row 0).
	m, _ = m.Update(pressSettKey(tea.KeyRight))
	_ = m
	if s.Theme != "mono" {
		t.Fatalf("settings pointer not updated: want theme='mono', got %q", s.Theme)
	}
}
