package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
)

// emitCodePasted runs the cmd (if any) and returns the CodePastedMsg it
// yields, plus whether one was emitted at all.
func emitCodePasted(cmd tea.Cmd) (CodePastedMsg, bool) {
	if cmd == nil {
		return CodePastedMsg{}, false
	}
	msg := cmd()
	cp, ok := msg.(CodePastedMsg)
	return cp, ok
}

// TestCodePaste_ValidPaste_EmitsNormalized verifies a valid bracketed paste
// is normalized (one trailing newline trimmed) and emitted as CodePastedMsg.
func TestCodePaste_ValidPaste_EmitsNormalized(t *testing.T) {
	m := NewCodePaste(theme.Default())
	m, cmd := m.Update(tea.PasteMsg{Content: "func f(){}\n"})
	cp, ok := emitCodePasted(cmd)
	if !ok {
		t.Fatal("valid paste must emit CodePastedMsg")
	}
	if cp.Text != "func f(){}" {
		t.Errorf("want normalized %q, got %q", "func f(){}", cp.Text)
	}
}

// TestCodePaste_EmptyPaste_ErrorsNoEmit verifies an empty paste does not emit
// and leaves the model showing an empty-input reason.
func TestCodePaste_EmptyPaste_ErrorsNoEmit(t *testing.T) {
	m := NewCodePaste(theme.Default())
	m, cmd := m.Update(tea.PasteMsg{Content: ""})
	if _, ok := emitCodePasted(cmd); ok {
		t.Fatal("empty paste must NOT emit CodePastedMsg")
	}
	v := m.View()
	if !strings.Contains(strings.ToLower(v), "empty") {
		t.Errorf("error view must explain the empty input, got:\n%s", v)
	}
}

// TestCodePaste_OverCap_ErrorsTooLarge verifies an over-cap paste errors with
// a size reason and does not emit.
func TestCodePaste_OverCap_ErrorsTooLarge(t *testing.T) {
	m := NewCodePaste(theme.Default())
	m, cmd := m.Update(tea.PasteMsg{Content: strings.Repeat("x", 10001)})
	if _, ok := emitCodePasted(cmd); ok {
		t.Fatal("over-cap paste must NOT emit CodePastedMsg")
	}
	if !strings.Contains(strings.ToLower(m.View()), "large") {
		t.Errorf("over-cap view must mention the size limit, got:\n%s", m.View())
	}
}

// TestCodePaste_Binary_ErrorsBinary verifies a NUL-containing paste errors
// with a not-text reason.
func TestCodePaste_Binary_ErrorsBinary(t *testing.T) {
	m := NewCodePaste(theme.Default())
	m, cmd := m.Update(tea.PasteMsg{Content: "pkg\x00main"})
	if _, ok := emitCodePasted(cmd); ok {
		t.Fatal("binary paste must NOT emit CodePastedMsg")
	}
	if !strings.Contains(strings.ToLower(m.View()), "text") {
		t.Errorf("binary view must say it is not valid text, got:\n%s", m.View())
	}
}

// TestCodePaste_RecoversAfterError verifies a valid paste after an error
// clears the error state and emits.
func TestCodePaste_RecoversAfterError(t *testing.T) {
	m := NewCodePaste(theme.Default())
	m, _ = m.Update(tea.PasteMsg{Content: ""}) // error state
	m, cmd := m.Update(tea.PasteMsg{Content: "ok\n"})
	cp, ok := emitCodePasted(cmd)
	if !ok {
		t.Fatal("valid paste after an error must emit CodePastedMsg")
	}
	if cp.Text != "ok" {
		t.Errorf("want %q, got %q", "ok", cp.Text)
	}
	if strings.Contains(strings.ToLower(m.View()), "empty") {
		t.Errorf("error reason must clear on recovery, got:\n%s", m.View())
	}
}

// TestCodePaste_NonPasteMsg_NoOp verifies non-paste messages (e.g. a key
// press) are ignored: model unchanged, nil cmd. esc is NOT handled here — the
// global Back handler routes it before this sub-model is reached.
func TestCodePaste_NonPasteMsg_NoOp(t *testing.T) {
	m := NewCodePaste(theme.Default())
	before := m.View()
	m2, cmd := m.Update(pressEnter())
	if cmd != nil {
		t.Error("non-paste msg must return nil cmd")
	}
	if m2.View() != before {
		t.Error("non-paste msg must not change the model")
	}
}

// TestCodePaste_WaitingView_ShowsInstruction verifies the waiting state shows
// the paste instruction and the esc hint.
func TestCodePaste_WaitingView_ShowsInstruction(t *testing.T) {
	m := NewCodePaste(theme.Default()).SetSize(80, 24)
	v := strings.ToLower(m.View())
	if !strings.Contains(v, "paste") || !strings.Contains(v, "esc") {
		t.Errorf("waiting view must invite a paste and mention esc, got:\n%s", m.View())
	}
}

// TestCodePaste_NoColor_SameLineStructure verifies the NO_COLOR theme yields
// the same line count as the colored theme (theme.Role styling only).
func TestCodePaste_NoColor_SameLineStructure(t *testing.T) {
	colored := NewCodePaste(theme.Load("default", false)).SetSize(80, 24)
	plain := NewCodePaste(theme.Load("default", true)).SetSize(80, 24)
	if lc, lp := strings.Count(colored.View(), "\n"), strings.Count(plain.View(), "\n"); lc != lp {
		t.Errorf("NO_COLOR line count %d != colored %d", lp, lc)
	}
}
