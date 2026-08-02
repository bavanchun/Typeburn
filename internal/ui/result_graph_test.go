package ui

import (
	"strconv"
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
	// The WPM axis fits the observed 60..120 range with a tenth of headroom at
	// each end, so the top tick is 126 and the bottom is 54 — not 0, which would
	// leave the whole curve crammed into the top rows.
	if !strings.Contains(lines[0], "126") {
		t.Errorf("top row missing the fitted WPM ceiling:\n%s", out)
	}
	// The error axis is lifted to a nice number so a lone error does not sit at
	// the ceiling; the run's own maximum is 2.
	if !strings.HasSuffix(strings.TrimRight(lines[0], " "), "4") {
		t.Errorf("top row missing the error-axis ceiling:\n%s", out)
	}
	last := lines[len(lines)-3] // rows: chart..., baseline, x-labels
	if !strings.Contains(last, "54") {
		t.Errorf("bottom chart row missing the fitted WPM floor:\n%s", out)
	}
	if !strings.HasSuffix(strings.TrimRight(last, " "), "0") {
		t.Errorf("bottom chart row missing the error-axis zero:\n%s", out)
	}
}

// A single error must not pin its marker to the top of the plot. Before the
// nice-number ceiling the marker sat at errors/maxErr == 1, which put it on the
// WPM axis's own ceiling and read as a speed spike on a run that never went
// near it.
func TestRenderResultGraph_LoneErrorDoesNotPinToTop(t *testing.T) {
	ps := []metrics.PerSecond{
		{Sec: 0, RawWPM: 80}, {Sec: 1, RawWPM: 82, Errors: 1},
		{Sec: 2, RawWPM: 81}, {Sec: 3, RawWPM: 83},
	}
	lines := strings.Split(stripANSI(renderSettled(ps)), "\n")
	if strings.Contains(lines[0], "x") {
		t.Errorf("a single error was plotted on the top row:\n%s", strings.Join(lines, "\n"))
	}
}

// The error axis ticks must never go up as they descend. Integer truncation
// used to label a one-error run 1, 0, 0.
func TestErrAxisLabel_Monotonic(t *testing.T) {
	tick := func(cr, maxErr int) int {
		v, err := strconv.Atoi(strings.TrimSpace(errAxisLabel(cr, 5, 2, maxErr)))
		if err != nil {
			t.Fatalf("non-numeric error tick at row %d: %v", cr, err)
		}
		return v
	}
	for raw := 1; raw <= 12; raw++ {
		maxErr := errAxisCeiling(raw)
		top, mid, bottom := tick(0, maxErr), tick(2, maxErr), tick(4, maxErr)
		if top <= mid || mid <= bottom {
			t.Errorf("maxErr=%d (ceiling %d): ticks %d/%d/%d are not descending",
				raw, maxErr, top, mid, bottom)
		}
	}
}

// The axis must describe the run that happened. Asserting that "0" and "4"
// appear somewhere passed happily while a 60-second run was labelled out to 80
// seconds, so this checks the last tick against the last measured second.
func TestRenderResultGraph_XAxisSeconds(t *testing.T) {
	for _, n := range []int{2, 8, 15, 30, 60, 120} {
		ps := make([]metrics.PerSecond, n)
		for i := range ps {
			ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(60 + i%20)}
		}
		out := stripANSI(RenderResultGraph(ps, 84, 5, n, theme.Default()))
		lines := strings.Split(out, "\n")
		axis := lines[len(lines)-1]

		var ticks []int
		for _, f := range strings.Fields(axis) {
			v, err := strconv.Atoi(f)
			if err != nil {
				t.Fatalf("n=%d: non-numeric tick %q in %q", n, f, axis)
			}
			ticks = append(ticks, v)
		}
		if len(ticks) < 2 {
			t.Fatalf("n=%d: expected at least two ticks, got %v", n, ticks)
		}
		if ticks[0] != 0 {
			t.Errorf("n=%d: first tick %d, want 0", n, ticks[0])
		}
		// No tick may claim a second the run never reached.
		for _, v := range ticks {
			if v > n-1 {
				t.Errorf("n=%d: tick %d exceeds the last measured second %d\naxis: %q",
					n, v, n-1, axis)
			}
		}
		// Ticks ascend.
		for i := 1; i < len(ticks); i++ {
			if ticks[i] <= ticks[i-1] {
				t.Errorf("n=%d: ticks not ascending: %v", n, ticks)
				break
			}
		}
	}
}

// A tick must sit above the sample it names, not merely be in range.
func TestRenderResultGraph_XAxisTicksAlignToSamples(t *testing.T) {
	const n, width = 30, 84
	ps := make([]metrics.PerSecond, n)
	for i := range ps {
		ps[i] = metrics.PerSecond{Sec: i, RawWPM: 80}
	}
	geo := graphGeometryFor(ps, width, n)
	axis := stripANSI(RenderResultGraph(ps, width, 5, n, theme.Default()))
	lines := strings.Split(axis, "\n")
	row := []rune(lines[len(lines)-1])

	// The label row is offset by the left axis gutter plus one space.
	const offset = leftAxisW + 1
	for sec := 0; sec < n; sec++ {
		want := strconv.Itoa(sec)
		at := offset + geo.CellOf(sec)
		if at+len(want) > len(row) {
			continue
		}
		if string(row[at:at+len(want)]) == want {
			return // found at least one tick sitting exactly over its sample
		}
	}
	t.Errorf("no tick aligned with its sample's cell\naxis: %q", lines[len(lines)-1])
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

// cleanSamples is a short, error-free run — the shape from the field report,
// where the chart has far fewer points than the panel has room for and the
// error axis has nothing to measure.
func cleanSamples() []metrics.PerSecond {
	ps := make([]metrics.PerSecond, 8)
	for i := range ps {
		ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(60 + i*5)}
	}
	return ps
}

