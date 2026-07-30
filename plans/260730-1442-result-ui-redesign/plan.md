---
title: Result-Screen UI Redesign (Monkeytype-style)
description: >-
  Redesign the Result screen to Monkeytype fidelity: big wpm + acc hero, a
  dual-axis WPM/Errors line graph, and a 2-column stats grid. Keeps Typeburn's
  heatmap, NO_COLOR/mono invariants, and reveal animation.
status: completed
priority: P2
branch: feat/result-ui-redesign
tags:
  - feature
  - ui
  - result-screen
  - metrics
blockedBy: []
blocks: []
created: '2026-07-30T14:42:00+07:00'
createdBy: 'ak:plan'
source: skill
---

# Result-Screen UI Redesign (Monkeytype-style)

## Overview

Rebuild the post-test Result screen to match the Monkeytype target image:
**two big numbers** (`wpm` + `acc`) up top, a real **dual-axis line graph**
(WPM line + error markers), and a **2-column stats grid**. Data already exists —
`metrics.PerSecond.Errors` is computed but unused today. Mostly additive: a new
graph renderer replaces the bar sparkline on the Result screen. `sparkline.go`
becomes dead code (History uses its own `sparklineInline`; public `Sparkline()`
has 0 callers) → deleted in Phase 3, with `minMax`/`xAxisLabels` moved into the
new graph file. Heatmap kept.

Brainstorm report: `plans/reports/brainstorm-260730-1435-result-ui-redesign.md`
Scout report: `plans/reports/scout-260730-1428-result-ui-redesign.md`

## Resolved Decisions (from brainstorm)

| Decision | Choice |
|---|---|
| Graph | Full dual-axis **line** chart (WPM line left Y + Errors w/ red X markers right Y, X = seconds) |
| Char stats | Keep **3-tuple** (correct/incorrect/extra) |
| Heatmap | **Keep** (below stats grid) |
| Session ts | **Omit** (duration only) |

## Target Layout (top→bottom, inside existing rounded `result` panel)

1. **Hero** — two big-number blocks side-by-side: `wpm <N>` (ASCII big-digits, left) + `acc <N>%` (right). `★ new best` stays on WPM. `raw` + `consistency` become secondary cards under/beside.
2. **Graph** — dual-axis line: WPM braille line (accent, left Y 0..maxWPM) + error `x` markers (error role, right Y 0..maxErr), X-axis = seconds.
3. **Stats grid (2 cols)** — left: `test type` / `raw` / `characters c/i/e` · right: `consistency` / `time <N>s`.
4. **Heatmap** — existing "most missed", unchanged.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Dual-Axis Graph Renderer (TDD)](./phase-01-graph-renderer-tdd.md) | Completed |
| 2 | [Hero: Two Big Numbers](./phase-02-hero-two-big-numbers.md) | Completed |
| 3 | [Stats Grid + Panel Wiring + Invariants](./phase-03-stats-grid-panel-wiring.md) | Completed |
| 4 | [Docs Sync + Full Verify](./phase-04-docs-sync-verify.md) | Completed |

## Acceptance Criteria

- Result hero shows `wpm` and `acc` as two large numbers; new-best `★` on WPM.
- Graph renders a WPM line + per-second error markers on dual Y-axes; X = seconds.
- Stats arranged in a 2-column grid (test type/raw/characters | consistency/time).
- Heatmap still renders below the grid.
- `PerSecond.Errors` now consumed by the graph (previously unused).
- NO_COLOR/mono: layout byte-identical (attributes only change); `nocolor_layout_invariant_test.go` green.
- Reveal animation: every frame layout-identical; settled frame byte-identical to static.
- teatest goldens intentionally re-goldened (no test weakened).
- New file(s) <200 LOC; theme `Role` only, no hex literals in UI code.
- `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` all green.
- `docs/wireframe/mockups.md` §3 + `docs/project-roadmap.md` updated.

## Dependencies

None. All prior plans in `./plans/` are `status: completed`; no overlapping
in-flight work. Result-screen files are owned exclusively by this plan.

## Validation Log

### Session 1 — 2026-07-30
**Trigger:** `/ak:plan validate` (Standard tier: Fact Checker + Contract Verifier)
**Questions asked:** 4

#### Verification Results
- Claims checked: 14
- Verified: 13 | Failed: 1 | Unverified: 0
- Tier: Standard
- **FAILED:** "sparkline.go stays — History reuses it". History uses its own
  `sparklineInline` (`screen_history_view.go:98`); public `Sparkline()`
  (`sparkline.go:24`) has 0 callers; `sparklineVisible` is live only via Result
  (`screen_result_view.go:127`). After redesign → dead code.
- Verified: `BigDigits` (`ascii_big_digits.go:101`), `BigDigitsFixed`
  (`screen_result_reveal.go:89`), `StatCard`/`revealLine`/`accColorRole`/
  `cardProgress`/`countUpValue`/`revealDone`, `sparkVisibleBars`
  (`screen_result_reveal.go:55`, single reveal source), `displayModeLabel`
  (`mode_label.go:6`), Makefile `test-race`/`lint`/`size-check`, `PerSecond.Errors`
  unused today, `TestNoColorInvariant_ResultRevealAndCelebration` exists.

#### Questions & Answers

1. **[Scope]** sparkline.go is dead code after redesign. What now?
   - Options: Delete+extract helpers | Keep untouched
   - **Answer:** Delete, extract shared helpers — move `minMax`/`xAxisLabels` into
     the new graph file if reused; delete the dead bar-sparkline renderer.
2. **[Architecture]** How to draw the WPM line in the cell grid?
   - Options: Braille sub-cell line | Multi-row block bars
   - **Answer:** Braille sub-cell line (U+2800, 2×4 dots) — true continuous line,
     closest to the Monkeytype image.
3. **[Architecture]** Error representation on the graph?
   - Options: x markers only | Errors line + x markers
   - **Answer:** x markers only — red `x` per error second, y-scaled on right axis.
     Mono-distinguishable from the braille WPM line; no overlaid second series.
4. **[Scope]** Graph block row budget (min terminal 60×20)?
   - Options: Compact ~5-6 rows | Taller ~8-10 rows
   - **Answer:** Compact ~5-6 rows — fits alongside hero + 2-col grid + heatmap
     without new degrade rules. Braille = 4 sub-rows/cell, adequate resolution.

#### Confirmed Decisions
- Braille line + x-only markers + ~5-6 row graph + delete sparkline.go.
- One reveal source (`sparkVisibleBars`) drives the graph draw-in.

#### Action Items
- [ ] Phase 1: extract `minMax`/`xAxisLabels` into `result_graph.go`; braille line; x markers; chartH≈5-6.
- [ ] Phase 3: delete `sparkline.go` + its dead `Sparkline()`/`sparklineVisible`; re-golden.
- [ ] Phase 2: note `BigDigitsFixed` lives in `screen_result_reveal.go` (not `ascii_big_digits.go`).

#### Impact on Phases
- Phase 1: gains helper extraction + confirmed braille/x/height.
- Phase 2: corrected `BigDigitsFixed` location ref.
- Phase 3: now owns sparkline.go deletion (was "additive only").

### Whole-Plan Consistency Sweep
- Re-read plan.md + all 4 phase files post-propagation.
- No unresolved contradictions. "sparkline.go stays" removed from Overview;
  deletion reflected in Phase 3. Braille/x-markers/height reflected in Phase 1.
  No stale terms, no duplicate embedded drafts. **Sweep: clean.**

## Unresolved Questions

None at plan gate. Implementation-phase details (exact braille dot mapping,
reveal easing for the line) resolved TDD-style in Phase 1.
