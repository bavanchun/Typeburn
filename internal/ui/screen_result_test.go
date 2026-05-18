package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/words"
)

// newTestResult constructs a ResultModel with sample data and 80×24 terminal.
func newTestResult() ResultModel {
	res := metrics.Result{
		NetWPM:         94,
		RawWPM:         108,
		Accuracy:       97,
		Consistency:    95,
		CorrectChars:   142,
		IncorrectChars: 4,
		ExtraChars:     1,
		MissedChars:    0,
		Errors:         4,
		DurationMs:     30000,
		PerSecond: []metrics.PerSecond{
			{Sec: 0, RawWPM: 60},
			{Sec: 1, RawWPM: 84},
			{Sec: 2, RawWPM: 96},
			{Sec: 3, RawWPM: 108},
			{Sec: 4, RawWPM: 120},
		},
	}
	msg := ResultMsg{
		Result:   res,
		Mode:     config.ModeTime,
		Length:   30,
		QuoteLen: words.QuoteShort,
	}
	return NewResult(msg, theme.Default(), config.DefaultKeymap()).SetSize(80, 24)
}

// TestNewResult_FieldsPopulated checks constructor sets all fields correctly.
func TestNewResult_FieldsPopulated(t *testing.T) {
	m := newTestResult()
	if m.res.NetWPM != 94 {
		t.Errorf("NetWPM: want 94, got %v", m.res.NetWPM)
	}
	if m.mode != config.ModeTime {
		t.Errorf("mode: want time, got %v", m.mode)
	}
	if m.length != 30 {
		t.Errorf("length: want 30, got %v", m.length)
	}
	if m.isBest {
		t.Error("isBest should default to false")
	}
}

// TestResultView_ContainsPanel checks that View includes rounded border chars.
func TestResultView_ContainsPanel(t *testing.T) {
	view := newTestResult().View()
	// Rounded border uses ╭ and ╰.
	if !strings.Contains(view, "╭") || !strings.Contains(view, "╰") {
		t.Errorf("expected rounded border chars in view:\n%s", view)
	}
}

// TestResultView_ContainsWPMDigitRegion checks that big-digit WPM content is present.
func TestResultView_ContainsWPMDigitRegion(t *testing.T) {
	view := newTestResult().View()
	// BigDigits uses block characters like █
	if !strings.Contains(view, "█") {
		t.Errorf("expected block-art digit chars in view:\n%s", view)
	}
}

// TestResultView_ContainsWPMLabel checks that the "wpm" label is present.
func TestResultView_ContainsWPMLabel(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "wpm") {
		t.Errorf("expected 'wpm' label in view:\n%s", view)
	}
}

// TestResultView_ContainsAccuracy checks that accuracy is rendered.
func TestResultView_ContainsAccuracy(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "97%") {
		t.Errorf("expected '97%%' in view:\n%s", view)
	}
}

// TestResultView_ContainsRaw checks that raw WPM is rendered.
func TestResultView_ContainsRaw(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "108") {
		t.Errorf("expected raw '108' in view:\n%s", view)
	}
}

// TestResultView_ContainsConsistency checks that consistency is rendered.
func TestResultView_ContainsConsistency(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "95%") {
		t.Errorf("expected '95%%' in view:\n%s", view)
	}
}

// TestResultView_ContainsStatsLine checks that char stats appear.
func TestResultView_ContainsStatsLine(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "correct") {
		t.Errorf("expected 'correct' label in view:\n%s", view)
	}
	if !strings.Contains(view, "incorrect") {
		t.Errorf("expected 'incorrect' label in view:\n%s", view)
	}
}

// TestResultView_ContainsMeta checks that meta line (duration · mode · english) appears.
func TestResultView_ContainsMeta(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "english") {
		t.Errorf("expected 'english' in meta line:\n%s", view)
	}
	if !strings.Contains(view, "time 30") {
		t.Errorf("expected 'time 30' in meta line:\n%s", view)
	}
}

// TestResultView_ContainsFooter checks that footer hints are rendered.
func TestResultView_ContainsFooter(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "tab") {
		t.Errorf("expected 'tab' hint in footer:\n%s", view)
	}
}

// TestSparkline_MultiSample checks that sparkline renders for multiple samples.
func TestSparkline_MultiSample(t *testing.T) {
	vals := []float64{60, 80, 100, 90, 110}
	out := Sparkline(vals, 40, 3, theme.Default())
	if out == "" {
		t.Fatal("Sparkline returned empty for multi-sample input")
	}
	// Should contain at least one bar character.
	hasBar := false
	for _, b := range sparkBars {
		if strings.ContainsRune(out, b) {
			hasBar = true
			break
		}
	}
	if !hasBar {
		t.Errorf("Sparkline output contains no bar chars:\n%s", out)
	}
}

