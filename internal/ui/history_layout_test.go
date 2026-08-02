package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// TestHistoryRow_MatchesTheDeclaredWidth ties historyRowW to the renderer. The
// rule width floors on it, so a column change that nobody reflected in the
// constant would silently let the rule cut into a row.
func TestHistoryRow_MatchesTheDeclaredWidth(t *testing.T) {
	th := theme.Load("default", false)
	rec := harnessRecords(1)[0]

	for _, selected := range []bool{false, true} {
		got := lipgloss.Width(stripANSI(renderHistoryRow(rec, selected, true, th)))
		if got != historyRowW {
			t.Errorf("selected=%v row renders %d cells, historyRowW says %d", selected, got, historyRowW)
		}
	}
}

// TestHistoryRuleW_NeverExceedsTheTerminal: the rule was a constant 62, which
// does not fit the 60-column minimum the product advertises.
func TestHistoryRuleW_NeverExceedsTheTerminal(t *testing.T) {
	for w := 60; w <= 90; w++ {
		if got := historyRuleW(w); got > w {
			t.Errorf("terminal %d: rule is %d cells", w, got)
		}
	}
	if got := historyRuleW(200); got != historyRuleMaxW {
		t.Errorf("a wide terminal should still cap the rule at %d, got %d", historyRuleMaxW, got)
	}
}

// TestHistoryView_FitsAndStaysCentred is the assertion the sparkline defect
// broke. lipgloss.Place sizes the block from its widest line, so an over-wide
// trend row does not spill on its own — it drags the entire screen sideways.
func TestHistoryView_FitsAndStaysCentred(t *testing.T) {
	th := theme.Load("default", false)
	km := config.DefaultKeymap()

	for _, n := range []int{1, 25, 120, 200} {
		for _, w := range []int{60, 61, 72, 80, 88, 120, 200} {
			m := NewHistory(harnessRecords(n), th, km).SetSize(w, 24)
			lines := strings.Split(stripANSI(m.View()), "\n")

			for i, ln := range lines {
				if got := lipgloss.Width(ln); got > w {
					t.Fatalf("%d records at width %d: line %d is %d cells: %q", n, w, i, got, ln)
				}
			}

			// Centring: with the frame placed to the full width, the title has
			// to sit at the middle, not flush left behind a wide neighbour.
			if !centredWithin(lines, w) {
				t.Errorf("%d records at width %d: the screen is not centred", n, w)
			}
		}
	}
}

// centredWithin reports whether the frame's content block is horizontally
// centred in w: the narrowest and widest indents must bracket the middle.
func centredWithin(lines []string, w int) bool {
	widest := 0
	for _, ln := range lines {
		if got := lipgloss.Width(strings.TrimRight(ln, " ")); got > widest {
			widest = got
		}
	}
	if widest == 0 || widest >= w {
		return true
	}
	for _, ln := range lines {
		trimmed := strings.TrimLeft(ln, " ")
		if trimmed == "" {
			continue
		}
		indent := lipgloss.Width(ln) - lipgloss.Width(trimmed)
		if indent == 0 && lipgloss.Width(strings.TrimRight(ln, " ")) < widest {
			return false
		}
	}
	return true
}

// TestHistoryTrend_KeepsItsLabel: the label is the only thing telling the user
// how much history the bars cover. Emitting one bar per record pushed it off
// the end of the line.
func TestHistoryTrend_KeepsItsLabel(t *testing.T) {
	th := theme.Load("default", false)
	km := config.DefaultKeymap()

	view := stripANSI(NewHistory(harnessRecords(200), th, km).SetSize(80, 24).View())

	if !strings.Contains(view, "last 200 tests") {
		t.Errorf("the trend label was cut off:\n%s", view)
	}
}

// TestDownsampleSpark_PreservesShape: compressing 200 records into 40 cells
// must still show the trend, so the bars have to average the records they
// stand for rather than sample or clip them.
func TestDownsampleSpark_PreservesShape(t *testing.T) {
	rising := make([]float64, 200)
	for i := range rising {
		rising[i] = float64(i)
	}

	got := downsampleSpark(rising, 40)

	if len(got) != 40 {
		t.Fatalf("got %d cells, want 40", len(got))
	}
	for i := 1; i < len(got); i++ {
		if got[i] <= got[i-1] {
			t.Fatalf("a strictly rising series stopped rising at cell %d: %v", i, got)
		}
	}
	if got[0] >= rising[len(rising)-1] || got[len(got)-1] <= rising[0] {
		t.Errorf("the compressed series does not span the original range: %v", got)
	}
}

// TestDownsampleSpark_LeavesShortSeriesAlone: a history that already fits must
// be drawn record-for-record.
func TestDownsampleSpark_LeavesShortSeriesAlone(t *testing.T) {
	vals := []float64{1, 2, 3}

	if got := downsampleSpark(vals, 40); len(got) != 3 {
		t.Errorf("got %d cells for a 3-record history, want 3", len(got))
	}
	if got := downsampleSpark(vals, 0); got != nil {
		t.Errorf("no room should mean no bars, got %v", got)
	}
}

// TestHistoryMeta_ReportsTheWindowNotTheCursor: the meta line describes what is
// on screen. Reporting the cursor made fourteen visible rows read "showing 1–1
// of 120".
func TestHistoryMeta_ReportsTheWindowNotTheCursor(t *testing.T) {
	th := theme.Load("default", false)
	km := config.DefaultKeymap()

	m := NewHistory(harnessRecords(120), th, km).SetSize(80, 24)
	view := stripANSI(m.View())

	vis := m.visibleCount()
	want := "showing 1–" + histItoa(vis) + " of 120"
	if !strings.Contains(view, want) {
		t.Errorf("meta line does not report the window %q:\n%s", want, view)
	}
}

// TestHistoryMeta_ClampsToTheRecordCount: a window taller than the history must
// not claim rows that do not exist.
func TestHistoryMeta_ClampsToTheRecordCount(t *testing.T) {
	th := theme.Load("default", false)

	got := stripANSI(renderHistoryMeta(0, 40, 3, th))

	if !strings.Contains(got, "showing 1–3 of 3") {
		t.Errorf("got %q, want the window clamped to 3 records", got)
	}
}

// TestHistoryView_UnsizedStillRenders keeps the pre-WindowSizeMsg path honest:
// with no width to fit, the trend draws every record rather than none.
func TestHistoryView_UnsizedStillRenders(t *testing.T) {
	th := theme.Load("default", false)
	km := config.DefaultKeymap()

	view := stripANSI(NewHistory([]storage.Record{harnessRecords(1)[0]}, th, km).View())

	if !strings.Contains(view, "trend") || !strings.Contains(view, "last 1 tests") {
		t.Errorf("unsized History lost its trend row:\n%s", view)
	}
}