// A chart as wide as the run was long is a postage stamp in a full-width panel.
// Short runs must stretch to the space they are given.
func TestRenderResultGraph_ShortRunFillsItsWidth(t *testing.T) {
	const width = 80
	out := stripANSI(RenderResultGraph(cleanSamples(), width, 5, len(cleanSamples()), theme.Default()))
	lines := strings.Split(out, "\n")

	widest := 0
	for _, ln := range lines {
		if n := len([]rune(strings.TrimRight(ln, " "))); n > widest {
			widest = n
		}
	}
	if widest <= len(cleanSamples())+leftAxisW+2 {
		t.Errorf("chart did not stretch: widest line %d for %d samples at width %d\n%s",
			widest, len(cleanSamples()), width, out)
	}
	if widest > width {
		t.Errorf("chart overflowed its width: %d > %d\n%s", widest, width, out)
	}
}

// The stretched line must span the full plot, not stop short of the right edge.
func TestRenderResultGraph_StretchedLineReachesBothEdges(t *testing.T) {
	out := stripANSI(RenderResultGraph(cleanSamples(), 80, 5, len(cleanSamples()), theme.Default()))
	lines := strings.Split(out, "\n")
	baseline := lines[len(lines)-2] // the ┼──── row

	plotEnd := len([]rune(strings.TrimRight(baseline, " ")))
	lastDrawn := 0
	for _, ln := range lines[:len(lines)-2] {
		if n := len([]rune(strings.TrimRight(ln, " "))); n > lastDrawn {
			lastDrawn = n
		}
	}
	if plotEnd-lastDrawn > 2 {
		t.Errorf("line stops %d cells short of the plot's right edge\n%s", plotEnd-lastDrawn, out)
	}
}

// A clean run has no errors to plot, so the right axis is nothing but a column
// of zeroes beside the chart.
func TestRenderResultGraph_NoErrorsHidesRightAxis(t *testing.T) {
	clean := stripANSI(RenderResultGraph(cleanSamples(), 60, 5, len(cleanSamples()), theme.Default()))
	for _, ln := range strings.Split(clean, "\n") {
		if strings.Contains(ln, "├") || strings.Contains(ln, "┴") {
			t.Errorf("clean run drew right-axis chrome: %q\n%s", ln, clean)
		}
	}

	// A run that does have errors must still draw it.
	withErrs := stripANSI(renderSettled(graphSamples()))
	if !strings.Contains(withErrs, "┤") || !strings.Contains(withErrs, "2") {
		t.Errorf("run with errors lost its right axis:\n%s", withErrs)
	}
}

// The reveal-width invariant has to hold for a stretched chart too, not only
// for the sample set that happens to match the cell budget.
func TestRenderResultGraph_StretchedRevealNoReflow(t *testing.T) {
	ps := cleanSamples()
	settled := RenderResultGraph(ps, 80, 5, len(ps), theme.Default())
	for vis := 0; vis <= len(ps); vis++ {
		got := RenderResultGraph(ps, 80, 5, vis, theme.Default())
		sw := strings.Split(stripANSI(settled), "\n")
		gw := strings.Split(stripANSI(got), "\n")
		if len(sw) != len(gw) {
			t.Fatalf("visible=%d: line count %d != settled %d", vis, len(gw), len(sw))
		}
		for i := range sw {
			if len([]rune(sw[i])) != len([]rune(gw[i])) {
				t.Fatalf("visible=%d line %d: width %d != settled %d", vis, i, len([]rune(gw[i])), len([]rune(sw[i])))
			}
		}
	}
}

// Long runs must still downsample rather than stretch.
func TestRenderResultGraph_LongRunStillDownsamples(t *testing.T) {
	ps := make([]metrics.PerSecond, 300)
	for i := range ps {
		ps[i] = metrics.PerSecond{Sec: i, RawWPM: float64(50 + i%40)}
	}
	out := stripANSI(RenderResultGraph(ps, 60, 5, len(ps), theme.Default()))
	for _, ln := range strings.Split(out, "\n") {
		if n := len([]rune(ln)); n > 60 {
			t.Fatalf("downsampled chart overflowed width 60: line width %d", n)
		}
	}
}

// NO_COLOR must not change the geometry for either path.
func TestRenderResultGraph_NoColorIdenticalWhenStretchedAndDownsampled(t *testing.T) {
	long := make([]metrics.PerSecond, 300)
	for i := range long {
		long[i] = metrics.PerSecond{Sec: i, RawWPM: float64(50 + i%40)}
	}
	for name, ps := range map[string][]metrics.PerSecond{
		"stretched":   cleanSamples(),
		"downsampled": long,
	} {
		color := strings.Split(stripANSI(RenderResultGraph(ps, 80, 5, len(ps), theme.Default())), "\n")
		mono := strings.Split(stripANSI(RenderResultGraph(ps, 80, 5, len(ps), theme.Load("default", true))), "\n")
		if len(color) != len(mono) {
			t.Fatalf("%s: line count %d != %d", name, len(mono), len(color))
		}
		for i := range color {
			if len([]rune(color[i])) != len([]rune(mono[i])) {
				t.Errorf("%s line %d: width %d != %d", name, i, len([]rune(mono[i])), len([]rune(color[i])))
			}
		}
	}
}
