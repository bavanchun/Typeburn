---
title: "Result Responsive Layout"
description: >-
  Make the Result screen use the width it has: cap and centre the panel, stretch
  the WPM graph, bound the stats gutter, drop the duplicated raw/consistency
  pair, and hide the error axis on clean runs. No new visual language.
status: in-progress
priority: P2
effort: "1.5d"
branch: feat/result-responsive-layout
tags: [ui, result, layout]
blockedBy: []
blocks: []
created: 2026-08-01
---

# Result Responsive Layout

## Overview

The Result screen looks sparse on a wide terminal. The cause is not taste — it
is that **no part of the screen adapts to width**. Rendered at 200 columns, the
panel is 192 wide and holds about 35 columns of content, with the right-hand
stats column flung 94 characters from the left one.

Four independent, conflicting width policies, all confirmed by reading the code
and by rendering the panel at 80 and 200 columns before writing this plan:

| # | Symptom | Cause |
|---|---|---|
| 1 | Panel grows without bound | `screen_result_view.go:87` — `panelW := m.w - 8` |
| 2 | Hero always flush-left | `screen_result_hero.go:22` — `_ = innerW` (parameter discarded) |
| 3 | 8-second test → 8-cell chart | `result_graph.go:57` — `cols := len(perSec)`; `width` only ever downsamples |
| 4 | Right stats column far away | `screen_result_view.go:160` — `colW := innerW / 2` |

Two further defects are independent of width and fixed regardless:

| # | Defect | Locations |
|---|---|---|
| 5 | `raw` and `consistency` rendered **twice** | hero `:68`,`:72` **and** grid `:145`,`:149` |
| 6 | Right error axis draws `0/0/0` on a 100%-accuracy run | `result_graph_axes.go:58` |

Rendering at 80 columns shows the same problems in milder form, so this is not
purely a wide-terminal issue — the design under-uses horizontal space at every
width and simply degrades further as the terminal grows. That is why the v2.6.0
redesign, reviewed at ~80 columns, did not catch it.

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | One width policy the whole screen obeys | P1 |
| 2 | Graph fills its panel instead of its data | P1 |
| 3 | Panel capped and centred rather than stretched | P1 |
| 4 | Each value shown exactly once | P1 |
| 5 | No axis drawn for data that does not exist | P2 |
| 6 | Ship as v2.8.0 to exercise the v2.7.0 animated updater | P2 |

## Non-goals

- **No new visual language.** No new hero treatment, no bordered stat cards, no
  restyled chart. v2.6.0 shipped a Result redesign one day before this plan;
  another restyle without a complaint about *style* (as opposed to *spacing*)
  would be churn.
- No changes to metrics, formulas, or what is measured.
- No changes to History, Home, Settings, or the typing screen.
- No per-width-tier layouts (hero-beside-graph and similar). Capping is the
  smaller fix that addresses the actual complaint; tiering can follow if the
  capped version still reads as empty.
- No new dependencies.

## Key decisions

**Cap, don't stretch.** Past roughly 90 columns a fixed two-column layout stops
reading as one object — the eye travels too far between related values. Capping
content and centring the panel is why option A was chosen over per-tier layouts.

**Integer cell repetition, not interpolation, for the graph.** Smoothing between
samples would look better and would draw WPM values the test never produced. For
a typing tester whose entire value is honest measurement, that trade is not
available.

**De-duplicate toward the hero.** `raw` and `consistency` stay where the eye
lands and read as headline stats; they leave the grid, which drops from five
items to three.

**`resultMaxContentW = 88` is a starting value.** Phase 3 checks it against real
renders at four widths and the plan records the final number.

## Phases

| # | Phase | Status | Depends on |
|---|-------|--------|------------|
| 1 | [Width Contract And Baseline Snapshots](./phase-01-start.md) | Completed | — |
| 2 | [Graph Fills Its Width](./phase-02-graph-fills-its-width.md) | Completed | 1 |
| 3 | [Panel Cap, Centring, And Stats](./phase-03-panel-cap-center-and-stats.md) | Completed | 2 |
| 4 | [Verify, Docs, And Release](./phase-04-verify-docs-and-release.md) | In progress | 3 |

Phase 1 changes nothing visible — it establishes the shared width contract and
records baselines so phases 2 and 3 produce reviewable diffs instead of
unreadable churn. Phase 3 is the only phase the user sees.

## Constraints

- Protected `main`: branch → PR → green `ci.yml` → squash-merge.
- `NO_COLOR`/mono must stay layout-identical; only attributes may differ.
- The reveal animation's settled frame must remain byte-identical to the static
  render, and every mid-reveal frame must keep the settled line count and
  per-line width (`result_graph_test.go:109` already enforces this).
- The degraded gate below 60×20 is untouched.
- Colors come from `internal/theme` Roles only; no hex literals.
- Every Go file under 200 LOC.

## Success Criteria

- [ ] `go test ./... -race -count=1`, `go vet ./...`, empty `gofmt -l .`
- [ ] `make size-check` passes
- [ ] Panel width capped and centred, verified at 60/80/120/200
- [ ] A short run's chart fills its budget; a long run still downsamples
- [ ] Zero-error runs draw no right axis
- [ ] `raw` and `consistency` appear exactly once each
- [ ] `NO_COLOR` frames layout-identical to colored at every tested width
- [ ] Reveal invariants still hold, including for stretched charts
- [ ] Manual pass at 3 widths × 2 color modes, plus one long run
- [ ] v2.8.0 published and verified; animated updater observed on a real download

## Relationship to prior work

Follows the v2.6.0 Result redesign (`plans/260730-1442-result-ui-redesign/`),
which introduced the current hero, dual-axis graph, and stats grid. This plan
does not revisit those decisions; it fixes the width handling they were built
without.

Ships as v2.8.0, which is also the first release that can exercise the animated
updater from `plans/260731-2107-animated-update-cli/` — a self-updater always
runs the *old* binary's update code, so v2.7.0's animation was unobservable when
v2.7.0 was the version being installed.

## Decisions taken during implementation

- **`resultMaxContentW` stayed at 88.** Checked against renders at 60/80/120/200;
  the resulting 94-column panel reads as one object and leaves balanced margins.
- **One stats column, not two.** With three items a second column left `time`
  stranded beside two rows of whitespace. One aligned column also removed the
  proportional-gutter branch entirely.
- **Centring applies at every width, not only past the cap.** The plan assumed
  narrow terminals would not move. They do, by 4 columns — the 8-column margin
  always existed and was simply all on the right. Splitting it is more balanced
  and keeps one rule instead of two. A test asserting the original assumption
  failed and the assumption, not the code, was wrong.
- **Panel inset corrected from 4 to 6.** A latent off-by-two in the border
  arithmetic, exposed the moment the chart tried to use its full width.

## Open questions

None.
