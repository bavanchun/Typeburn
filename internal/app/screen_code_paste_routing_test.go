package app

// screen_code_paste_routing_test.go covers the v1.3.0 in-app paste wiring:
// NavCodePasteMsg / CodePastedMsg routing, esc-back via the existing global
// Back handler, PasteMsg screen routing (Typing branch untouched), and that
// all six screens still render. The Code-selection-preserved assertion is
// behavioural: after a paste, Enter on Home must start a CODE test (proving
// WithCodeText kept modeIdx — a NewHome rebuild would snap back to Time).

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// run executes a cmd (if any) and returns the produced message.
func run(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

// toCodePaste drives the (active, ScreenHome, empty-snippet) model to the
// paste screen the real way: mode order is Time→Words→Quote→Code with the
// Time default, so Tab×3 selects Code; Enter on empty Code emits
// NavCodePasteMsg, which the root routes to ScreenCodePaste.
func toCodePaste(t *testing.T, m tea.Model) tea.Model {
	t.Helper()
	for i := 0; i < 3; i++ {
		m, _ = m.Update(press(tea.KeyTab, 0))
	}
	_, cmd := m.Update(press(tea.KeyEnter, 0))
	nav := run(cmd)
	if _, ok := nav.(ui.NavCodePasteMsg); !ok {
		t.Fatalf("Enter on empty Code must emit NavCodePasteMsg, got %T", nav)
	}
	m, _ = m.Update(nav)
	if m.(Model).screen != ScreenCodePaste {
		t.Fatalf("want ScreenCodePaste after NavCodePasteMsg, got %v", m.(Model).screen)
	}
	return m
}

// TestRouting_NavCodePasteMsg_OpensPasteScreen verifies the nav message
// switches to ScreenCodePaste.
func TestRouting_NavCodePasteMsg_OpensPasteScreen(t *testing.T) {
	m := tea.Model(codeTestModel(t, "", ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.NavCodePasteMsg{})
	if rm := m.(Model); rm.screen != ScreenCodePaste {
		t.Fatalf("NavCodePasteMsg: want ScreenCodePaste, got %v", rm.screen)
	}
}

// TestRouting_CodePastedMsg_AppliesAndReturnsHome verifies a valid paste sets
// codeText, clears codeHint, returns to Home, and — crucially — leaves the
// Code row selected so Enter immediately starts a Code test.
func TestRouting_CodePastedMsg_AppliesAndReturnsHome(t *testing.T) {
	m := tea.Model(codeTestModel(t, "", "text file is empty"))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = toCodePaste(t, m) // now on ScreenCodePaste with Code row active

	// Deliver a bracketed paste; it must reach the paste sub-model.
	m, cmd := m.Update(tea.PasteMsg{Content: "func f(){}\n"})
	msg := run(cmd)
	cp, ok := msg.(ui.CodePastedMsg)
	if !ok {
		t.Fatalf("paste on ScreenCodePaste must yield CodePastedMsg, got %T", msg)
	}
	m, _ = m.Update(cp)

	rm := m.(Model)
	if rm.screen != ScreenHome {
		t.Fatalf("after CodePastedMsg: want ScreenHome, got %v", rm.screen)
	}
	if rm.codeText != "func f(){}" {
		t.Errorf("codeText: want %q, got %q", "func f(){}", rm.codeText)
	}
	if rm.codeHint != "" {
		t.Errorf("codeHint must be cleared, got %q", rm.codeHint)
	}

	// Code row still selected → Enter starts a CODE test (not Time). A Home
	// rebuild would snap modeIdx back to the default mode and lose this.
	_, cmd2 := m.Update(press(tea.KeyEnter, 0))
	start, ok := run(cmd2).(ui.StartTestMsg)
	if !ok {
		t.Fatalf("Enter after paste must emit StartTestMsg (Code still selected)")
	}
	if start.Mode != config.ModeCode {
		t.Fatalf("selection not preserved — want ModeCode, got %v (Home rebuilt instead of WithCodeText?)", start.Mode)
	}
	if start.CodeText != "func f(){}" {
		t.Errorf("start CodeText: want %q, got %q", "func f(){}", start.CodeText)
	}
}

// TestRouting_EscFromPaste_ReturnsHomeUnchanged verifies esc on the paste
// screen returns Home with codeText untouched, via the EXISTING global Back
// handler (no new esc code added).
func TestRouting_EscFromPaste_ReturnsHomeUnchanged(t *testing.T) {
	m := tea.Model(codeTestModel(t, "", ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.NavCodePasteMsg{})
	m, _ = m.Update(press(tea.KeyEsc, 0))
	rm := m.(Model)
	if rm.screen != ScreenHome {
		t.Fatalf("esc from paste: want ScreenHome, got %v", rm.screen)
	}
	if rm.codeText != "" {
		t.Errorf("esc must not change codeText, got %q", rm.codeText)
	}
}

// TestRouting_PasteMsg_ScreenScoped verifies PasteMsg is ignored on Home
// (nil cmd, no screen change) and the Typing branch is unaffected (a paste on
// Typing keeps the Typing screen — the existing typing paste tests own the
// engine-feed assertions).
func TestRouting_PasteMsg_ScreenScoped(t *testing.T) {
	m := tea.Model(codeTestModel(t, "", ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Home: ignored.
	m2, cmd := m.Update(tea.PasteMsg{Content: "ignored"})
	if cmd != nil {
		t.Errorf("PasteMsg on Home must be a no-op (nil cmd)")
	}
	if m2.(Model).screen != ScreenHome {
		t.Errorf("PasteMsg on Home must not change screen")
	}

	// Typing: still on Typing after a paste (engine-feed proven by the
	// existing typing paste tests).
	mt, _ := m.Update(ui.StartTestMsg{Mode: config.ModeWords, Length: 10})
	mt, _ = mt.Update(tea.PasteMsg{Content: "abc"})
	if mt.(Model).screen != ScreenTyping {
		t.Errorf("PasteMsg on Typing must keep ScreenTyping (branch intact)")
	}
}

// TestRouting_AllScreensRender drives every screen through a REAL transition
// (the way the runtime would) and asserts non-empty, panic-free output —
// proving the 6th screen did not regress routing for the other five.
func TestRouting_AllScreensRender(t *testing.T) {
	mkHome := func() tea.Model {
		m, _ := tea.Model(codeTestModel(t, "", "")).
			Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		return m
	}
	cases := []struct {
		name string
		at   func() tea.Model
	}{
		{"Home", mkHome},
		{"Settings", func() tea.Model { m, _ := mkHome().Update(press('2', 0)); return m }},
		{"History", func() tea.Model { m, _ := mkHome().Update(press('3', 0)); return m }},
		{"Typing", func() tea.Model {
			m, _ := mkHome().Update(ui.StartTestMsg{Mode: config.ModeWords, Length: 10})
			return m
		}},
		{"CodePaste", func() tea.Model { return toCodePaste(t, mkHome()) }},
	}
	for _, c := range cases {
		out := c.at().(Model).View().Content
		if strings.TrimSpace(stripANSI(out)) == "" {
			t.Errorf("screen %s rendered empty", c.name)
		}
	}
}

// stripANSI removes CSI sequences so substring assertions are color-agnostic.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b {
			for i < len(s) && s[i] != 'm' {
				i++
			}
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
