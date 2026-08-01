---
phase: 3
title: "Panel Cap, Centring, And Stats"
status: pending
priority: P1
effort: "4h"
dependencies: [2]
---

# Phase 3: Panel Cap, Centring, And Stats

## Overview

Turn on the width cap from Phase 1, centre the panel, bound the stats gutter,
and delete the duplicated `raw` / `consistency` pair. This is the phase the user
actually sees.

## Requirements

- Functional
  - Panel content is capped at `resultMaxContentW` and the panel is horizontally
    centred in the terminal.
  - The stats grid's two columns sit a readable distance apart at any width.
  - `raw` and `consistency` each appear exactly once on the screen.
  - Below the two-column threshold the grid still stacks vertically.
- Non-functional
  - `NO_COLOR`/mono layout-identical; only attributes differ.
  - The reveal animation still settles byte-identical to the static render.
  - The degraded gate below 60×20 is untouched.

## Architecture

### Cap and centre

Flip `layoutFor` to the real policy: `InnerW = min(termW - 12, resultMaxContentW)`,
with `LeftPad = (termW - PanelW) / 2` so the panel sits in the middle. Below the
cap nothing changes — narrow terminals behave exactly as they do today, which is
what keeps this low-risk.

`renderHero` currently begins `_ = innerW` (`screen_result_hero.go:22`). With a
capped panel the hero no longer needs to fight for space, but the parameter
should either be used or removed; leaving a discarded parameter is what let the
hero drift out of the layout system in the first place.

### The duplicate

`raw` and `consistency` are rendered twice on the same screen:

| Value | First render | Second render |
|---|---|---|
| `raw` | `screen_result_hero.go:68` (secondary card) | `screen_result_view.go:145` (stats grid) |
| `consistency` | `screen_result_hero.go:72` (secondary card) | `screen_result_view.go:149` (stats grid) |

Keep the hero cards and drop them from the grid. The hero is where the eye lands
and where they read as headline stats; in the grid they are just two more rows.
That leaves the grid holding `test type`, `characters`, and `time` — three
items, which is a much better fit for the space than five.

**This is a user-visible content change, not just a layout one.** No information
is lost; it is shown once instead of twice.

### The gutter

`colW := innerW / 2` (`screen_result_view.go:160`) is why the right column is 94
characters away at 200 columns. With the cap in place `innerW` never exceeds 88,
so this is already largely fixed — but the division should still be replaced by
an explicit bounded gutter so the layout does not silently regress if the cap is
ever raised.

With the grid down to three items, re-evaluate whether two columns earn their
keep at all, or whether one column of three rows reads better. Decide from the
rendered output in step 4, not from this document.

## Related Code Files

- Modify: `internal/ui/result_layout.go` (real cap + centring)
- Modify: `internal/ui/screen_result_view.go` (apply `LeftPad`, rebuild stats grid)
- Modify: `internal/ui/screen_result_hero.go` (use or drop `innerW`)
- Modify: `internal/ui/screen_result_test.go`, `internal/ui/testdata/result_baseline_*.txt`

## Implementation Steps

1. Implement the real `layoutFor`. Re-record the baselines; read the 200-column
   diff before accepting it.
2. Apply `LeftPad` in `renderPanel`. Confirm the footer and the update hint,
   which render outside the panel, still line up.
3. Remove `raw` and `consistency` from the stats grid; keep them in the hero.
4. Replace `innerW / 2` with a bounded gutter. Render at 80/120/200 and pick
   one- or two-column for the remaining three items based on what is actually
   legible.
5. Resolve `_ = innerW` in the hero.
6. Tests:
   - panel width never exceeds the cap for terminal widths up to 400
   - panel is centred: left margin within 1 column of right margin
   - `raw` and `consistency` each appear exactly once in the settled frame
   - stacked layout still used below the two-column threshold
   - `NO_COLOR` frame line-count and per-line width identical to colored
   - settled reveal frame byte-identical to the static render (existing guard)

## Success Criteria

- [ ] `go test ./... -race -count=1` green
- [ ] Panel capped and centred; verified at 60/80/120/200
- [ ] `raw` and `consistency` appear exactly once each
- [ ] `NO_COLOR` layout-identical
- [ ] Baseline diffs reviewed by a human, not auto-accepted
- [ ] Every touched file under 200 LOC

## Risk Assessment

**Risk:** centring shifts the panel but the footer and update hint render
outside it and stay left-aligned, so the screen looks misaligned rather than
centred.
**Mitigation:** step 2 checks them explicitly; they are in
`screen_result_view.go` above `renderPanel` and easy to miss.

**Risk:** dropping grid rows is a content decision someone later reads as an
accidental deletion.
**Mitigation:** recorded here and in the CHANGELOG as intentional
de-duplication, with the two source locations named.

**Risk:** `resultMaxContentW = 88` is a guess.
**Mitigation:** step 4 checks it against real renders at four widths; the plan
records the final value and why.
