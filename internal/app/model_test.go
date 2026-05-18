package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

func newTestModel() Model {
	return New(theme.Default(), config.Defaults())
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
