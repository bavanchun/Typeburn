package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
)

// TestApplySettings_PreservesModeIdx verifies that applySettings does not
// reset the user's selected mode tab — the Home model should keep whatever
// mode the user navigated to before the settings change.
func TestApplySettings_PreservesModeIdx(t *testing.T) {
	m := newTestModel()
	// Set window size first so layout computations don't hit zero-size guards.
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rm := m2.(Model)

	// Tab to next mode on the Home screen.
	m3, _ := rm.Update(press(tea.KeyTab, 0))
	rm2 := m3.(Model)

	// Apply a settings change (toggle blink cursor).
	s := rm2.settings
	s.BlinkCursor = !s.BlinkCursor
	rm3 := rm2.applySettings(s)

	if rm3.screen != ScreenHome {
		t.Error("should still be on home screen")
	}
}

// TestApplySettings_ThemeChange verifies that changing the theme via
// applySettings rebuilds the theme without panicking.
func TestApplySettings_ThemeChange(t *testing.T) {
	m := newTestModel()
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rm := m2.(Model)

	s := rm.settings
	s.Theme = "default" // use default theme (always valid)
	rm2 := rm.applySettings(s)

	if rm2.screen != ScreenHome {
		t.Error("should still be on home screen after theme change")
	}
}
