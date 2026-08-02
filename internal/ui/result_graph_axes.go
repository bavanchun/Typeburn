package ui

import (
	"fmt"
	"strings"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
)

// bucketSamples folds perSec into ceil(len/secPerCell) buckets: mean RawWPM,
// summed Errors. Sec keeps each bucket's starting second.
func bucketSamples(perSec []metrics.PerSecond, secPerCell int) []metrics.PerSecond {
	out := make([]metrics.PerSecond, 0, (len(perSec)+secPerCell-1)/secPerCell)
	for i := 0; i < len(perSec); i += secPerCell {
		end := i + secPerCell
		if end > len(perSec) {
			end = len(perSec)
		}
		var b metrics.PerSecond
		b.Sec = perSec[i].Sec
		for _, ps := range perSec[i:end] {
			b.RawWPM += ps.RawWPM
			b.Errors += ps.Errors
		}
		b.RawWPM /= float64(end - i)
		out = append(out, b)
	}
	return out
}

// axisPipe returns the left (┤/│) or right (├/│) axis join for a row; tick rows
// carry the horizontal stub, plain rows a continuous vertical line.
func axisPipe(left, tick bool) string {
	if tick {
		if left {
			return "┤"
		}
		return "├"
	}
	return "│"
}

// wpmAxisRange widens the observed WPM range by a tenth at each end and returns
// the axis bounds.
//
// Scaling from zero wastes the plot: a run that lived between 55 and 85 wpm drew
// its whole curve in the top two of five rows and left the rest blank, which
// reads as missing data rather than as a stable pace. A run that never varied
// still needs a non-zero span, hence the floor.
func wpmAxisRange(lo, hi float64) (float64, float64) {
	pad := (hi - lo) * 0.1
	if pad <= 0 {
		pad = hi * 0.1
	}
	if pad < 1 {
		pad = 1
	}
	lo, hi = lo-pad, hi+pad
	if lo < 0 {
		lo = 0
	}
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

// errAxisCeiling lifts the error scale to a nice number.
//
// Without it, the single second that happens to hold the run's maximum sits at
// errors/maxErr == 1 and pins its marker to the top row — floating at the WPM
// axis's ceiling on a run that never went near it, so a lone typo reads as a
// speed spike. The floor also keeps the axis labels monotonic: maxErr of 1
// truncated to the ticks 1, 0, 0.
func errAxisCeiling(maxErr int) int {
	if maxErr <= 0 {
		return 0
	}
	if maxErr < 4 {
		return 4
	}
	return maxErr
}

// wpmAxisLabel renders the left Y-axis WPM tick (top, midpoint, bottom) for the
// fitted range.
func wpmAxisLabel(cr, chartH, midRow int, loWPM, hiWPM float64) string {
	switch {
	case cr == 0:
		return fmt.Sprintf("%4.0f", hiWPM)
	case cr == chartH-1:
		return fmt.Sprintf("%4.0f", loWPM)
	case cr == midRow:
		return fmt.Sprintf("%4.0f", (loWPM+hiWPM)/2)
	default:
		return strings.Repeat(" ", 4)
	}
}

// errAxisLabel renders the right Y-axis error tick (top=max, mid=max/2, bottom=0).
func errAxisLabel(cr, chartH, midRow, maxErr int) string {
	switch {
	case cr == 0:
		return fmt.Sprintf("%-3d", maxErr)
	case cr == chartH-1:
		return "0  "
	case cr == midRow:
		return fmt.Sprintf("%-3d", maxErr/2)
	default:
		return strings.Repeat(" ", 3)
	}
}

// clampInt pins v into [lo,hi].
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// minMax returns the minimum and maximum values in vals. Assumes len(vals) > 0.
func minMax(vals []float64) (min, max float64) {
	min, max = vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}

// xAxisLabels builds a compact width-n x-axis label row. Markers sit at cell
// intervals of 4 (every cell for short charts) and print the cell's starting
// second (cell index × secPerCell, 1 when the chart is not downsampled).
// Returns a plain string; the caller applies styling.
// xAxisLabels renders the second ticks under the chart. Positions come from the
// same sample→cell mapping the plot uses, not from a cells-per-second ratio:
// the two disagree once a chart is stretched, and a label placed by the ratio
// announces a time the run never reached — a 60-second test was labelled out to
// 80 seconds.
//
// cellOf must be the plot's own mapping. n is the chart's cell width, cols the
// number of samples, and secPerCell the bucketing factor applied by
// downsampling.
func xAxisLabels(n, cols, secPerCell int, cellOf func(int) int) string {
	if n == 0 || cols == 0 {
		return ""
	}
	if secPerCell < 1 {
		secPerCell = 1
	}

	// Step over samples, widening until consecutive labels cannot collide.
	step := 1
	for step < cols && cellOf(step)-cellOf(0) < 4 {
		step++
	}

	parts := make([]string, 0, n)
	lastEnd := 0
	for i := 0; i < cols; i += step {
		label := fmt.Sprintf("%d", i*secPerCell)
		at := cellOf(i)
		if at < lastEnd || at+len(label) > n {
			continue
		}
		for lastEnd < at {
			parts = append(parts, " ")
			lastEnd++
		}
		parts = append(parts, label)
		lastEnd += len(label)
	}
	for lastEnd < n {
		parts = append(parts, " ")
		lastEnd++
	}
	return strings.Join(parts, "")
}
