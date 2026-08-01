package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// TestResult_RestartSame_CodeModeKeepsSnippet is the regression guard for the
// Result-screen restart-same path dropping the Code snippet (which made the
// restarted test instantly complete on an empty target).
func TestResult_RestartSame_CodeModeKeepsSnippet(t *testing.T) {
	const snippet = "func f() {\n\treturn 1\n}"
	msg := ResultMsg{
		Result:   metrics.Result{},
		Mode:     config.ModeCode,
		CodeText: snippet,
	}
	m := NewResult(msg, theme.Default(), config.DefaultKeymap())
	out := m.restartSameCmd()()
	st, ok := out.(StartTestMsg)
	if !ok {
		t.Fatalf("want StartTestMsg, got %T", out)
	}
	if st.Mode != config.ModeCode {
		t.Errorf("mode: want code, got %q", st.Mode)
	}
	if st.CodeText != snippet {
		t.Errorf("restart-same dropped the snippet: got %q want %q", st.CodeText, snippet)
	}
}

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

// TestResultView_ContainsStatsGrid checks the 2-col stats grid: characters
// 3-tuple, test type, and time entries.
func TestResultView_ContainsStatsGrid(t *testing.T) {
	view := stripANSI(newTestResult().View())
	if !strings.Contains(view, "characters") {
		t.Errorf("expected 'characters' label in view:\n%s", view)
	}
	if !strings.Contains(view, "142/4/1") {
		t.Errorf("expected characters tuple '142/4/1' in view:\n%s", view)
	}
	if !strings.Contains(view, "english") {
		t.Errorf("expected 'english' in test-type entry:\n%s", view)
	}
	if !strings.Contains(view, "time 30") {
		t.Errorf("expected 'time 30' in test-type entry:\n%s", view)
	}
	if !strings.Contains(view, "30s") {
		t.Errorf("expected duration '30s' in time entry:\n%s", view)
	}
	// The grid is one aligned column. raw and consistency live in the hero and
	// must not be repeated here — the same number printed twice on one screen
	// is what this replaced.
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, "test type") && strings.Contains(line, "consistency") {
			t.Errorf("consistency must not appear in the stats grid:\n%s", view)
		}
	}
}

// raw and consistency are headline stats in the hero. Printing them again in
// the grid was pure duplication; this pins that they appear exactly once.
func TestResultView_NoDuplicatedStats(t *testing.T) {
	view := stripANSI(newTestResult().View())
	for _, label := range []string{"raw", "consistency"} {
		if n := strings.Count(view, label); n != 1 {
			t.Errorf("%q appears %d times, want exactly 1:\n%s", label, n, view)
		}
	}
}

