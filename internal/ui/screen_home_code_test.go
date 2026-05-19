package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// newTestHomeCode builds a HomeModel with an empty codeText.
func newTestHomeCode(codeText, codeHint string) HomeModel {
	return NewHome(config.Defaults(), theme.Default(), config.DefaultKeymap(), codeText, codeHint)
}

// TestHome_CodeInModeOrder verifies ModeCode is included in the mode cycle.
func TestHome_CodeInModeOrder(t *testing.T) {
	found := false
	for _, m := range modeOrder {
		if m == config.ModeCode {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("ModeCode must be present in modeOrder")
	}
}

// TestHome_CodeLabelInLabels verifies "Code" appears in modeLabels.
func TestHome_CodeLabelInLabels(t *testing.T) {
	found := false
	for _, l := range modeLabels {
		if l == "Code" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("modeLabels must include 'Code', got %v", modeLabels)
	}
}

// TestHome_TabCycleIncludesCode verifies tab cycling reaches Code mode.
func TestHome_TabCycleIncludesCode(t *testing.T) {
	h := newTestHomeCode("", "")
	seen := false
	for i := 0; i < len(modeOrder)*2; i++ {
		h, _ = h.Update(pressTab())
		if h.currentMode() == config.ModeCode {
			seen = true
			break
		}
	}
	if !seen {
		t.Fatal("tab cycling must reach ModeCode")
	}
}

// TestHome_CodeNoText_EnterOpensPaste verifies Enter on Code with no snippet
// emits NavCodePasteMsg (v1.3.0: was a no-op in v1.2.0 — intended change).
func TestHome_CodeNoText_EnterOpensPaste(t *testing.T) {
	h := newTestHomeCode("", "text file is empty")
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	_, cmd := h.Update(pressEnter())
	if cmd == nil {
		t.Fatal("enter on empty Code must emit NavCodePasteMsg, got nil cmd")
	}
	if _, ok := cmd().(NavCodePasteMsg); !ok {
		t.Fatalf("want NavCodePasteMsg, got %T", cmd())
	}
}

// TestHome_CodeNoText_SpaceOpensPaste verifies space (the Start binding is
// enter/space) also opens paste on empty Code.
func TestHome_CodeNoText_SpaceOpensPaste(t *testing.T) {
	h := newTestHomeCode("", "")
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	_, cmd := h.Update(pressKey(' ', 0))
	if cmd == nil {
		t.Fatal("space on empty Code must emit NavCodePasteMsg, got nil cmd")
	}
	if _, ok := cmd().(NavCodePasteMsg); !ok {
		t.Fatalf("want NavCodePasteMsg, got %T", cmd())
	}
}

// TestHome_WithCodeText_PreservesSelection locks the invariant: WithCodeText
// replaces ONLY codeText/codeHint and preserves modeIdx/lenIdx (white-box —
// same package). A NewHome rebuild would reset modeIdx to DefaultMode.
func TestHome_WithCodeText_PreservesSelection(t *testing.T) {
	h := newTestHomeCode("", "")
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	h, _ = h.Update(pressKey(' ', 0)) // no-op on selection; just exercise state
	want := h.modeIdx
	got := h.WithCodeText("snippet", "")
	if got.codeText != "snippet" {
		t.Errorf("codeText: want %q, got %q", "snippet", got.codeText)
	}
	if got.codeHint != "" {
		t.Errorf("codeHint must be empty, got %q", got.codeHint)
	}
	if got.modeIdx != want {
		t.Errorf("modeIdx not preserved: want %d, got %d", want, got.modeIdx)
	}
	for mode, idx := range h.lenIdx {
		if got.lenIdx[mode] != idx {
			t.Errorf("lenIdx[%v] not preserved: want %d, got %d", mode, idx, got.lenIdx[mode])
		}
	}
}

// TestHome_CodeWithText_StartCmdEmitsMsg verifies Enter emits StartTestMsg when
// code text is loaded.
func TestHome_CodeWithText_StartCmdEmitsMsg(t *testing.T) {
	codeText := "func main() {\n\treturn\n}"
	h := newTestHomeCode(codeText, "")
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	_, cmd := h.Update(pressEnter())
	if cmd == nil {
		t.Fatal("enter with code text must return non-nil cmd")
	}
	msg := cmd()
	sm, ok := msg.(StartTestMsg)
	if !ok {
		t.Fatalf("want StartTestMsg, got %T", msg)
	}
	if sm.Mode != config.ModeCode {
		t.Errorf("want Mode=ModeCode, got %v", sm.Mode)
	}
	if sm.CodeText != codeText {
		t.Errorf("want CodeText=%q, got %q", codeText, sm.CodeText)
	}
}

// TestHome_CodeOptLeftRightNoPanic verifies OptLeft/OptRight do not panic on
// the Code row (which has no length cycler — optionCount==0).
func TestHome_CodeOptLeftRightNoPanic(t *testing.T) {
	h := newTestHomeCode("some code", "")
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("OptLeft panicked on Code row: %v", r)
		}
	}()
	h, _ = h.Update(pressLeft())
	h, _ = h.Update(pressRight())
	// No assertion needed — just must not panic or change mode.
	if h.currentMode() != config.ModeCode {
		t.Error("OptLeft/OptRight must not change mode on Code row")
	}
}

// TestHome_CodeRow_HintNoText verifies renderOptions for Code without text
// shows the disabled-style hint string.
func TestHome_CodeRow_HintNoText(t *testing.T) {
	h := newTestHomeCode("", "")
	h = h.SetSize(80, 24)
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	view := h.View()
	if !strings.Contains(view, "press enter to paste") {
		t.Fatalf("Code row without text must invite an in-app paste, got:\n%s", view)
	}
}

// TestHome_CodeRow_ReadyWithText verifies renderOptions for Code with text
// shows the "ready" hint.
func TestHome_CodeRow_ReadyWithText(t *testing.T) {
	h := newTestHomeCode("some code text", "")
	h = h.SetSize(80, 24)
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	view := h.View()
	if !strings.Contains(view, "ready") {
		t.Fatalf("Code row with text must show 'ready' hint, got:\n%s", view)
	}
}

// TestHome_CodeRow_ErrorHint verifies renderOptions shows a custom error hint
// when codeHint is set (load failure).
func TestHome_CodeRow_ErrorHint(t *testing.T) {
	h := newTestHomeCode("", "text file is empty")
	h = h.SetSize(80, 24)
	for h.currentMode() != config.ModeCode {
		h, _ = h.Update(pressTab())
	}
	view := h.View()
	if !strings.Contains(view, "text file is empty") {
		t.Fatalf("Code row with error hint must show it, got:\n%s", view)
	}
}
