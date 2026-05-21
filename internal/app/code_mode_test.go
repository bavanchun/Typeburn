package app

// code_mode_test.go covers the wiring for ModeCode through the root model:
// StartTestMsg{Mode:ModeCode} → ScreenTyping with code target, key regression,
// and result persistence with IsNewBest==false.

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// codeTestModel returns a sandboxed root model with codeText wired in.
func codeTestModel(t *testing.T, codeText, codeHint string) Model {
	t.Helper()
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	return New(theme.Default(), config.Defaults(), codeText, codeHint, nil)
}

// TestCodeMode_StartTestMsgRoutesToTyping verifies StartTestMsg{Mode:ModeCode}
// transitions root model to ScreenTyping.
func TestCodeMode_StartTestMsgRoutesToTyping(t *testing.T) {
	m := tea.Model(codeTestModel(t, "hello world", ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m2, _ := m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: "hello world"})
	rm := m2.(Model)
	if rm.screen != ScreenTyping {
		t.Fatalf("after StartTestMsg(ModeCode): want ScreenTyping, got %v", rm.screen)
	}
}

// TestCodeMode_TypingTargetMatchesCodeText verifies the typing model's target
// is exactly the CodeText from StartTestMsg (not words.ForMode output).
func TestCodeMode_TypingTargetMatchesCodeText(t *testing.T) {
	codeText := "func main() {\n\treturn\n}"
	m := tea.Model(codeTestModel(t, codeText, ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m2, _ := m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: codeText})
	rm := m2.(Model)
	got := rm.typing.TargetText()
	if got != codeText {
		t.Fatalf("typing target: want %q, got %q", codeText, got)
	}
}

// TestCodeMode_TabDuringTest_DeliveredToEngine verifies that Tab during a code
// test is delivered to the typing engine (not consumed as nav). Tab in the
// typing engine is RestartSame — so after pressing Tab the startMs resets to 0.
func TestCodeMode_TabDuringTest_DeliveredToEngine(t *testing.T) {
	codeText := "a\tb" // contains a literal tab
	m := tea.Model(codeTestModel(t, codeText, ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: codeText})

	// Type first char to start the timer.
	m, _ = sm_sendText(m, "a")
	beforeStartMs := m.(Model).typing.ExportedStartMs()
	if beforeStartMs == 0 {
		t.Fatal("startMs should be non-zero after first keystroke")
	}

	// Press Tab — should restart (RestartSame), resetting startMs to 0.
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	afterStartMs := m.(Model).typing.ExportedStartMs()
	if afterStartMs != 0 {
		t.Errorf("Tab during code test should reset startMs (RestartSame), got %d", afterStartMs)
	}
	// Must still be on ScreenTyping.
	if m.(Model).screen != ScreenTyping {
		t.Errorf("Tab must not navigate away from ScreenTyping, got %v", m.(Model).screen)
	}
}

// TestCodeMode_EnterDuringTest_DeliveredToEngine verifies Enter is forwarded
// to the typing engine during a code test. Enter is a printable '\n' rune that
// advances the engine; after one Enter the engine has one more event in its log.
func TestCodeMode_EnterDuringTest_DeliveredToEngine(t *testing.T) {
	// Target starts with a newline so Enter is immediately correct.
	codeText := "\nhello"
	m := tea.Model(codeTestModel(t, codeText, ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: codeText})

	logBefore := len(m.(Model).typing.ExportedLog())

	// Send Enter as a text keystroke (text="\n").
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter, Text: "\n"}))

	logAfter := len(m.(Model).typing.ExportedLog())
	if logAfter <= logBefore {
		t.Errorf("Enter must advance the engine log: before=%d after=%d", logBefore, logAfter)
	}
}

// TestCodeMode_ViewContainsCodeChars verifies the typing view renders chars
// from the code target. ANSI sequences intersperse individual chars so we
// check for each rune individually (they appear as plain bytes inside escapes).
func TestCodeMode_ViewContainsCodeChars(t *testing.T) {
	codeText := "helloworld"
	m := tea.Model(codeTestModel(t, codeText, ""))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: codeText})

	view := sm_view(m)
	// Each character of the target must appear as a plain byte somewhere in the
	// ANSI-decorated view. 'h', 'e', 'l', 'o', 'w', 'r', 'd' all unique enough.
	for _, ch := range []string{"h", "e", "l", "o", "w", "r", "d"} {
		if !strings.Contains(view, ch) {
			t.Fatalf("typing view missing char %q from code target, got:\n%s", ch, view)
		}
	}
}

// TestCodeMode_ResultNotNewBest verifies that completing a code test does NOT
// set isBest on the ResultModel (IsNewBest returns false for code mode).
func TestCodeMode_ResultNotNewBest(t *testing.T) {
	codeText := "hi"
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("XDG_DATA_HOME", tmp)
	m := tea.Model(New(theme.Default(), config.Defaults(), codeText, "", nil))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(ui.StartTestMsg{Mode: config.ModeCode, CodeText: codeText})

	// Type all chars so the engine fires completeCmd.
	var cmd tea.Cmd
	for _, r := range []rune(codeText) {
		m, cmd = sm_sendText(m, string(r))
		if cmd != nil {
			if msg := cmd(); msg != nil {
				m, _ = m.Update(msg)
			}
		}
		if m.(Model).screen == ScreenResult {
			break
		}
	}

	if m.(Model).screen != ScreenResult {
		t.Fatal("code test completion must transition to ScreenResult")
	}
	// Result view must NOT contain the star marker (★ = new best).
	view := sm_view(m)
	if strings.Contains(view, "★") {
		t.Fatalf("code mode result must never show new-best (★), got:\n%s", view)
	}
}
