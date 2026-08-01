package ui

import (
	"flag"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// update rewrites the recorded baselines instead of comparing against them.
// Run with -update after an intentional layout change, then read the diff.
var updateBaselines = flag.Bool("update", false, "rewrite result layout baselines")

// baselineWidths spans the range that matters: 60 is the degraded-gate
// boundary, 80 the conventional terminal, 200 a full-screen one.
var baselineWidths = []int{60, 80, 120, 200}

// shortRunResult is an 8-second run with no errors — the case from the report,
// where the chart has far fewer points than the panel has room for and the
// error axis has nothing to show.
func shortRunResult() ResultMsg {
	per := make([]metrics.PerSecond, 8)
	for i := range per {
		per[i] = metrics.PerSecond{
			Sec: i, RawWPM: float64(60 + i*5), CorrectChars: 6, TotalChars: 6,
		}
	}
	return ResultMsg{
		Result: metrics.Result{
			NetWPM: 74, RawWPM: 74, Accuracy: 100, Consistency: 59,
			CorrectChars: 52, DurationMs: 8000, PerSecond: per,
		},
		Mode:   config.ModeWords,
		Length: 10,
	}
}

// erroringRun exercises the wider of the two chart shapes: errors mean the
// right-hand axis is drawn, costing columns the clean run does not spend.
func erroringRun() []metrics.PerSecond {
	ps := make([]metrics.PerSecond, 12)
	for i := range ps {
		ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(55 + i*3)}
		if i%3 == 0 {
			ps[i].Errors = i%4 + 1
		}
	}
	return ps
}

// settledPanel renders the panel with the reveal fully complete, so snapshots
// capture the static frame rather than an animation step.
func settledPanel(t *testing.T, msg ResultMsg, termW int, noColor bool) string {
	t.Helper()
	m := NewResult(msg, theme.Load("default", noColor), config.DefaultKeymap()).SetSize(termW, 40)
	m.revealStartMs, m.nowMs = 0, 1<<40
	return m.renderPanel()
}

func baselinePath(termW int, noColor bool) string {
	mode := "color"
	if noColor {
		mode = "nocolor"
	}
	return filepath.Join("testdata", "result_baseline_"+mode+"_"+strconv.Itoa(termW)+".txt")
}

// Baselines exist to make later layout diffs reviewable, not to assert that the
// current layout is correct. When a phase changes the layout on purpose, rerun
// with -update and read what moved.
func TestResultBaselines(t *testing.T) {
	for _, w := range baselineWidths {
		for _, noColor := range []bool{false, true} {
			got := settledPanel(t, shortRunResult(), w, noColor)
			path := baselinePath(w, noColor)

			if *updateBaselines {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatalf("mkdir testdata: %v", err)
				}
				if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
					t.Fatalf("write %s: %v", path, err)
				}
				continue
			}

			want, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s (run with -update to record): %v", path, err)
			}
			if got != string(want) {
				t.Errorf("%s changed; rerun with -update and review the diff\n--- got ---\n%s", path, got)
			}
		}
	}
}

// The floor has to hold for absurd inputs, not just plausible ones — a
// zero-width or negative width reaches here from a terminal that reports its
// size before it has one.
func TestLayoutFor_NeverDegenerate(t *testing.T) {
	for w := -10; w <= 400; w++ {
		lay := layoutFor(w)
		if lay.InnerW < 1 {
			t.Fatalf("termW=%d: InnerW=%d, want >= 1", w, lay.InnerW)
		}
		if lay.PanelW < resultMinPanelW {
			t.Fatalf("termW=%d: PanelW=%d, want >= %d", w, lay.PanelW, resultMinPanelW)
		}
	}
}

// Wider terminals must never yield a narrower panel.
func TestLayoutFor_Monotonic(t *testing.T) {
	prev := layoutFor(1)
	for w := 2; w <= 400; w++ {
		cur := layoutFor(w)
		if cur.PanelW < prev.PanelW {
			t.Fatalf("termW=%d: PanelW %d < %d at termW=%d", w, cur.PanelW, prev.PanelW, w-1)
		}
		prev = cur
	}
}

