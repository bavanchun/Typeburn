package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// baseHistTime is a fixed reference used to build deterministic history records.
var baseHistTime = time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)

// makeHistRecord builds a storage.Record for use in UI tests.
func makeHistRecord(offsetSec int, wpm int) storage.Record {
	return storage.Record{
		Time:        baseHistTime.Add(time.Duration(offsetSec) * time.Second),
		Mode:        "time",
		Length:      30,
		WPM:         wpm,
		RawWPM:      float64(wpm) + 5,
		Accuracy:    97.0,
		Consistency: 90.0,
	}
}

// newTestHistory builds a HistoryModel with the given records and 80×30 terminal.
func newTestHistory(records []storage.Record) HistoryModel {
	return NewHistory(records, theme.Default(), config.DefaultKeymap()).SetSize(80, 30)
}

// TestHistoryModel_EmptyState checks that an empty history renders the friendly
// empty-state message and does not panic.
func TestHistoryModel_EmptyState(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("View panicked on empty history: %v", r)
		}
	}()
	m := newTestHistory(nil)
	view := m.View()
	if !strings.Contains(view, "no tests yet") {
		t.Errorf("expected empty-state message in view:\n%s", view)
	}
}

// TestHistoryModel_TitlePresent checks the H I S T O R Y title is rendered.
func TestHistoryModel_TitlePresent(t *testing.T) {
	m := newTestHistory([]storage.Record{makeHistRecord(0, 88)})
	view := m.View()
	if !strings.Contains(view, "H I S T O R Y") {
		t.Errorf("expected title 'H I S T O R Y' in view:\n%s", view)
	}
}

// TestHistoryModel_TableContainsWPM checks that a known WPM appears in the table.
func TestHistoryModel_TableContainsWPM(t *testing.T) {
	rec := makeHistRecord(0, 94)
	m := newTestHistory([]storage.Record{rec})
	view := m.View()
	if !strings.Contains(view, "94") {
		t.Errorf("expected WPM 94 in table view:\n%s", view)
	}
}

// TestHistoryModel_TableContainsDate checks that the date appears in the table.
func TestHistoryModel_TableContainsDate(t *testing.T) {
	rec := makeHistRecord(0, 80)
	m := newTestHistory([]storage.Record{rec})
	view := m.View()
	if !strings.Contains(view, "2026-05-18") {
		t.Errorf("expected date '2026-05-18' in table view:\n%s", view)
	}
}

// TestHistoryModel_DownMovesSelection checks that the ↓ key advances selection.
func TestHistoryModel_DownMovesSelection(t *testing.T) {
	records := []storage.Record{
		makeHistRecord(0, 80),
		makeHistRecord(1, 85),
		makeHistRecord(2, 90),
	}
	m := newTestHistory(records)
	if m.sel != 0 {
		t.Fatalf("initial sel should be 0, got %d", m.sel)
	}
	m2, _ := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	if m2.sel != 1 {
		t.Errorf("after ↓: want sel=1, got %d", m2.sel)
	}
}

// TestHistoryModel_GJumpsToTop checks that 'g' moves selection to the first row.
func TestHistoryModel_GJumpsToTop(t *testing.T) {
	records := []storage.Record{
		makeHistRecord(0, 80),
		makeHistRecord(1, 85),
		makeHistRecord(2, 90),
	}
	m := newTestHistory(records)
	// Move down first.
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	// 'g' = Top binding (lowercase g, no modifiers).
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'g'}))
	if m.sel != 0 {
		t.Errorf("after g: want sel=0, got %d", m.sel)
	}
	if m.top != 0 {
		t.Errorf("after g: want top=0, got %d", m.top)
	}
}

// TestHistoryModel_ShiftGJumpsToBottom checks that 'G' moves selection to last row.
func TestHistoryModel_ShiftGJumpsToBottom(t *testing.T) {
	records := make([]storage.Record, 5)
	for i := range records {
		records[i] = makeHistRecord(i, 80+i)
	}
	m := newTestHistory(records)
	// Bottom binding is sk('g') = code:'g' + ModShift (per config.DefaultKeymap).
	m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: 'g', Mod: tea.ModShift}))
	if m.sel != len(records)-1 {
		t.Errorf("after G: want sel=%d, got %d", len(records)-1, m.sel)
	}
}

