---
phase: 1
title: "Dual-Axis Graph Renderer (TDD)"
status: completed
priority: P2
effort: "1d"
dependencies: []
---

# Phase 1: Dual-Axis Graph Renderer (TDD)

<!-- Updated: Validation Session 1 - braille line confirmed; x markers only; chartH≈5-6 rows; extract minMax/xAxisLabels from sparkline.go -->

**Validation lock (S1):** braille sub-cell WPM line (U+2800); red `x` error
markers only (no separate errors line); chart block ≈5-6 rows; reuse
`minMax`/`xAxisLabels` by moving them here from `sparkline.go` (DRY).

## Overview

Build a new pure, theme-role-only graph renderer that draws a WPM-over-time
**line** (braille sub-cell dots) plus per-second **error markers** on dual
Y-axes. Consumes `PerSecond.{RawWPM,Errors}` (Errors is currently unused).
Reveal-compatible (`visible` param, like `sparklineVisible`). Mono/NO_COLOR-safe.

## Requirements

- Functional: given `[]PerSecond`, render a fixed-height chart: WPM line in
  `RoleAccent`, error `x` markers in `RoleError`, left Y-axis ticks (WPM
  0/mid/max), right Y-axis ticks (Errors 0/mid/max), X-axis = seconds.
- Non-functional: pure function (no I/O); only `theme.Role` + stdlib; <200 LOC;
  byte-identical output for a settled (`visible==len`) render across calls;
  layout identical under NO_COLOR (attributes only).

## Architecture

New file `internal/ui/result_graph.go`. Public entry:

```go
// RenderResultGraph draws a dual-axis WPM/Errors chart. visible animates the
// rightmost draw-in (len = settled/static, byte-identical to a non-animated render).
func RenderResultGraph(perSec []metrics.PerSecond, width, chartH, visible int, th theme.Theme) string
```

- **Braille line (WPM):** build a 2D dot grid (cols×chartH-rows of 4 sub-rows).
  Map second `i` → x column; RawWPM → y by left-axis scale. OR dots into
  `U+2800`-base braille chars (2×4 sub-cell resolution). Interpolate between
  adjacent samples so the line is continuous (connect the dots).
- **Error markers:** for each second with `Errors>0`, render an `x` at that
  column, y by right-axis (Errors) scale. The `x` replaces the braille cell at
  that column/row so the two series never collide ambiguously.
- **Axes:** left labels `%4.0f` WPM (max/mid/0); right labels `%-3d` Errors;
  baseline `┼──…` with second ticks (reuse `xAxisLabels` logic or share it).
- **Reveal:** positions at/after `visible` blanked to equal-width spaces (mirror
  `sparklineVisible` contract) so layout never reflows during draw-in.
- **Mono/NO_COLOR:** braille + `x` glyphs are color-independent; roles become
  attributes only → layout byte-identical. No new `Role` needed (reuse
  `RoleAccent`, `RoleError`, `RoleTextFaint`).

Edge cases (unit-tested): empty `perSec` → `""`; single second → one point;
all-equal WPM → flat line at mid; zero errors → no markers, right axis `0 0 0`.

## Related Code Files

- Create: `internal/ui/result_graph.go` (~150-190 LOC)
- Create: `internal/ui/result_graph_test.go` (table-driven, golden-ish expected strings)
- Read-only refs: `internal/ui/sparkline.go` (mirror `sparklineVisible`/`minMax`/`xAxisLabels` patterns — do NOT duplicate; extract shared bits to a small helper if DRY applies), `internal/metrics/per_second.go`, `internal/theme/*.go`

## Implementation Steps

1. Write `RenderResultGraph` signature + table tests first (TDD red): empty,
   single, flat, mixed, all-error, no-error, dual-axis label format, settled
   byte-stability, `visible` blanking.
2. Implement braille dot-mapping helper (`wpmToDots`) + OR-into-cell.
3. Implement error-marker overlay (`x` at error columns, right-axis y).
4. Implement dual Y-axis labels + X-axis seconds row.
5. Add `visible` blanking for reveal; assert settled == static.
6. Run `go test ./internal/ui/ -run TestRenderResultGraph -race -count=1`.

## Success Criteria

- [ ] `RenderResultGraph` renders WPM braille line + error `x` markers + dual axes.
- [ ] `PerSecond.Errors` consumed; empty/edge cases handled.
- [ ] Settled render byte-identical to static; `visible` animates without reflow.
- [ ] No hex literals; only `theme.Role`. File <200 LOC.
- [ ] `result_graph_test.go` green under `-race`.

## Risk Assessment

- **Braille math complexity:** mitigate with a tiny dot-grid struct + focused
  tests per shape (line, flat, single point) before composition.
- **WPM vs Errors scale collision in mono:** the `x`-replaces-cell rule keeps
  the two series unambiguous without color. Verify in `nocolor_layout_invariant_test`.
- **DRY with sparkline.go:** share `minMax`/`xAxisLabels` only if extraction is
  clean; otherwise keep local copies (KISS over premature sharing).
