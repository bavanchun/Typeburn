---
phase: 2
title: "Graph Fills Its Width"
status: completed
priority: P1
effort: "4h"
dependencies: [1]
---

# Phase 2: Graph Fills Its Width

## Overview

Make the WPM graph occupy the width it is given instead of the width its data
happens to have, and stop drawing an error axis when there were no errors.

## Requirements

- Functional
  - A test with few data points renders a graph as wide as the panel allows.
  - The right error axis and its `x` markers are omitted entirely when the run
    had zero errors.
  - Downsampling for long runs is unchanged.
- Non-functional
  - Every reveal frame keeps the settled frame's exact line count and per-line
    width — this is already enforced and must stay enforced.
  - `NO_COLOR` renders layout-identical.

## Architecture

### Why the graph is tiny

`result_graph.go:57` is `cols := len(perSec)`. `width` is consulted only to
*downsample* when there is too much data (`result_graph.go:52`), never to
*expand* when there is too little. An 8-second test therefore renders 8 cells
wide no matter the terminal — the postage-stamp chart in the report.

Add the mirror of downsampling: when `len(perSec)` is below the cell budget,
each second occupies more than one cell.

```go
// cellsPerSec is the mirror of secPerCell: short runs stretch so the chart
// fills its panel instead of huddling at the left edge. Integer repetition,
// not interpolation — inventing intermediate WPM values would draw data the
// test never produced.
cellsPerSec := 1
if budget := width - axisOverhead; len(perSec) > 0 && len(perSec) < budget {
    cellsPerSec = budget / len(perSec)
}
```

Integer repetition is a deliberate choice over interpolation. A smoothed curve
between two samples looks better and reports measurements that do not exist; for
a typing test whose whole point is honest metrics, that trade is not available.

### Why the error axis is noise

`result_graph_axes.go:58` renders `errAxisLabel` unconditionally, so a clean
100%-accuracy run draws a right-hand column of `0`, `0`, `0` plus its tick
glyphs — roughly 4 columns of pure noise on the run the user is happiest about.

Gate the whole right axis on `maxErr > 0`. This is a *layout* change, so it must
be decided once per render and applied to every row uniformly, never per-row, or
the reveal-width invariant breaks.

### The invariant that constrains both changes

`TestRenderResultGraph_VisibleBlanksWithoutReflow`
(`result_graph_test.go:109`) walks `visible` from 0 to `len(perSec)` and asserts
every frame has the settled frame's line count and per-line rune width. Both
changes above are width-affecting, so they must be computed from `perSec` and
`width` only — never from `visible`. Any dependence on `visible` fails that test,
which is exactly what it is for.

`TestRenderResultGraph_DualAxisLabels` (`result_graph_test.go:69`) uses samples
carrying 2 errors, so gating on `maxErr > 0` leaves it green; it needs no edit.

## Related Code Files

- Modify: `internal/ui/result_graph.go` (cell stretching)
- Modify: `internal/ui/result_graph_axes.go` (conditional right axis)
- Modify: `internal/ui/result_graph_test.go` (new cases)

## Implementation Steps

1. Extract the axis overhead (currently the literal `- 4 - 3 - 2` at
   `result_graph.go:52`) into a named value, since the right-axis gate now
   changes it.
2. Add `cellsPerSec` stretching. Guard `len(perSec) == 0` and `budget <= 0`.
3. Gate the right axis and the `x` markers on `maxErr > 0`, decided once before
   the row loop.
4. Tests:
   - 8 points at width 80 produce a chart materially wider than 8 cells, and no
     wider than the budget.
   - a zero-error run emits no right-axis tick and no `x`; a run with errors
     still emits both.
   - the reveal invariant holds for a stretched chart (parameterise the existing
     test over a short series as well as the current one).
   - long-run downsampling output is unchanged.
   - `NO_COLOR` line count and widths identical to colored, for both a stretched
     and a downsampled series.

## Success Criteria

- [x] `go test ./internal/ui/ -race -count=1` green with no existing assertion
      weakened
- [x] A short run's chart fills its budget; a long run still downsamples
- [x] Zero-error runs render no right axis
- [x] Reveal-width invariant passes for stretched charts
- [x] `result_graph.go` and `result_graph_axes.go` stay under 200 LOC

## Risk Assessment

**Risk:** stretching interacts with the reveal so a mid-reveal frame is a
different width than the settled one — the exact class of bug the invariant test
exists to catch.
**Mitigation:** width is derived from `perSec` and `width` only. The invariant
test is extended to cover a short (stretched) series, not just the current
sample set.

**Risk:** removing the right axis shifts the chart and silently invalidates the
recorded baselines.
**Mitigation:** intended. Phase 1's baselines are re-recorded here and the diff
is reviewed rather than auto-accepted.
