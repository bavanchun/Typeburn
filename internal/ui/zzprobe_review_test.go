package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// ---- probe 1: lipgloss Width semantics vs resultPanelInset -----------------

func TestProbe_Inset(t *testing.T) {
	for _, panelW := range []int{40, 52, 72, 94} {
		st := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(panelW)
		body := strings.Repeat("A", 200)
		out := st.Render(body)
		lines := strings.Split(out, "\n")
		t.Logf("panelW=%d rendered=%d lines=%d", panelW, lipgloss.Width(lines[0]), len(lines))
		// how many A per content line?
		for _, ln := range lines {
			s := strings.Trim(stripANSI(ln), "│ ")
			if strings.Contains(s, "A") {
				t.Logf("  content run = %d (panelW-content = %d)", len(s), panelW-len(s))
				break
			}
		}
	}
}

// ---- probe 2: panel/graph fit at many widths, with and without errors ------

func errRunResult() ResultMsg {
	per := make([]metrics.PerSecond, 8)
	for i := range per {
		per[i] = metrics.PerSecond{Sec: i, RawWPM: float64(60 + i*5), Errors: i % 3}
	}
	r := shortRunResult()
	r.Result.PerSecond = per
	return r
}

func TestProbe_PanelFitAllWidths(t *testing.T) {
	for w := 40; w <= 220; w++ {
		for _, msg := range []ResultMsg{shortRunResult(), errRunResult()} {
			p := settledPanel(t, msg, w, true)
			maxw := 0
			for _, ln := range strings.Split(p, "\n") {
				if n := lipgloss.Width(ln); n > maxw {
					maxw = n
				}
			}
			if maxw > w {
				t.Errorf("termW=%d: panel width %d exceeds terminal", w, maxw)
			}
		}
	}
}

func TestProbe_GraphFitsInnerW(t *testing.T) {
	th := theme.Load("default", true)
	mk := func(n int, errs bool) []metrics.PerSecond {
		ps := make([]metrics.PerSecond, n)
		for i := range ps {
			ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(40 + i%50)}
			if errs && i%3 == 0 {
				ps[i].Errors = 1 + i%4
			}
		}
		return ps
	}
	for w := 40; w <= 220; w += 1 {
		inner := layoutFor(w).InnerW
		for _, n := range []int{1, 2, 3, 8, 15, 30, 60, 120, 300, 1000} {
			for _, errs := range []bool{false, true} {
				out := stripANSI(RenderResultGraph(mk(n, errs), inner, 5, n, th))
				for i, ln := range strings.Split(out, "\n") {
					if got := len([]rune(ln)); got > inner {
						t.Fatalf("termW=%d inner=%d n=%d errs=%v line %d width %d > inner",
							w, inner, n, errs, i, got)
					}
				}
			}
		}
	}
}

// ---- probe 3: tiny/negative widths, cols==1 -------------------------------

func TestProbe_TinyWidths(t *testing.T) {
	th := theme.Default()
	for _, w := range []int{-5, 0, 1, 4, 5, 6, 9, 10, 12} {
		for _, n := range []int{1, 2, 5, 40} {
			ps := make([]metrics.PerSecond, n)
			for i := range ps {
				ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(50 + i)}
			}
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Errorf("PANIC width=%d n=%d: %v", w, n, r)
					}
				}()
				out := stripANSI(RenderResultGraph(ps, w, 5, n, th))
				maxw := 0
				for _, ln := range strings.Split(out, "\n") {
					if k := len([]rune(ln)); k > maxw {
						maxw = k
					}
				}
				t.Logf("width=%d n=%d -> maxline=%d", w, n, maxw)
			}()
		}
	}
}

func TestProbe_SinglePointStretch(t *testing.T) {
	th := theme.Default()
	out := stripANSI(RenderResultGraph([]metrics.PerSecond{{Sec: 0, RawWPM: 72}}, 82, 5, 1, th))
	t.Logf("single sample at width 82:\n%s", out)
	geo := graphGeometryFor([]metrics.PerSecond{{Sec: 0, RawWPM: 72}}, 82, 1)
	t.Logf("cols=%d screenCols=%d cellsPerSec=%d", geo.Cols, geo.ScreenCols, geo.CellsPerSec)
}