// Whatever the geometry, the rendered panel must actually be the width the
// layout claims — otherwise centring math in later phases is built on a lie.
func TestLayoutFor_MatchesRenderedWidth(t *testing.T) {
	for _, w := range baselineWidths {
		lay := layoutFor(w)
		panel := settledPanel(t, shortRunResult(), w, true)
		for i, line := range strings.Split(panel, "\n") {
			if got := lipgloss.Width(line); got != lay.PanelW {
				t.Errorf("termW=%d line %d: rendered width %d, want PanelW=%d",
					w, i, got, lay.PanelW)
			}
		}
		// And it must never spill past the terminal.
		if lay.PanelW > w && w >= resultMinPanelW+resultPanelChrome {
			t.Errorf("termW=%d: panel width %d exceeds the terminal", w, lay.PanelW)
		}
		// InnerW must be the space a section can actually fill. If it
		// over-reports, a section that uses all of it wraps and breaks the
		// border — which is exactly what an under-counted inset caused.
		//
		// Both error states matter: a run with errors draws a right-hand axis
		// the clean run does not, so it is the wider of the two.
		for name, ps := range map[string][]metrics.PerSecond{
			"clean":  shortRunResult().Result.PerSecond,
			"errors": erroringRun(),
		} {
			graph := RenderResultGraph(ps, lay.InnerW, 5, len(ps), theme.Load("default", true))
			for i, line := range strings.Split(graph, "\n") {
				if got := lipgloss.Width(line); got > lay.InnerW {
					t.Errorf("termW=%d %s graph line %d: width %d exceeds InnerW %d",
						w, name, i, got, lay.InnerW)
				}
			}
		}
	}
}

// Past the cap the panel must stop growing. A panel that tracks the terminal
// forever is how a 200-column screen ended up holding 35 columns of content.
func TestLayoutFor_CapsContentWidth(t *testing.T) {
	for w := 60; w <= 400; w++ {
		if got := layoutFor(w).InnerW; got > resultMaxContentW {
			t.Fatalf("termW=%d: InnerW=%d exceeds cap %d", w, got, resultMaxContentW)
		}
	}
}

// NO_COLOR may change attributes, never geometry.
func TestResultPanel_NoColorLayoutIdentical(t *testing.T) {
	for _, w := range baselineWidths {
		color := strings.Split(settledPanel(t, shortRunResult(), w, false), "\n")
		mono := strings.Split(settledPanel(t, shortRunResult(), w, true), "\n")
		if len(color) != len(mono) {
			t.Fatalf("termW=%d: line count %d != %d", w, len(mono), len(color))
		}
		for i := range color {
			if a, b := lipgloss.Width(color[i]), lipgloss.Width(mono[i]); a != b {
				t.Errorf("termW=%d line %d: color width %d != no-color %d", w, i, a, b)
			}
		}
	}
}

// The panel must be centred in the terminal the user actually sees. This
// asserts against the rendered View, not against layoutFor: the root already
// centres the frame with lipgloss.Place, so a layout that also pads is centred
// twice and drifts right. A test that only checked layoutFor's own arithmetic
// passed while the panel sat visibly off-centre.
func TestResultView_PanelIsCentredOnScreen(t *testing.T) {
	for _, w := range []int{80, 120, 160, 200, 300} {
		m := NewResult(shortRunResult(), theme.Load("default", true), config.DefaultKeymap()).SetSize(w, 45)
		m.revealStartMs, m.nowMs = 0, 1<<40

		var top string
		for _, ln := range strings.Split(m.View(), "\n") {
			if plain := stripANSI(ln); strings.Contains(plain, "╭") {
				top = plain
				break
			}
		}
		if top == "" {
			t.Fatalf("termW=%d: no panel border found in the rendered view", w)
		}

		left := lipgloss.Width(top) - lipgloss.Width(strings.TrimLeft(top, " "))
		panelW := lipgloss.Width(strings.TrimSpace(top))
		right := w - left - panelW
		if diff := left - right; diff > 1 || diff < -1 {
			t.Errorf("termW=%d: left margin %d vs right %d (panel %d) — not centred on screen",
				w, left, right, panelW)
		}
	}
}
