package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// graphSamples is the shared mixed-shape fixture: rising WPM with two error
// seconds (sec 1 → 2 errors, sec 3 → 1 error).
func graphSamples() []metrics.PerSecond {
	return []metrics.PerSecond{
		{Sec: 0, RawWPM: 60},
		{Sec: 1, RawWPM: 84, Errors: 2},
		{Sec: 2, RawWPM: 96},
		{Sec: 3, RawWPM: 108, Errors: 1},
		{Sec: 4, RawWPM: 120},
	}
}

func renderSettled(ps []metrics.PerSecond) string {
	return RenderResultGraph(ps, 60, 5, len(ps), theme.Default())
}

func TestRenderResultGraph_Empty(t *testing.T) {
	if out := RenderResultGraph(nil, 60, 5, 0, theme.Default()); out != "" {
		t.Errorf("empty input: want \"\", got %q", out)
	}
}

func TestRenderResultGraph_ContainsBrailleLine(t *testing.T) {
	out := renderSettled(graphSamples())
	found := false
	for _, r := range stripANSI(out) {
		if r >= 0x2800 && r <= 0x28FF {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected braille line chars in graph:\n%s", out)
	}
}

func TestRenderResultGraph_ErrorMarkers(t *testing.T) {
	out := stripANSI(renderSettled(graphSamples()))
	if got := strings.Count(out, "x"); got != 2 {
		t.Errorf("want 2 error x markers (sec 1, sec 3), got %d:\n%s", got, out)
	}
}

func TestRenderResultGraph_NoErrors_NoMarkers(t *testing.T) {
	ps := graphSamples()
	for i := range ps {
		ps[i].Errors = 0
	}
	out := stripANSI(renderSettled(ps))
	if strings.Contains(out, "x") {
		t.Errorf("no-error run must have no x markers:\n%s", out)
	}
	// Right axis collapses to zeros when maxErr == 0.
	if !strings.Contains(out, "0") {
		t.Errorf("expected zero right-axis labels:\n%s", out)
	}
}

func TestRenderResultGraph_DualAxisLabels(t *testing.T) {
	out := stripANSI(renderSettled(graphSamples()))
	lines := strings.Split(out, "\n")
	// Top row: left label = max WPM (120), right label = max errors (2).
	if !strings.Contains(lines[0], "120") {
		t.Errorf("top row missing left max WPM label:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(lines[0], " "), "2") {
		t.Errorf("top row missing right max-errors label:\n%s", out)
	}
	// Bottom chart row carries the 0 baselines on both axes.
	last := lines[len(lines)-3] // rows: chart..., baseline, x-labels
	if !strings.Contains(last, "0") {
		t.Errorf("bottom chart row missing 0 labels:\n%s", out)
	}
}

func TestRenderResultGraph_XAxisSeconds(t *testing.T) {
	out := stripANSI(renderSettled(graphSamples()))
	lines := strings.Split(out, "\n")
	xrow := lines[len(lines)-1]
	if !strings.Contains(xrow, "0") || !strings.Contains(xrow, "4") {
		t.Errorf("x-axis row missing second markers:\n%q", xrow)
	}
}

func TestRenderResultGraph_SettledByteStable(t *testing.T) {
	ps := graphSamples()
	a := RenderResultGraph(ps, 60, 5, len(ps), theme.Default())
	b := RenderResultGraph(ps, 60, 5, len(ps), theme.Default())
	if a != b {
		t.Error("settled render must be deterministic across calls")
	}
	// visible > len clamps to len → still identical.
	c := RenderResultGraph(ps, 60, 5, len(ps)+3, theme.Default())
	if a != c {
		t.Error("visible beyond len must clamp and match settled render")
	}
}

func TestRenderResultGraph_VisibleBlanksWithoutReflow(t *testing.T) {
	ps := graphSamples()
	settled := RenderResultGraph(ps, 60, 5, len(ps), theme.Default())
	for vis := 0; vis <= len(ps); vis++ {
		got := RenderResultGraph(ps, 60, 5, vis, theme.Default())
		sw := strings.Split(stripANSI(settled), "\n")
		gw := strings.Split(stripANSI(got), "\n")
		if len(sw) != len(gw) {
			t.Fatalf("visible=%d: line count %d != settled %d", vis, len(gw), len(sw))
		}
		for i := range sw {
			if len([]rune(sw[i])) != len([]rune(gw[i])) {
				t.Fatalf("visible=%d line %d: width %d != settled %d\n%q\n%q",
					vis, i, len([]rune(gw[i])), len([]rune(sw[i])), gw[i], sw[i])
			}
		}
	}
	// Fully-revealed animated frame must be byte-identical to static.
	if settled != RenderResultGraph(ps, 60, 5, len(ps), theme.Default()) {
		t.Error("fully-revealed frame must equal static render")
	}
}

func TestRenderResultGraph_SinglePoint(t *testing.T) {
	out := renderSettled([]metrics.PerSecond{{Sec: 0, RawWPM: 72}})
	if out == "" {
		t.Fatal("single sample should render a chart, not empty")
	}
	if !strings.Contains(stripANSI(out), "72") {
		t.Errorf("expected max label 72:\n%s", out)
	}
}

func TestRenderResultGraph_FlatLine(t *testing.T) {
	ps := []metrics.PerSecond{
		{Sec: 0, RawWPM: 80}, {Sec: 1, RawWPM: 80},
		{Sec: 2, RawWPM: 80}, {Sec: 3, RawWPM: 80},
	}
	out := stripANSI(renderSettled(ps))
	// All-equal WPM scales 0..80 → line sits at the top row; must not panic and
	// must show the max label.
	if !strings.Contains(out, "80") {
		t.Errorf("flat line: expected 80 max label:\n%s", out)
	}
}

func TestRenderResultGraph_ZeroWPM(t *testing.T) {
	ps := []metrics.PerSecond{{Sec: 0, RawWPM: 0}, {Sec: 1, RawWPM: 0}}
	out := renderSettled(ps)
	if out == "" {
		t.Fatal("zero-WPM run should still render axes")
	}
}

func TestRenderResultGraph_NoColorLayoutIdentical(t *testing.T) {
	ps := graphSamples()
	colored := RenderResultGraph(ps, 60, 5, len(ps), theme.Default())
	mono := RenderResultGraph(ps, 60, 5, len(ps), theme.Load("default", true))
	cl := strings.Split(stripANSI(colored), "\n")
	ml := strings.Split(stripANSI(mono), "\n")
	if len(cl) != len(ml) {
		t.Fatalf("line count differs under NO_COLOR: %d != %d", len(ml), len(cl))
	}
	for i := range cl {
		if cl[i] != ml[i] {
			t.Errorf("line %d differs under NO_COLOR:\n%q\n%q", i, cl[i], ml[i])
		}
	}
}

func TestRenderResultGraph_LongRunDownsamples(t *testing.T) {
	ps := make([]metrics.PerSecond, 120)
	for i := range ps {
		ps[i] = metrics.PerSecond{Sec: i, RawWPM: 80 + float64(i%20), Errors: i % 7 / 6}
	}
	const width = 56 // TierNarrow inner width
	out := stripANSI(RenderResultGraph(ps, width, 5, len(ps), theme.Default()))
	for i, line := range strings.Split(out, "\n") {
		if got := len([]rune(line)); got > width {
			t.Errorf("line %d width %d exceeds panel width %d:\n%q", i, got, width, line)
		}
	}
	// Reveal blanking must still be reflow-free after downsampling.
	settled := strings.Split(out, "\n")
	half := strings.Split(stripANSI(RenderResultGraph(ps, width, 5, 60, theme.Default())), "\n")
	if len(settled) != len(half) {
		t.Fatalf("downsampled reveal changed line count: %d != %d", len(half), len(settled))
	}
	for i := range settled {
		if len([]rune(settled[i])) != len([]rune(half[i])) {
			t.Errorf("downsampled reveal reflowed line %d", i)
		}
	}
}