// TestHistoryModel_ScrollWindow checks that the window top adjusts when selection
// exceeds the visible area.
func TestHistoryModel_ScrollWindow(t *testing.T) {
	// Build enough records to exceed visible rows in a small terminal.
	const termH = 22 // small enough that visibleCount is < total records
	const numRecords = 20
	records := make([]storage.Record, numRecords)
	for i := range records {
		records[i] = makeHistRecord(i, 80+i)
	}
	m := NewHistory(records, theme.Default(), config.DefaultKeymap()).SetSize(80, termH)
	vis := m.visibleCount()

	// Press ↓ enough times to push selection past the visible window.
	for i := 0; i < vis+2; i++ {
		m, _ = m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyDown}))
	}
	if m.top == 0 {
		t.Errorf("top should have scrolled past 0 after %d down presses, sel=%d", vis+2, m.sel)
	}
	// Selection should always be visible: top <= sel < top+vis.
	if m.sel < m.top || m.sel >= m.top+vis {
		t.Errorf("sel=%d not in window [%d, %d)", m.sel, m.top, m.top+vis)
	}
}

// TestHistoryModel_BestStarMarked checks that the ★ appears for the per-mode
// best row when there are multiple records.
func TestHistoryModel_BestStarMarked(t *testing.T) {
	records := []storage.Record{
		{Time: baseHistTime, Mode: "time", Length: 30, WPM: 80, Accuracy: 97, Consistency: 90},
		{Time: baseHistTime.Add(time.Second), Mode: "time", Length: 30, WPM: 94, Accuracy: 98, Consistency: 92},
		{Time: baseHistTime.Add(2 * time.Second), Mode: "time", Length: 30, WPM: 75, Accuracy: 95, Consistency: 88},
	}
	m := newTestHistory(records)
	view := m.View()
	if !strings.Contains(view, "★") {
		t.Errorf("expected ★ in view for best row:\n%s", view)
	}
}

// TestHistoryModel_EscEmitsAbortMsg checks that esc navigates to Home.
func TestHistoryModel_EscEmitsAbortMsg(t *testing.T) {
	m := newTestHistory(nil)
	_, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEsc}))
	if cmd == nil {
		t.Fatal("esc should return a cmd")
	}
	if _, ok := cmd().(AbortMsg); !ok {
		t.Fatalf("expected AbortMsg from esc, got %T", cmd())
	}
}

// TestHistoryModel_FooterPresent checks that footer hints are rendered.
func TestHistoryModel_FooterPresent(t *testing.T) {
	m := newTestHistory(nil)
	view := m.View()
	if !strings.Contains(view, "esc") {
		t.Errorf("expected 'esc' hint in footer:\n%s", view)
	}
}

// TestHistoryModel_MetaLinePresent checks that the "showing X-Y of N" meta
// line is rendered when records are present.
func TestHistoryModel_MetaLinePresent(t *testing.T) {
	records := []storage.Record{makeHistRecord(0, 80), makeHistRecord(1, 85)}
	m := newTestHistory(records)
	view := m.View()
	if !strings.Contains(view, "showing") {
		t.Errorf("expected meta 'showing' line in view:\n%s", view)
	}
}

// TestResultModel_WithBest_SetsTrueBadge checks that WithBest(true) causes
// the ★ new best badge to appear in the result view.
func TestResultModel_WithBest_SetsTrueBadge(t *testing.T) {
	msg := ResultMsg{
		Result: makeTestMetricsResult(),
		Mode:   config.ModeTime,
		Length: 30,
	}
	m := NewResult(msg, theme.Default(), config.DefaultKeymap()).
		WithBest(true).SetSize(80, 40)
	view := m.View()
	if !strings.Contains(view, "★") {
		t.Errorf("expected ★ new best badge in view when isBest=true:\n%s", view)
	}
}

// TestResultModel_WithBest_FalseNoBadge checks that WithBest(false) hides badge.
func TestResultModel_WithBest_FalseNoBadge(t *testing.T) {
	msg := ResultMsg{
		Result: makeTestMetricsResult(),
		Mode:   config.ModeTime,
		Length: 30,
	}
	m := NewResult(msg, theme.Default(), config.DefaultKeymap()).
		WithBest(false).SetSize(80, 40)
	view := m.View()
	if strings.Contains(view, "new best") {
		t.Errorf("unexpected 'new best' badge in view when isBest=false:\n%s", view)
	}
}
