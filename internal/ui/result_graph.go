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

	geo := graphGeometryFor(perSec, width, visible)
	perSec, visible = geo.Samples, geo.Visible
	showErrAxis, secPerCell := geo.ShowErrAxis, geo.SecPerCell
	cols, screenCols, cellsPerSec := geo.Cols, geo.ScreenCols, geo.CellsPerSec
	cellOf := geo.CellOf

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
	gridW := screenCols * 2 // 2 dot-columns per cell let the line interpolate smoothly

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
		x := 2 * cellOf(i)
		dots[yi][x] = true
		if prevY >= 0 {
			drawSeg(dots, 2*cellOf(i-1), prevY, x, yi)
		}
		prevY = yi
	}

	// Error-marker cell-row per screen column (−1 = none); only visible seconds
	// that actually carried an error.
	errRow := make([]int, screenCols)
	for i := range errRow {
		errRow[i] = -1
	}
	if maxErr > 0 {
		for i := 0; i < visible; i++ {
			if perSec[i].Errors > 0 {
				ey := int(math.Round((1 - float64(perSec[i].Errors)/float64(maxErr)) * float64(gridH-1)))
				errRow[cellOf(i)] = clampInt(ey, 0, gridH-1) / 4
			}
		}
	}

	// Rightmost screen cell the reveal has reached. -1 blanks the whole chart,
	// which is the visible==0 first frame.
	revealedTo := -1
	if visible > 0 {
		revealedTo = cellOf(visible - 1)
	}
	if visible >= cols {
		revealedTo = screenCols - 1
	}

	accent := th.Style(theme.RoleAccent)
	errStyle := th.Style(theme.RoleError)
	faint := th.Style(theme.RoleTextFaint)

	midRow := (chartH - 1) / 2
	rightW := rightAxisW
	if !showErrAxis {
		rightW = 0
	}

	var b strings.Builder
	for cr := 0; cr < chartH; cr++ {
		tick := cr == 0 || cr == chartH-1 || cr == midRow
		b.WriteString(faint.Render(wpmAxisLabel(cr, chartH, midRow, maxWPM)))
		b.WriteString(faint.Render(axisPipe(true, tick)))
		for cc := 0; cc < screenCols; cc++ {
			switch {
			case cc > revealedTo:
				b.WriteString(" ")
			case errRow[cc] == cr:
				b.WriteString(errStyle.Render("x"))
			case brailleAt(dots, cr, cc) != 0:
				b.WriteString(accent.Render(string(brailleAt(dots, cr, cc))))
			default:
				b.WriteString(" ")
			}
		}
		if showErrAxis {
			b.WriteString(faint.Render(axisPipe(false, tick)))
			b.WriteString(faint.Render(errAxisLabel(cr, chartH, midRow, maxErr)))
		}
		if cr < chartH-1 {
			b.WriteString("\n")
		}
	}
	// Baseline spanning both axes, then the per-second X-axis label row.
	b.WriteString("\n")
	rightCap := "┴"
	if !showErrAxis {
		rightCap = ""
	}
	b.WriteString(faint.Render(strings.Repeat(" ", leftAxisW) + "┼" + strings.Repeat("─", screenCols) + rightCap + strings.Repeat(" ", rightW)))
	b.WriteString("\n")
	tail := " "
	if !showErrAxis {
		tail = ""
	}
	b.WriteString(faint.Render(strings.Repeat(" ", leftAxisW) + " " + xAxisLabels(screenCols, secPerCell, cellsPerSec) + tail + strings.Repeat(" ", rightW)))
	return b.String()
}

// bucketSamples (downsampling) and the axis/scale helpers live in
// result_graph_axes.go.