// TestResultView_ContainsGraph checks the dual-axis graph replaces the old
// bar sparkline: braille line chars plus the wpm-over-time header.
func TestResultView_ContainsGraph(t *testing.T) {
	view := stripANSI(newTestResult().View())
	if !strings.Contains(view, "wpm over time") {
		t.Errorf("expected 'wpm over time' header:\n%s", view)
	}
	found := false
	for _, r := range view {
		if r >= 0x2800 && r <= 0x28FF {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected braille graph line in view:\n%s", view)
	}
}

// TestResultView_ContainsFooter checks that footer hints are rendered.
func TestResultView_ContainsFooter(t *testing.T) {
	view := newTestResult().View()
	if !strings.Contains(view, "tab") {
		t.Errorf("expected 'tab' hint in footer:\n%s", view)
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

// TestResultView_UpdateHint_Absent checks no footer hint line when updateHint is nil.
func TestResultView_UpdateHint_Absent(t *testing.T) {
	m := newTestResult() // updateHint is nil by default
	view := m.View()
	if strings.Contains(view, "available") {
		t.Errorf("expected no update hint in view, got:\n%s", view)
	}
}

// TestResultView_UpdateHint_Present checks the footer hint renders version + command.
func TestResultView_UpdateHint_Present(t *testing.T) {
	hint := &update.Result{
		Current:          "v2.0.0",
		Latest:           "v2.1.0",
		UpgradeAvailable: true,
	}
	m := newTestResult().WithUpdateHint(hint)
	view := m.View()
	if !strings.Contains(view, "v2.1.0") {
		t.Errorf("expected version in hint, view:\n%s", view)
	}
	if !strings.Contains(view, "typeburn update") {
		t.Errorf("expected command in hint, view:\n%s", view)
	}
}

// TestRenderUpdateHint_InjectionGuard checks that non-semver Latest is silently suppressed.
func TestRenderUpdateHint_InjectionGuard(t *testing.T) {
	cases := []struct {
		name   string
		latest string
	}{
		{"ansi escape", "\x1b[31mv2.1.0\x1b[0m"},
		{"shell injection", "v2.1.0; rm -rf /"},
		{"empty string", ""},
		{"plain text", "latest"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hint := &update.Result{Latest: c.latest, UpgradeAvailable: true}
			m := newTestResult().WithUpdateHint(hint)
			got := m.renderUpdateHint()
			if got != "" {
				t.Errorf("expected empty for invalid semver %q, got %q", c.latest, got)
			}
		})
	}
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

// TestResultHero_TwoBigNumbers checks the redesigned hero: WPM big-digit block
// plus a prominent acc block, with raw/consistency as a secondary card row.
func TestResultHero_TwoBigNumbers(t *testing.T) {
	m := newTestResult()
	hero := stripANSI(m.renderHero())
	if !strings.Contains(hero, "97%") {
		t.Errorf("hero missing acc value 97%%:\n%s", hero)
	}
	if !strings.Contains(hero, "acc") {
		t.Errorf("hero missing acc label:\n%s", hero)
	}
	if !strings.Contains(hero, "wpm") {
		t.Errorf("hero missing wpm label:\n%s", hero)
	}
	if !strings.Contains(hero, "raw") || !strings.Contains(hero, "consistency") {
		t.Errorf("hero missing raw/consistency secondary cards:\n%s", hero)
	}
	// Acc value must share a line with big-digit block content (side-by-side),
	// not sit on its own row below the digits.
	sideBySide := false
	for _, line := range strings.Split(hero, "\n") {
		if strings.Contains(line, "97%") && strings.ContainsRune(line, '█') {
			sideBySide = true
			break
		}
	}
	if !sideBySide {
		t.Errorf("acc value should render beside the WPM digits:\n%s", hero)
	}
	// raw + consistency form a secondary row BELOW the big blocks — never
	// beside the ASCII digits (that was the old 3-card side column).
	for i, line := range strings.Split(hero, "\n") {
		if strings.Contains(line, "raw") && strings.ContainsRune(line, '█') {
			t.Errorf("raw card must sit below the digits, found beside them on line %d:\n%s", i, hero)
		}
		if strings.Contains(line, "raw") && !strings.Contains(line, "consistency") {
			t.Errorf("raw and consistency should share the secondary row:\n%s", hero)
		}
	}
}

// TestResultStatsGrid_NeverWrapsInsideColumn guards the 2-col/stacked
// threshold: the longest left line ("test type  words 100 · english") must
// never wrap inside its column block at any panel width.
func TestResultStatsGrid_NeverWrapsInsideColumn(t *testing.T) {
	res := metrics.Result{RawWPM: 108, Consistency: 95, CorrectChars: 142,
		IncorrectChars: 4, ExtraChars: 1, DurationMs: 30000}
	msg := ResultMsg{Result: res, Mode: config.ModeWords, Length: 100}
	m := NewResult(msg, theme.Default(), config.DefaultKeymap())
	for innerW := 36; innerW <= 100; innerW++ {
		grid := stripANSI(m.renderStatsGrid(innerW))
		for _, line := range strings.Split(grid, "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "english") {
				t.Fatalf("innerW=%d: left column wrapped mid-entry:\n%s", innerW, grid)
			}
		}
	}
}
