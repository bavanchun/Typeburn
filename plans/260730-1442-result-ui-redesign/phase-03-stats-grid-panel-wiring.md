---
phase: 3
title: "Stats Grid + Panel Wiring + Invariants"
status: completed
priority: P2
effort: "1d"
dependencies: [1, 2]
---

# Phase 3: Stats Grid + Panel Wiring + Invariants

<!-- Updated: Validation Session 1 - sparkline.go deletion moved here (dead code after graph wired); minMax/xAxisLabels already extracted in Phase 1 -->

**Validation lock (S1):** delete `sparkline.go` (`Sparkline()` +
`sparklineVisible`) after the graph is wired — verified 0 external callers
(History uses its own `sparklineInline` at `screen_history_view.go:98`).
`minMax`/`xAxisLabels` already moved to `result_graph.go` in Phase 1.

## Overview

Restructure the Result panel: replace the bar-sparkline call with the new graph
renderer, arrange char-stats + meta into a **2-column stats grid**, and re-verify
the reveal flow + NO_COLOR/mono layout invariants. Re-golden all Result-screen
teatest snapshots.

## Requirements

- Functional: `renderPanel` order = hero → graph → stats grid (2 cols) → heatmap.
  Stats grid left = `test type` / `raw` / `characters c/i/e`; right =
  `consistency` / `time <N>s`. Graph driven by `PerSecond` via `RenderResultGraph`.
- Non-functional: <200 LOC per touched file; reveal animates the graph draw-in;
  `nocolor_layout_invariant_test.go` green; settled frame byte-stable.

## Architecture

Modify `internal/ui/screen_result_view.go`:
- `renderSparkline` → `renderGraph`: calls `RenderResultGraph(perSec, innerW,
  chartH, visible, th)` where `visible` comes from the existing reveal progress
  (`sparkVisibleBars` reused or a shared `revealVisible` helper).
- New `renderStatsGrid(innerW)`: 2-column `lipgloss.JoinHorizontal`. Left column
  = test-type line (`displayModeLabel`) + raw + characters (3-tuple); right
  column = consistency + time. Labels `RoleTextMuted`, values bold
  `RoleTextPrimary`, incorrect in `RoleError` when >0.
- Panel section order in `renderPanel`: hero → graph → stats grid → heatmap.
- Remove the old single-row `renderCharStats` + `renderMeta` callsites (fold
  their content into `renderStatsGrid`); keep helpers only if reused.

Modify `internal/ui/screen_result_reveal.go` only if the graph needs a distinct
reveal easing (else reuse the shared `visible` progress).

## Related Code Files

- Modify: `internal/ui/screen_result_view.go` (158 LOC → ~140-180)
- Possibly modify: `internal/ui/screen_result_reveal.go` (reveal `visible` source)
- Modify: `internal/ui/screen_result_test.go`, `screen_result_reveal_test.go`, `screen_result_heatmap_test.go` (re-golden)
- Delete: `internal/ui/sparkline.go` (dead after graph wired; `minMax`/`xAxisLabels` already in `result_graph.go`)
- Read-only refs: `internal/ui/result_graph.go` (Phase 1), `internal/ui/screen_result_hero.go` (Phase 2), `internal/ui/mode_label.go` (`displayModeLabel`)

## Implementation Steps

1. Update Result view tests for new section order + 2-col grid (TDD red).
2. Implement `renderGraph` wrapping `RenderResultGraph` with reveal `visible`.
3. Implement `renderStatsGrid` (2-col, labels/values styling, error highlight).
4. Reorder `renderPanel`; drop old `renderCharStats`/`renderMeta` callsites.
5. Re-golden `screen_result_*_test.go` snapshots; run
   `go test ./internal/ui/ -run "TestResult|TestReveal|TestHeatmap|TestNoColor" -race`.
6. Confirm `nocolor_layout_invariant_test.go` still green.

## Success Criteria

- [ ] Panel order: hero → graph → 2-col stats grid → heatmap.
- [ ] Graph animates via reveal; settled byte-stable.
- [ ] Stats grid 2-col with correct/incorrect/extra (3-tuple), error highlight.
- [ ] NO_COLOR/mono layout-invariant green; no hex literals.
- [ ] All re-goldened Result tests green under `-race`.

## Risk Assessment

- **Golden churn volume:** many snapshots change. Mitigate by regolding in one
  focused commit with a clear "intentional re-golden" message (not test weakening).
- **Reveal `visible` mismatch:** graph and old sparkline must use the same
  progress source or the draw-in jitters. Reuse the single `sparkVisibleBars`
  helper (rename to `revealVisible` if it generalizes) — one source of truth.
- **2-col width at 60 cols:** grid may overflow; floor columns / stack vertically
  below a width tier threshold via `width_tier.go`.
