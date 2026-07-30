package ui

import (
	"math"
	"strings"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// brailleBase is the Unicode base code point for the braille patterns block
// (U+2800 BLANK .. U+28FF). Each cell encodes a 2×4 dot block as one rune.
const brailleBase = 0x2800

// brailleBits maps a cell-local dot coordinate [column 0..1][row 0..3] to the
// braille dot bit, per the standard 2×4 layout (dots 1..8).
var brailleBits = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40}, // left column → dots 1,2,3,7
	{0x08, 0x10, 0x20, 0x80}, // right column → dots 4,5,6,8
}

// RenderResultGraph draws a dual-axis WPM/Errors chart from per-second samples.
//
// The WPM series is a continuous braille sub-cell line (RoleAccent, left Y-axis
// 0..maxWPM). Seconds carrying errors render a red "x" marker (RoleError, right
// Y-axis 0..maxErr) in place of that column's braille cell at the error-scaled
// row, so the two series never ambiguously overlap — including under NO_COLOR/mono
// where roles collapse to attributes but layout is byte-identical.
//
// One braille cell is drawn per second (X-axis = seconds); runs longer than the
// available `width` downsample into equal buckets (mean WPM, summed errors) so
// long Time-mode tests never overflow the panel. `visible` (in input-sample
// units) animates a rightward draw-in: columns at/after the mapped cell blank
// to equal-width spaces so the layout never reflows; visible==len(perSec) is
// byte-identical to static.
func RenderResultGraph(perSec []metrics.PerSecond, width, chartH, visible int, th theme.Theme) string {
	if len(perSec) == 0 {
		return ""
	}
	if chartH <= 0 {
		chartH = 5
	}
	if visible < 0 {
		visible = 0
	}
	if visible > len(perSec) {
		visible = len(perSec)
	}

	// Downsample to the chart cell budget (width minus axis labels and pipes).
	secPerCell := 1
	if maxCells := width - 4 - 3 - 2; maxCells >= 8 && len(perSec) > maxCells {
		secPerCell = (len(perSec) + maxCells - 1) / maxCells
		perSec = bucketSamples(perSec, secPerCell)
		visible = (visible + secPerCell - 1) / secPerCell
	}
	cols := len(perSec)

	// Scales: WPM 0..maxWPM (left), Errors 0..maxErr (right).
	wpmVals := make([]float64, cols)
	maxErr := 0
	for i, ps := range perSec {
		wpmVals[i] = ps.RawWPM
		if ps.Errors > maxErr {
			maxErr = ps.Errors
		}
	}
	_, maxWPM := minMax(wpmVals)
	if maxWPM <= 0 {
		maxWPM = 1 // avoid divide-by-zero; flat line parks at the bottom
	}

	gridH := chartH * 4
	gridW := cols * 2 // 2 dot-columns per cell let the line interpolate smoothly

	wpmY := func(wpm float64) int {
		return clampInt(int(math.Round((1-wpm/maxWPM)*float64(gridH-1))), 0, gridH-1)
	}

	// Dot grid for the WPM line; sample i lives at dot-column 2i, with segment
	// interpolation filling the gaps so the line is continuous.
	dots := make([][]bool, gridH)
	for r := range dots {
		dots[r] = make([]bool, gridW)
	}
	prevY := -1
	for i := 0; i < visible; i++ {
		yi := wpmY(perSec[i].RawWPM)
		dots[yi][2*i] = true
		if prevY >= 0 {
			drawSeg(dots, 2*(i-1), prevY, 2*i, yi)
		}
		prevY = yi
	}

	// Error-marker cell-row per column (−1 = none); only visible, error seconds.
	errRow := make([]int, cols)
	for i := range errRow {
		errRow[i] = -1
	}
	if maxErr > 0 {
		for i := 0; i < visible; i++ {
			if perSec[i].Errors > 0 {
				ey := int(math.Round((1 - float64(perSec[i].Errors)/float64(maxErr)) * float64(gridH-1)))
				errRow[i] = clampInt(ey, 0, gridH-1) / 4
			}
		}
	}

	accent := th.Style(theme.RoleAccent)
	errStyle := th.Style(theme.RoleError)
	faint := th.Style(theme.RoleTextFaint)

	midRow := (chartH - 1) / 2
	const leftW, rightW = 4, 3

	var b strings.Builder
	for cr := 0; cr < chartH; cr++ {
		tick := cr == 0 || cr == chartH-1 || cr == midRow
		b.WriteString(faint.Render(wpmAxisLabel(cr, chartH, midRow, maxWPM)))
		b.WriteString(faint.Render(axisPipe(true, tick)))
		for cc := 0; cc < cols; cc++ {
			switch {
			case cc >= visible:
				b.WriteString(" ")
			case errRow[cc] == cr:
				b.WriteString(errStyle.Render("x"))
			case brailleAt(dots, cr, cc) != 0:
				b.WriteString(accent.Render(string(brailleAt(dots, cr, cc))))
			default:
				b.WriteString(" ")
			}
		}
		b.WriteString(faint.Render(axisPipe(false, tick)))
		b.WriteString(faint.Render(errAxisLabel(cr, chartH, midRow, maxErr)))
		if cr < chartH-1 {
			b.WriteString("\n")
		}
	}
	// Baseline spanning both axes, then the per-second X-axis label row.
	b.WriteString("\n")
	b.WriteString(faint.Render(strings.Repeat(" ", leftW) + "┼" + strings.Repeat("─", cols) + "┴" + strings.Repeat(" ", rightW)))
	b.WriteString("\n")
	b.WriteString(faint.Render(strings.Repeat(" ", leftW) + " " + xAxisLabels(cols, secPerCell) + " " + strings.Repeat(" ", rightW)))
	return b.String()
}

// bucketSamples (downsampling) and the axis/scale helpers live in
// result_graph_axes.go.

// drawSeg connects (x0,y0)-(x1,y1) on the dot grid by sampling densely enough to
// fill both the vertical and horizontal spans, so flat segments render solid and
// steep segments have no vertical gaps.
func drawSeg(dots [][]bool, x0, y0, x1, y1 int) {
	span := y1 - y0
	if span < 0 {
		span = -span
	}
	if dx := x1 - x0; dx > span {
		span = dx
	}
	if span < 2 {
		span = 2
	}
	for s := 0; s <= span; s++ {
		t := float64(s) / float64(span)
		x := int(math.Round(float64(x0) + t*float64(x1-x0)))
		y := int(math.Round(float64(y0) + t*float64(y1-y0)))
		if y >= 0 && y < len(dots) && x >= 0 && x < len(dots[y]) {
			dots[y][x] = true
		}
	}
}

// brailleAt packs the 2×4 dot block at cell (cr,cc) into a braille rune; returns
// 0 when the block is empty so the caller emits a plain space of equal width.
func brailleAt(dots [][]bool, cr, cc int) rune {
	var b byte
	for dx := 0; dx < 2; dx++ {
		for dy := 0; dy < 4; dy++ {
			r, c := cr*4+dy, cc*2+dx
			if r < len(dots) && c < len(dots[r]) && dots[r][c] {
				b |= brailleBits[dx][dy]
			}
		}
	}
	if b == 0 {
		return 0
	}
	return rune(brailleBase) + rune(b)
}

// Axis labels, tick joins, and scale helpers live in result_graph_axes.go.