// TestSparkline_EmptySample checks that Sparkline handles empty input gracefully.
func TestSparkline_EmptySample(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sparkline panicked on empty input: %v", r)
		}
	}()
	out := Sparkline(nil, 40, 3, theme.Default())
	if out != "" {
		t.Errorf("expected empty string for nil input, got %q", out)
	}
}

// TestSparkline_SingleSample checks that Sparkline handles a single sample without panic.
func TestSparkline_SingleSample(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sparkline panicked on single sample: %v", r)
		}
	}()
	out := Sparkline([]float64{72}, 40, 3, theme.Default())
	if out == "" {
		t.Error("Sparkline returned empty for single sample")
	}
}

// TestSparkline_AllEqualSamples checks no panic and renders when all values equal.
func TestSparkline_AllEqualSamples(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Sparkline panicked on all-equal samples: %v", r)
		}
	}()
	out := Sparkline([]float64{80, 80, 80, 80}, 40, 3, theme.Default())
	if out == "" {
		t.Error("Sparkline returned empty for all-equal samples")
	}
}

// TestResultUpdate_TabEmitsStartTestMsg checks that tab key emits StartTestMsg
// with the same mode, length, and quoteLen as the original test.
func TestResultUpdate_TabEmitsStartTestMsg(t *testing.T) {
	m := newTestResult()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyTab}))
	if cmd == nil {
		t.Fatal("tab should return a cmd")
	}
	msg := cmd()
	sm, ok := msg.(StartTestMsg)
	if !ok {
		t.Fatalf("expected StartTestMsg, got %T", msg)
	}
	if sm.Mode != config.ModeTime {
		t.Errorf("mode: want time, got %v", sm.Mode)
	}
	if sm.Length != 30 {
		t.Errorf("length: want 30, got %v", sm.Length)
	}
}

// TestResultUpdate_EnterEmitsStartTestMsg checks enter key also restarts same test.
func TestResultUpdate_EnterEmitsStartTestMsg(t *testing.T) {
	m := newTestResult()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	if cmd == nil {
		t.Fatal("enter should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(StartTestMsg); !ok {
		t.Fatalf("expected StartTestMsg, got %T", msg)
	}
}

// TestResultUpdate_EscEmitsAbortMsg checks esc navigates to Home via AbortMsg.
func TestResultUpdate_EscEmitsAbortMsg(t *testing.T) {
	m := newTestResult()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd == nil {
		t.Fatal("esc should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(AbortMsg); !ok {
		t.Fatalf("expected AbortMsg, got %T", msg)
	}
}

// TestResultUpdate_CtrlREmitsAbortMsg checks ctrl+r returns to Home for new test selection.
func TestResultUpdate_CtrlREmitsAbortMsg(t *testing.T) {
	m := newTestResult()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: 'r', Mod: tea.ModCtrl}))
	if cmd == nil {
		t.Fatal("ctrl+r should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(AbortMsg); !ok {
		t.Fatalf("expected AbortMsg for ctrl+r, got %T", msg)
	}
}

// TestResultUpdate_3EmitsNavHistoryMsg checks that '3' navigates to History.
func TestResultUpdate_3EmitsNavHistoryMsg(t *testing.T) {
	m := newTestResult()
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: '3'}))
	if cmd == nil {
		t.Fatal("'3' should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(NavHistoryMsg); !ok {
		t.Fatalf("expected NavHistoryMsg, got %T", msg)
	}
}

// TestBigDigits_NonNegative checks BigDigits renders without panic for 0-999.
func TestBigDigits_NonNegative(t *testing.T) {
	th := theme.Default()
	for _, n := range []int{0, 1, 9, 42, 94, 100, 999} {
		out := BigDigits(n, th)
		if out == "" {
			t.Errorf("BigDigits(%d) returned empty", n)
		}
	}
}

// TestBigDigits_Negative checks BigDigits clamps negative to 0.
func TestBigDigits_Negative(t *testing.T) {
	out := BigDigits(-5, theme.Default())
	zero := BigDigits(0, theme.Default())
	if out != zero {
		t.Errorf("BigDigits(-5) should equal BigDigits(0)")
	}
}

// TestResultView_NoZeroSize checks View doesn't panic when w/h are 0.
func TestResultView_NoZeroSize(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked with zero size: %v", r)
		}
	}()
	m := newTestResult()
	m.w, m.h = 0, 0
	_ = m.View()
}

// TestAccColorRole checks the accuracy color thresholds.
func TestAccColorRole(t *testing.T) {
	cases := []struct {
		acc  float64
		want theme.Role
	}{
		{100, theme.RoleSuccess},
		{97, theme.RoleSuccess},
		{96, theme.RoleTextPrimary},
		{90, theme.RoleTextPrimary},
		{89, theme.RoleWarning},
		{0, theme.RoleWarning},
	}
	for _, c := range cases {
		got := accColorRole(c.acc)
		if got != c.want {
			t.Errorf("accColorRole(%.0f): want %v, got %v", c.acc, c.want, got)
		}
	}
}
