package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// ── Degraded-mode gate ──────────────────────────────────────────────────────

// TestDegraded_BelowWidthThreshold verifies the degraded notice is rendered
// when the terminal is narrower than 60 columns.
func TestDegraded_BelowWidthThreshold(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 59, Height: 30})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("59×30 should show degraded notice, got:\n%s", view)
	}
	if !strings.Contains(view, "59×30") {
		t.Fatalf("degraded notice should show current size 59×30, got:\n%s", view)
	}
}

// TestDegraded_BelowHeightThreshold verifies the degraded notice is rendered
// when the terminal is shorter than 20 rows.
func TestDegraded_BelowHeightThreshold(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 80, Height: 19})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("80×19 should show degraded notice, got:\n%s", view)
	}
}

// TestDegraded_BothDimensions verifies degraded mode at 59×19.
func TestDegraded_BothDimensions(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 59, Height: 19})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("59×19 should show degraded notice")
	}
	if !strings.Contains(view, "59×19") {
		t.Fatalf("degraded notice should show actual 59×19, got:\n%s", view)
	}
}

// TestDegraded_NotAtSafeMinimum verifies that exactly 60×20 does NOT trigger
// degraded mode — the threshold is strictly <60 or <20.
func TestDegraded_NotAtSafeMinimum(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 60, Height: 20})
	view := m.(Model).View().Content
	if strings.Contains(view, "Terminal too small") {
		t.Fatalf("60×20 must NOT show degraded notice (is safe minimum)")
	}
}

// TestDegraded_ContainsRequiredLines checks the exact copy required by §4.3.
func TestDegraded_ContainsRequiredLines(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 59, Height: 18})
	view := m.(Model).View().Content
	checks := []string{
		"Terminal too small",
		"Need at least 60×20",
		"Resize to continue",
		"ctrl+c quit",
	}
	for _, s := range checks {
		if !strings.Contains(view, s) {
			t.Errorf("degraded notice missing %q, full view:\n%s", s, view)
		}
	}
}

// TestDegraded_CtrlCQuits verifies ctrl+c still hard-quits while in degraded mode.
func TestDegraded_CtrlCQuits(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 10})
	_, cmd := m.Update(press('c', tea.ModCtrl))
	if cmd == nil {
		t.Fatal("ctrl+c in degraded mode should return a Cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c in degraded mode should produce QuitMsg, got %T", cmd())
	}
}

// TestDegraded_RecoverOnResize verifies the notice clears when resized back
// above the threshold.
func TestDegraded_RecoverOnResize(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 40, Height: 10}) // degraded
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24}) // recover
	view := m.(Model).View().Content
	if strings.Contains(view, "Terminal too small") {
		t.Fatalf("after resize to 80×24 the degraded notice must disappear")
	}
}

// ── Esc-on-Home quit prompt ─────────────────────────────────────────────────

// TestEscOnHome_ShowsQuitPrompt verifies that esc on the Home screen shows
// the quit-confirmation overlay (not immediate quit).
func TestEscOnHome_ShowsQuitPrompt(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m2, cmd := m.Update(press(tea.KeyEsc, 0))
	if cmd != nil {
		// Should not have emitted a quit command.
		if msg := cmd(); msg != nil {
			if _, ok := msg.(tea.QuitMsg); ok {
				t.Fatal("esc on Home must not quit immediately — should show prompt")
			}
		}
	}
	// quitPrompt must be set on the returned model.
	rm := m2.(Model)
	if rm.quitPrompt == nil {
		t.Fatal("esc on Home must set quitPrompt, got nil")
	}
}

// TestEscOnHome_YConfirmsQuit verifies 'y' inside the quit-prompt quits.
func TestEscOnHome_YConfirmsQuit(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0)) // open prompt
	_, cmd := m.Update(press('y', 0))
	if cmd == nil {
		t.Fatal("'y' in quit prompt should return a Cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("'y' in quit prompt should produce QuitMsg, got %T", cmd())
	}
}

// TestEscOnHome_NDissmissesPrompt verifies 'n' dismisses without quitting.
func TestEscOnHome_NDismissesPrompt(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0)) // open prompt
	m, cmd := m.Update(press('n', 0))
	if cmd != nil {
		if msg := cmd(); msg != nil {
			if _, ok := msg.(tea.QuitMsg); ok {
				t.Fatal("'n' in quit prompt must NOT quit")
			}
		}
	}
	if rm := m.(Model); rm.quitPrompt != nil {
		t.Fatal("'n' must dismiss the quit prompt (quitPrompt should be nil)")
	}
}