// ---- probe 4: reveal invariants exhaustively ------------------------------

func TestProbe_RevealExhaustive(t *testing.T) {
	th := theme.Default()
	for _, n := range []int{1, 2, 3, 5, 8, 17, 29, 30, 31, 60, 119, 120, 121, 300} {
		for _, w := range []int{34, 40, 51, 60, 80, 88, 120} {
			for _, errs := range []bool{false, true} {
				ps := make([]metrics.PerSecond, n)
				for i := range ps {
					ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(30 + (i*7)%90)}
					if errs && i%5 == 0 {
						ps[i].Errors = 1 + i%3
					}
				}
				settled := strings.Split(stripANSI(RenderResultGraph(ps, w, 5, n, th)), "\n")
				for vis := 0; vis <= n; vis++ {
					got := strings.Split(stripANSI(RenderResultGraph(ps, w, 5, vis, th)), "\n")
					if len(got) != len(settled) {
						t.Fatalf("n=%d w=%d errs=%v vis=%d: lines %d != %d", n, w, errs, vis, len(got), len(settled))
					}
					for i := range settled {
						if len([]rune(got[i])) != len([]rune(settled[i])) {
							t.Fatalf("n=%d w=%d errs=%v vis=%d line %d: %d != %d\n%q\n%q",
								n, w, errs, vis, i, len([]rune(got[i])), len([]rune(settled[i])), got[i], settled[i])
						}
					}
				}
				full := RenderResultGraph(ps, w, 5, n, th)
				if full != RenderResultGraph(ps, w, 5, n+5, th) {
					t.Fatalf("n=%d w=%d: over-clamped visible differs", n, w)
				}
			}
		}
	}
}

// reveal must be monotone: a cell drawn at vis must stay drawn at vis+1.
func TestProbe_RevealMonotone(t *testing.T) {
	th := theme.Default()
	for _, n := range []int{5, 8, 17, 60, 121} {
		for _, w := range []int{40, 60, 88} {
			ps := make([]metrics.PerSecond, n)
			for i := range ps {
				ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(30 + (i*13)%90)}
			}
			prev := ""
			for vis := 0; vis <= n; vis++ {
				cur := stripANSI(RenderResultGraph(ps, w, 5, vis, th))
				if prev != "" {
					pl := strings.Split(prev, "\n")
					cl := strings.Split(cur, "\n")
					for i := range pl {
						pr, cr := []rune(pl[i]), []rune(cl[i])
						for j := range pr {
							if pr[j] != ' ' && cr[j] == ' ' {
								t.Errorf("n=%d w=%d vis=%d: cell (%d,%d) un-drew %q->%q", n, w, vis, i, j, string(pr[j]), string(cr[j]))
								break
							}
						}
					}
				}
				prev = cur
			}
		}
	}
}

// ---- probe 5: x-axis label truthfulness ----------------------------------

func TestProbe_XAxisLabels(t *testing.T) {
	th := theme.Default()
	for _, tc := range []struct{ n, w int }{{8, 82}, {15, 82}, {8, 51}, {30, 82}, {120, 51}, {300, 51}} {
		ps := make([]metrics.PerSecond, tc.n)
		for i := range ps {
			ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(50 + i%40)}
		}
		geo := graphGeometryFor(ps, tc.w, tc.n)
		out := stripANSI(RenderResultGraph(ps, tc.w, 5, tc.n, th))
		lines := strings.Split(out, "\n")
		t.Logf("n=%d w=%d cols=%d screenCols=%d cellsPerSec=%d secPerCell=%d\n  x=%q\n  cellOf(last)=%d",
			tc.n, tc.w, geo.Cols, geo.ScreenCols, geo.CellsPerSec, geo.SecPerCell,
			lines[len(lines)-1], geo.CellOf(geo.Cols-1))
	}
}

// ---- probe 6: stats grid / hero content ---------------------------------

func TestProbe_PanelDump(t *testing.T) {
	for _, w := range []int{60, 80, 120, 200} {
		fmt.Printf("=== termW=%d ===\n%s\n", w, stripANSI(settledPanel(t, shortRunResult(), w, true)))
	}
}
