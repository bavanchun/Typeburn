package ui

import "github.com/bavanchun/Typeburn/v2/internal/metrics"

// Axis chrome widths: the left column carries the WPM scale, the right the
// error scale. minDownsampleCells is the point below which downsampling stops
// being meaningful and the chart is simply drawn as-is.
const (
	leftAxisW          = 4
	rightAxisW         = 3
	minDownsampleCells = 8
)

// graphGeometry is the chart's width policy, resolved once per render. Keeping
// it separate from drawing is what lets the downsample and stretch paths be
// reasoned about — and tested — without a rendered frame.
type graphGeometry struct {
	Samples     []metrics.PerSecond // possibly bucketed
	Visible     int                 // rescaled to match Samples
	Cols        int                 // data columns
	ScreenCols  int                 // cells the chart occupies
	SecPerCell  int                 // seconds per data column after bucketing
	ShowErrAxis bool

	// CellOf maps a sample index to its screen cell.
	CellOf func(i int) int
}

// graphGeometryFor resolves the chart geometry for a sample set and width.
func graphGeometryFor(perSec []metrics.PerSecond, width, visible int) graphGeometry {
	// Whether the error axis will be drawn has to be known before the cell
	// budget, since it costs columns. Bucketing sums errors, so "any error at
	// all" is invariant under downsampling and can be decided from the raw
	// samples — no need to downsample first.
	showErrAxis := false
	for _, ps := range perSec {
		if ps.Errors > 0 {
			showErrAxis = true
			break
		}
	}

	// Cell budget: the width minus the axis columns actually drawn, and their
	// pipes.
	budget := width - leftAxisW - 1
	if showErrAxis {
		budget -= rightAxisW + 1
	}

	// Downsample when there is more data than budget.
	secPerCell := 1
	if budget >= minDownsampleCells && len(perSec) > budget {
		secPerCell = (len(perSec) + budget - 1) / budget
		perSec = bucketSamples(perSec, secPerCell)
		visible = (visible + secPerCell - 1) / secPerCell
	}
	cols := len(perSec)

	// ...and stretch when there is less. Without this the chart is exactly as
	// wide as the run was long, so a short test renders a postage stamp in a
	// full-width panel.
	//
	// Stretching synthesizes no samples: it widens the segment drawSeg already
	// drew between two measured seconds. That segment is, and always was,
	// linearly interpolated — stretching gives it more pixels, not more truth —
	// so read the vertices, not the line between them.
	// Samples are spread from the first cell to the last, taking the whole
	// budget. The alternative — giving every sample an equal cells-per-sample
	// block — leaves the final sample short of the right edge and reads as the
	// chart stopping early.
	//
	// Anything that positions something against this chart (the x-axis labels,
	// error markers) must use CellOf rather than re-deriving a ratio: a second
	// mapping silently disagrees with this one, which is how the axis came to
	// announce times the run never reached.
	// A single sample has nothing to spread between, so claiming the budget
	// would draw one dot at the far left under a full-width baseline.
	screenCols := cols
	if cols > 1 && cols < budget {
		screenCols = budget
	}
	cellOf := func(i int) int {
		if cols < 2 {
			return 0
		}
		return i * (screenCols - 1) / (cols - 1)
	}

	return graphGeometry{
		Samples: perSec, Visible: visible,
		Cols: cols, ScreenCols: screenCols,
		SecPerCell: secPerCell, ShowErrAxis: showErrAxis, CellOf: cellOf,
	}
}
