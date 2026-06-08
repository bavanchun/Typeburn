package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// TestView_DegradedNotice verifies that a terminal below the 60×20 safe
// minimum renders the degraded-mode notice instead of screen content.
func TestView_DegradedNotice(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 50, Height: 15})
	content := m.(Model).View().Content

	found := strings.Contains(content, "resize") ||
		strings.Contains(content, "60") ||
		strings.Contains(content, "20")
	if !found {
		t.Errorf("degraded notice should mention resize/60/20, got:\n%s", content)
	}
}

// TestView_DegradedNotice_ExactBoundary checks that exactly 60×20 does NOT
// trigger degraded notice (the condition is strictly less-than).
func TestView_DegradedNotice_ExactBoundary(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	content := m.(Model).View().Content

	if strings.Contains(content, "resize") {
		t.Error("60×20 should NOT trigger degraded notice")
	}
}

// TestView_PersistenceNotice verifies that when persistErr is set, the View()
// output contains the error string as a toast overlay.
func TestView_PersistenceNotice(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rm := m.(Model)
	rm.persistErr = "test error"
	content := rm.View().Content

	if !strings.Contains(content, "test error") {
		t.Errorf("persistence notice should contain %q, got:\n%s", "test error", content)
	}
}

// TestView_ScreenTyping_Centered navigates to the typing screen via
// StartTestMsg and verifies View() doesn't panic with a valid window size.
func TestView_ScreenTyping_Centered(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.StartTestMsg{Mode: config.ModeTime, Length: 30})

	rm := m.(Model)
	if rm.screen != ScreenTyping {
		t.Fatalf("expected ScreenTyping, got %v", rm.screen)
	}

	content := rm.View().Content
	if content == "" {
		t.Error("typing screen view should not be empty")
	}
}

// TestView_PlaceholderView verifies placeholderView renders the screen title
// and nav-hint for each screen constant.
func TestView_PlaceholderView(t *testing.T) {
	cases := []struct {
		screen Screen
		title  string
	}{
		{ScreenTyping, "Typing"},
		{ScreenResult, "Result"},
		{ScreenSettings, "Settings"},
		{ScreenHistory, "History"},
		{ScreenHome, "Home"},
		{Screen(255), "Home"}, // unknown defaults to Home
	}
	th := newTestModel().theme
	for _, c := range cases {
		t.Run(c.title, func(t *testing.T) {
			got := placeholderView(c.screen, th)
			if !strings.Contains(got, c.title) {
				t.Errorf("placeholderView(%v) should contain %q", c.screen, c.title)
			}
			if !strings.Contains(got, "ctrl+c") {
				t.Error("placeholderView should contain nav hint")
			}
		})
	}
}

// TestScreenTitle verifies screenTitle returns correct labels.
func TestScreenTitle(t *testing.T) {
	cases := []struct {
		s    Screen
		want string
	}{
		{ScreenTyping, "Typing"},
		{ScreenResult, "Result"},
		{ScreenSettings, "Settings"},
		{ScreenHistory, "History"},
		{ScreenHome, "Home"},
		{Screen(99), "Home"},
	}
	for _, c := range cases {
		if got := screenTitle(c.s); got != c.want {
			t.Errorf("screenTitle(%v) = %q, want %q", c.s, got, c.want)
		}
	}
}

// TestInit_ReturnsNil verifies Init() returns nil cmd.
func TestInit_ReturnsNil(t *testing.T) {
	m := newTestModel()
	if cmd := m.Init(); cmd != nil {
		t.Error("Init() should return nil")
	}
}

// TestView_QuitPromptOverlay verifies the quit-prompt view renders when active.
func TestView_QuitPromptOverlay(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0))
	rm := m.(Model)
	if rm.quitPrompt == nil {
		t.Fatal("quitPrompt should be set after esc on Home")
	}
	content := rm.View().Content
	if content == "" {
		t.Error("quit prompt view should not be empty")
	}
}

// TestView_ZeroSize verifies View() doesn't panic with zero window size.
func TestView_ZeroSize(t *testing.T) {
	m := newTestModel()
	content := m.View().Content
	_ = content // no panic = pass
}

// TestView_ScreenResult verifies View() on the result screen renders.
func TestView_ScreenResult(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	rm := m.(Model)
	msg := ui.ResultMsg{
		Result: metrics.Result{NetWPM: 80, Accuracy: 95, DurationMs: 30000},
		Mode:   config.ModeTime,
		Length: 30,
	}
	rm = rm.handleResultMsg(msg)
	content := rm.View().Content
	if content == "" {
		t.Error("result screen view should not be empty")
	}
}

// TestView_ScreenHistory verifies View() on the history screen renders.
func TestView_ScreenHistory(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press('3', 0))
	content := m.(Model).View().Content
	if content == "" {
		t.Error("history screen view should not be empty")
	}
}
