package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// The rail is one row per big-digit row, so the hero and the rail always end on
// the same line. Anything else leaves a ragged block beside the digits.
func TestRailRows_MatchTheDigitBlockHeight(t *testing.T) {
	for _, m := range []ResultModel{newTestResult(), newTestResultAt(106)} {
		if got := len(m.railRows()); got != numRows {
			t.Errorf("rail has %d rows, want %d", got, numRows)
		}
	}
}

// With no comparable history the rail says so instead of inventing a zero, and
// promotes this run's own figures. It is the first thing a new user sees.
func TestRailRows_FirstRunState(t *testing.T) {
	rows := newTestResult().railRows()
	joined := strings.Join([]string{rows[0].label, rows[1].label}, " · ")
	if joined != "first run · no history yet" {
		t.Errorf("first-run rail reads %q", joined)
	}
	for _, r := range rows {
		if r.label == "personal best" || r.label == "rank" {
			t.Errorf("first-run rail must not claim a %q it does not have", r.label)
		}
	}
	if rows[3].label != "raw" || rows[4].label != "consistency" {
		t.Errorf("first-run rail should promote raw and consistency, got %q/%q",
			rows[3].label, rows[4].label)
	}
}

// The delta is a glyph plus a magnitude. A bare tinted number is invisible with
// NO_COLOR and nearly invisible in the attribute-only mono theme, so the
// direction must survive stripping every escape.
func TestDeltaValue_DirectionIsAGlyph(t *testing.T) {
	cases := []struct {
		delta float64
		want  string
	}{
		{6, "▲ 6 wpm"},
		{-4, "▼ 4 wpm"},
		{0, "= 0 wpm"},
		{0.2, "= 0 wpm"},
	}
	for _, c := range cases {
		got, _ := deltaValue(c.delta)
		if got != c.want {
			t.Errorf("deltaValue(%.1f) = %q, want %q", c.delta, got, c.want)
		}
	}
}

// A run withheld from history took no place in it. Printing one would be the
// same impossible claim the withholding exists to prevent.
func TestRailRows_UnrankedRunSaysSo(t *testing.T) {
	m := newTestResult().WithContext(sampleContext().Unranked())
	rows := m.railRows()
	if got := rows[len(rows)-1].value; got != "not ranked" {
		t.Errorf("rank value %q, want %q", got, "not ranked")
	}
	if !strings.Contains(stripANSI(m.standing()), "not ranked") {
		t.Errorf("closing row should say the run is not ranked: %q", stripANSI(m.standing()))
	}
}

// renderRail emits exactly the column it was given, on every line, whatever the
// labels are — the band's arithmetic depends on it.
func TestRenderRail_ExactColumnWidth(t *testing.T) {
	th := theme.Load("default", true)
	for _, m := range []ResultModel{newTestResult(), newTestResultAt(106)} {
		rows := m.railRows()
		for _, short := range []bool{false, true} {
			natural := railNaturalW(rows, short)
			for _, colW := range []int{natural, natural + 1, natural + 12} {
				for _, p := range []float64{0, 0.5, 1} {
					for i, line := range renderRail(rows, colW, short, p, th) {
						if got := lipgloss.Width(line); got != colW {
							t.Fatalf("short=%v colW=%d p=%.1f line %d: width %d",
								short, colW, p, i, got)
						}
					}
				}
			}
		}
	}
}

// The rail renders at its own natural width inside whatever column it is given,
// so a label and its value never drift apart on a wide terminal. Stretching the
// rail to a 200-column inner area put them about 50 columns apart, which is the
// measured reason the panel's content width stays capped.
func TestRenderRail_LabelAndValueStayTogether(t *testing.T) {
	const readableSpan = 20
	th := theme.Load("default", true)
	for _, m := range []ResultModel{newTestResult(), newTestResultAt(106)} {
		rows := m.railRows()
		for _, short := range []bool{false, true} {
			if got := railLabelValueSpan(rows, short); got > readableSpan {
				t.Errorf("short=%v: widest label-to-value gap is %d cells, want <= %d",
					short, got, readableSpan)
			}
			// And rendering into a much wider column must not widen the gap: the
			// extra space goes to the left of the block, not inside it.
			wide := renderRail(rows, railNaturalW(rows, short)+40, short, 1, th)
			for _, line := range wide {
				trimmed := strings.TrimRight(stripANSI(line), " ")
				if run := longestSpaceRun(trimmed); run > readableSpan {
					t.Errorf("short=%v: %q has a %d-cell gap inside the rail", short, trimmed, run)
				}
			}
		}
	}
}

// longestSpaceRun returns the longest run of spaces inside s, ignoring the
// leading indent that right-flushes the block.
func longestSpaceRun(s string) int {
	s = strings.TrimLeft(s, " ")
	longest, run := 0, 0
	for _, r := range s {
		if r == ' ' {
			run++
			if run > longest {
				longest = run
			}
			continue
		}
		run = 0
	}
	return longest
}
