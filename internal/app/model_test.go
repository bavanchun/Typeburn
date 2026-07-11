package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

func newTestModel() Model {
	return New(theme.Default(), config.Defaults(), "", "", nil)
}

func press(code rune, mod tea.KeyMod) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Code: code, Mod: mod})
}

func TestStartsOnHome(t *testing.T) {
	if m := newTestModel(); m.screen != ScreenHome {
		t.Fatalf("want ScreenHome, got %v", m.screen)
	}
}

func TestWindowSizeStored(t *testing.T) {
	m, _ := newTestModel().Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	rm := m.(Model)
	if rm.w != 100 || rm.h != 40 {
		t.Fatalf("size not stored: got %dx%d", rm.w, rm.h)
	}
}

func TestNavigationKeys(t *testing.T) {
	cases := []struct {
		code rune
		want Screen
	}{
		{'2', ScreenSettings},
		{'3', ScreenHistory},
		{'1', ScreenHome},
	}
	m := tea.Model(newTestModel())
	for _, c := range cases {
		m, _ = m.Update(press(c.code, 0))
		if got := m.(Model).screen; got != c.want {
			t.Fatalf("key %q: want %v, got %v", c.code, c.want, got)
		}
	}
}

func TestEscFromSubScreenReturnsHome(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(press('2', 0)) // → Settings
	m, _ = m.Update(press(tea.KeyEsc, 0))
	if got := m.(Model).screen; got != ScreenHome {
		t.Fatalf("esc should return Home, got %v", got)
	}
}

func TestCtrlCQuits(t *testing.T) {
	_, cmd := newTestModel().Update(press('c', tea.ModCtrl))
	if cmd == nil {
		t.Fatal("ctrl+c should return a command")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatalf("ctrl+c should return tea.Quit, got %T", cmd())
	}
}

// TestUpdateHint_ForwardedToResult checks that an updateHint set on Model is
// forwarded to the ResultModel via handleResultMsg.
func TestUpdateHint_ForwardedToResult(t *testing.T) {
	hint := &update.Result{Latest: "v2.1.0", UpgradeAvailable: true}
	m := New(theme.Default(), config.Defaults(), "", "", hint)

	resultMsg := ui.ResultMsg{
		Result: metrics.Result{NetWPM: 80, Accuracy: 95, DurationMs: 30000},
		Mode:   config.ModeTime,
		Length: 30,
	}
	m2 := m.handleResultMsg(resultMsg)
	if m2.result.UpdateHint() == nil {
		t.Error("updateHint should be forwarded to ResultModel")
	}
}

// TestUpdateHint_NilSafe checks that nil updateHint doesn't cause panics via handleResultMsg.
func TestUpdateHint_NilSafe(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("handleResultMsg panicked with nil updateHint: %v", r)
		}
	}()
	m := New(theme.Default(), config.Defaults(), "", "", nil)
	resultMsg := ui.ResultMsg{
		Result: metrics.Result{NetWPM: 80, Accuracy: 95, DurationMs: 30000},
		Mode:   config.ModeTime,
		Length: 30,
	}
	_ = m.handleResultMsg(resultMsg)
}

func TestViewRendersActiveScreen(t *testing.T) {
	m := tea.Model(newTestModel())
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press('2', 0))
	c := m.(Model).View().Content
	// Settings screen renders the title as "S E T T I N G S" (spaced caps).
	// Check for the characteristic separator row that only the real screen renders.
	if !strings.Contains(c, "S E T T I N G S") && !strings.Contains(c, "Theme") {
		t.Fatalf("view should show Settings screen content, got:\n%s", c)
	}
}