// TestEscOnHome_SecondEscDissmissesPrompt verifies esc inside the prompt dismisses.
func TestEscOnHome_SecondEscDismissesPrompt(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0)) // open prompt
	m, _ = m.Update(press(tea.KeyEsc, 0)) // dismiss
	if rm := m.(Model); rm.quitPrompt != nil {
		t.Fatal("second esc must dismiss the quit prompt")
	}
}

// TestEscOnHome_EnterOnYesQuits verifies enter with "yes" selected quits.
func TestEscOnHome_EnterOnYesQuits(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0))      // open prompt (sel=1 = no)
	m, _ = m.Update(press(tea.KeyLeft, 0))     // move to yes (sel=0)
	_, cmd := m.Update(press(tea.KeyEnter, 0)) // confirm
	if cmd == nil {
		t.Fatal("enter on yes should return Cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("enter on yes should produce QuitMsg, got %T", cmd())
	}
}

// TestCtrlC_HardQuitsInPrompt verifies ctrl+c bypasses the quit-prompt
// and quits immediately.
func TestCtrlC_HardQuitsInPrompt(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0)) // open prompt
	_, cmd := m.Update(press('c', tea.ModCtrl))
	if cmd == nil {
		t.Fatal("ctrl+c must return Cmd even inside quit prompt")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c in prompt must produce QuitMsg, got %T", cmd())
	}
}

// TestQuitPromptView_ContainsOptions verifies the prompt view renders both
// yes and no options.
func TestQuitPromptView_ContainsOptions(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press(tea.KeyEsc, 0))
	view := m.(Model).View().Content
	if !strings.Contains(view, "yes") || !strings.Contains(view, "no") {
		t.Fatalf("quit prompt view must show 'yes' and 'no', got:\n%s", view)
	}
}

// ── NO_COLOR audit ──────────────────────────────────────────────────────────

// TestNoColor_DegradedNotice verifies the degraded notice renders structurally
// correctly under NO_COLOR (attribute-only theme, no hex colors).
func TestNoColor_DegradedNotice(t *testing.T) {
	th := theme.Load("default", true) // noColor=true
	m := New(th, config.Defaults(), "", "", nil)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 15})
	view := m2.(Model).View().Content
	if !strings.Contains(view, "Terminal too small") {
		t.Fatalf("NO_COLOR degraded notice missing, got:\n%s", view)
	}
	// Must not contain raw hex color codes (e.g., #E3B341).
	if strings.Contains(view, "#") {
		t.Fatalf("NO_COLOR view must not contain hex color strings, got:\n%s", view)
	}
}

// TestNoColor_HomeScreen verifies the Home screen renders the accent bar ▎
// (the focus signal that works without color) under NO_COLOR.
func TestNoColor_HomeScreen(t *testing.T) {
	th := theme.Load("default", true) // noColor=true
	m := New(th, config.Defaults(), "", "", nil)
	m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := m2.(Model).View().Content
	// The active tab must have the ▎ marker (works without color per §5.5).
	if !strings.Contains(view, "▎") {
		t.Fatalf("NO_COLOR Home screen must contain ▎ focus marker, got:\n%s", view)
	}
}

// TestNoColor_SettingsScreen verifies selected row still has ▎ under NO_COLOR.
func TestNoColor_SettingsScreen(t *testing.T) {
	th := theme.Load("default", true)
	m := New(th, config.Defaults(), "", "", nil)
	tm, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	tm, _ = tm.Update(press('2', 0)) // → Settings
	view := tm.(Model).View().Content
	if !strings.Contains(view, "▎") {
		t.Fatalf("NO_COLOR Settings must contain ▎ selection marker, got:\n%s", view)
	}
}

// ── Keybinding audit — G (capital) and shift+tab robustness ─────────────────

// TestBottom_CapitalGKey verifies that the Bottom binding (design §8.6 "G")
// matches shift+g as configured in the default keymap.
func TestBottom_CapitalGKey(t *testing.T) {
	km := config.DefaultKeymap()
	// Design §8.6: G = jump to bottom. shift+g chord (uppercase via ModShift).
	shiftG := tea.Key{Code: 'g', Mod: tea.ModShift}
	if !km.Bottom.Matches(shiftG) {
		t.Error("Bottom binding must match shift+g")
	}
}

// TestPrevMode_ShiftTab verifies PrevMode matches shift+tab.
func TestPrevMode_ShiftTab(t *testing.T) {
	km := config.DefaultKeymap()
	shiftTab := tea.Key{Code: tea.KeyTab, Mod: tea.ModShift}
	if !km.PrevMode.Matches(shiftTab) {
		t.Error("PrevMode binding must match shift+tab")
	}
}
