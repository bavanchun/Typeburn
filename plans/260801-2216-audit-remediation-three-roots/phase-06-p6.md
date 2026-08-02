---
phase: 6
title: "Result A2 Redesign"
status: pending
priority: P2
effort: "2d"
dependencies: [1, 3]
---

# Phase 6: Result A2 Redesign

## Overview

Replace the Result layout with a three-zone hero band, a comparison rail, and a
full-width chart. Fills the measured dead space with information the user wants,
and fits 80×24 with the footer visible.

**This phase opens with a falsification gate.** The A2 geometry as originally
specified is measurably wrong, and the gate exists to catch that before any code
is written.

## Requirements

- Functional
  - Result fits 80×24 **with the update hint and a notice present**; degrades to 60×20.
  - Accuracy carries real typographic weight.
  - The rail answers "is this run good?" from history already in memory.
  - Chart y-axis fits the data range.
  - No information carried by colour alone.
- Non-functional
  - `NO_COLOR`/mono layout-identical; reveal invariants hold; files under 200 LOC.

## Architecture

### Gate 0 — falsify the geometry before writing code

The specified `[wpm 17] 6 [acc 19] 6 [rail 40] = 88` was derived from a 2-digit
sample. Measured `BigDigits` widths, ANSI stripped:

```
BigDigits(87)=17   (96)=19   (100)=22   (120)=22   (200)=26
```

A 100 wpm run needs 22; **100% accuracy — routine on a `words 10` run** — needs
22 plus the `%`, about 24. Together: `22 + 6 + 24 + 6 + 40 = 98 > 88`. And
`TestLayoutFor_MatchesRenderedWidth` (`result_layout_test.go:135`) asserts every
panel line equals `lay.PanelW` exactly, so this is a hard test failure.

#### Gate 0 pre-measurement — re-measured after the glyph fix

The widths above predate Phase 1's glyph-table correction and are now **too
small**. Re-measured against `lipgloss.Width(BigDigits(n, th))` on current `main`:

```
BigDigits(87)=17  (96)=18  (100)=24  (106)=24  (120)=23  (200)=28
layoutFor(60)  InnerW=46      layoutFor(80)  InnerW=66
layoutFor(88)  InnerW=74      layoutFor(120) InnerW=88 (capped, =200)
```

So the worst realistic case is `100 wpm @ 100%`: `wpmW=24`, `accW=24+2=26`.

| width | InnerW | rail left after `24 + g + 26 + g` |
|---|---|---|
| 120/200 | 88 | g=6 → **26**; g=4 → **30** |
| 88 | 74 | g=6 → **12** |
| 80 | 66 | g=6 → **4** |
| 60 | 46 | negative |

Against `railMinW = 40`, **rung 1 (gutters→4) is never sufficient at any width.**
Rung 2 (accuracy demoted to text, ~8 cells) yields 48 at InnerW=88 — the first
rung that clears 40 — and only 26 at InnerW=66, so 80 columns lands on rung 3.

**Consequence for implementation:** the fallback ladder is the *normal* path for
any three-digit WPM or 100% accuracy, not an edge case. Two big-digit zones do not
coexist with a 40-cell rail at any supported width. Design the ladder as the
primary layout decision, and treat "both zones big" as the exception that only a
two-digit/two-digit run reaches. The success criterion "every fallback rung has a
test that reaches it" is satisfied by ordinary runs, not contrived ones.

**This is the same methodological error that produced the glyph bug** — the
v2.8.0 screenshot showed 74 wpm, so the zero defect looked fine. Do not repeat it
a third time.

Two rows are also missing from the 24-row budget:
- `screen_result_view.go:36-43` already reserves an `updateLine` row when an
  update is available — exactly the users the update feature targets.
- Phase 3 makes a persistence/AFK notice load-bearing.

**Gate (≈1h, before any implementation):** hand-render the layout at 80×24 and
60×20 with `NetWPM=100`, `Accuracy=100`, `updateHint != nil`, and a notice
present. Assert `panelLines <= 22` and every line width `== layoutFor(80).PanelW`.
If it fails, resolve via the ladder below before writing code. If it cannot be
resolved, Result needs a dedicated height change and D2 is not subsumed — say so
rather than shipping a broken budget.

### Zone widths are derived, never hardcoded

```go
// Zone widths come from the digits actually being rendered. Hardcoding them
// breaks the moment a run reaches 100 wpm or 100% accuracy, both routine.
wpmW := lipgloss.Width(BigDigits(wpm, th))
accW := lipgloss.Width(BigDigits(acc, th)) + 2 // trailing " %"
railW := innerW - wpmW - accW - 2*gutter
```

Fallback ladder when `railW < railMinW`, applied in order:
1. shrink gutters to 4
2. drop accuracy to text form (`acc 100%`), keeping WPM big
3. collapse the rail to the two-column meta block already designed for 80 cols

Each rung must be exercised by a test, not assumed reachable.

### The measured problem

Inside the 88-column inner area at 120 cols, excluding the graph: **259 of 1144
cells inked — 22.6% fill**. The unused cells form **one contiguous rectangle from
col ~30 to col 88 across 13 rows**; the eye reads that as an unfinished region,
not whitespace.

- **Optical centre is ~11 columns left of geometric centre** — ink centroid ≈ col
  33 of 88, border axis col 44. **v2.8.0 centred the box and left the ink where
  it was**, which is why capping felt tidier without fixing anything.
- **The chart is 60% empty vertically too:** y-axis `0…max` while data lives in
  55–85, so 3 of 5 plot rows carry no curve.
- **Hierarchy collapsed to two tiers** — `acc 96%` reads identically to `time 30s`.
- **Nothing says whether 87 wpm is good.** History is loaded at
  `model_history.go:46` at the moment the ResultModel is built and used for one
  boolean. PB, rank and last-10 average are in hand and discarded. Better
  whitespace alone just centres the void.

### Layout

`[big wpm] gutter [big acc] gutter [right-flush rail]`, chart full-width beneath,
closing meta line with ink at both edges.

```
╭────────────────────────────── result ──────────────────────────────╮
│  wpm                acc                     time 30 · english · 30s │
│   <big digits>       <big digits> %   personal best         91 wpm  │
│                                       this run             ▼ 4 wpm  │
│                                       avg last 10           81 wpm  │
│                                       raw                   92 wpm  │
│                                       consistency              78%  │
│                                       characters      220 / 8 / 1   │
│  wpm over time                                                      │
│  … chart, full width …                                              │
│  most missed  e ×3  t ×2                            #2 of 47 runs   │
╰─────────────────────────────────────────────────────────────────────╯
```

**Do not widen past 88 at 200 columns.** The stretched variant (inner 116) puts
`personal best` and `91 wpm` 50 columns apart and the pairing dies. The cap stays;
the emptiness is *inside* the panel and the outer margin is legitimate
max-measure whitespace.

### Chart

- Y-axis fits the observed range ±10% instead of `0…max`.
- **Error `x` marker** (`result_graph.go:97`): when one second holds the run's
  max, `errors/maxErr == 1` pins the marker to row 0 — it floats at 64 wpm on a
  run that never dipped there, reading as a WPM spike. Fix with a nice-number
  ceiling (`maxErr = max(maxErr, 4)`).
- **`errAxisLabel`** (`result_graph_axes.go:65`): integer truncation makes the
  axis **non-monotonic** — `maxErr=1 → 1/0/0`, `maxErr=3 → 3/1/0`. Subsumed by
  the ceiling; otherwise suppress the mid tick below `maxErr < 2`.

### Colour-only encoding

`220/8/1` (`screen_result_view.go:134-140`) carries the middle number's meaning
**entirely** in `RoleError`. Under `mono` that is `#FFFFFF` vs primary `#F2F2F2`;
under `NO_COLOR` it is gone. Label it: `220 correct · 8 wrong · 1 extra`. The
delta must be a glyph — `▼ 4 wpm` / `▲ 6 wpm`, never a bare tinted number.

### Data dependency

Pass a pre-computed value struct, not `[]storage.Record`:

```go
type resultContext struct {
    PB, Avg10   float64
    Rank, Total int
    HasHistory  bool
}
```

`BestWPMPerBucket` / `BestBucketKey` / `EffectiveWPM` already exist, so PB is a map
lookup; only avg-last-10 and rank need a ~25 LOC pure helper. Built in
`internal/app` — **Phase 3 owns `model_history.go`, hence this phase's dependency
on 3.**

`NewResult(msg, th, km)` (`screen_result.go:41`) is the only constructor, so
`screen_result.go` gains the field and a `WithContext` setter. Red-team flagged
this file as missing from the ownership table; it is listed now.

**First-run state is required, not a nicety** — with no history the rail reads
`first run · no history yet` and `raw`/`consistency`/`characters` promote up. It is
what a brand-new user sees.

### Celebration interaction

`celebration.go:42-45` bails when no row is entirely blank. A2 targets 22 panel
rows, making an all-blank row *less* likely than today — so this phase makes an
already-dead feature deader. Either give the celebration an explicit reserved row
in the A2 budget, or delete the feature. **Do not leave it silently dead.**

### Mid-reveal geometry is unmeasured

Phase 1's harness settles the reveal (`revealStartMs = 0`, `nowMs = 1<<40`), so
`revealDone` is true by construction in every frame-fits cell. During the
count-up the hero is narrower and `sparkVisibleBars` is partial — **an entire
geometry no assertion covers.** A2 positions the rail by column arithmetic
against the hero, so this is the phase where it starts to matter. Add a
mid-reveal case to `screenCases` before changing the layout, not after.

## Related Code Files

- Create: `internal/ui/result_comparison_rail.go`, `internal/ui/result_context.go`
- Modify: `screen_result_hero.go`, `screen_result_view.go`, `screen_result.go`, `screen_result_reveal.go`, `result_layout.go`, `result_graph.go`, `result_graph_axes.go`, `stat_card.go` (near-dead once `renderStatsGrid` goes), `celebration.go`
- Modify: `internal/app/model_history.go` (build `resultContext`) — **after Phase 3**
- Regenerate: `internal/ui/testdata/result_baseline_*.txt`
- Modify: `screen_result_test.go`, `result_layout_test.go`, `nocolor_layout_invariant_test.go`, `phase09_polish_test.go`, `screen_result_heatmap_test.go`, `frame_fits_test.go`

## Implementation Steps

1. **Gate 0** — the falsification render above. Do not proceed until it passes or
   the ladder resolves it.
2. Rebase on Phase 3; confirm `model_history.go` is at its post-Phase-3 state.
3. `result_layout.go`: derived zone widths + `Compact` (h≤24) + the fallback
   ladder. Unit-test the arithmetic at 60/80/120/200 × {2-digit, 3-digit, 100%}
   before rendering anything.
4. `resultContext` + the pure avg/rank helper, tested including the empty case.
5. Three-zone hero. Verify against Phase 1's harness at every size.
6. Rail with the first-run variant.
7. Chart: y-axis fit, nice-number ceiling, axis-label fix.
8. Label the char triple; glyph delta.
9. Resolve the celebration: reserved row or removal.
10. Delete `renderStatsGrid`; delete the Result `knownOverflow` entries.
11. Regenerate goldens and **read the diff**.
12. Re-verify reveal invariants exhaustively — highest-risk surface in the phase.

## Success Criteria

- [ ] Gate 0 passed, with the tested configuration recorded in the PR body
- [ ] 80×24 renders ≤24 rows **with update hint + notice present**; 60×20 ≤20
- [ ] Zone widths derived at render time; no hardcoded digit width anywhere
- [ ] Every fallback rung has a test that reaches it
- [ ] Result `knownOverflow` entries deleted
- [ ] Fill ratio measured and materially above 22.6% — record the number
- [ ] Rail shows PB / delta / avg10 / rank; first-run variant correct with empty history
- [ ] Chart y-axis fits data; single-error run does not pin `x`; error axis monotonic
- [ ] `220/8/1` labelled; delta uses `▼`/`▲`
- [ ] `NO_COLOR`/mono layout-identical at 60/80/120/200
- [ ] Reveal: every mid-reveal frame keeps settled line count and per-line width; fully revealed byte-identical to static
- [ ] Celebration either works or is removed — not silently dead
- [ ] Every touched file under 200 LOC

## Risk Assessment

**Risk:** the reveal animation breaks — the frame shape changed and `resultCards`
goes 3→2. Most likely regression in the plan.
**Mitigation:** step 12 re-runs the exhaustive sweep (17 ms steps × 7
configurations × 5 widths) the audit used to prove it airtight. Any single
failing frame blocks.

**Risk:** the 24-row budget is wrong and is discovered late.
**Mitigation:** Gate 0 exists precisely for this and runs before code.

**Risk:** regenerating goldens alongside a large layout change makes the diff
unreviewable.
**Mitigation:** goldens regenerate once, at step 11, after all layout work; the
frame-fits harness independently asserts structure, so goldens are not the only
guard. Note Phase 1 changed the fixture WPM so the goldens can now actually see
digit defects.

**Risk:** 88 gets "improved" wider later, breaking label/value pairing.
**Mitigation:** the cap comment records the measured reason (50-column separation
at inner 116) and a test asserts the rail's label and value stay within a
readable span.

## Rollback

Largest-blast-radius phase. Revert is clean **only if Phase 3 stays** — this phase
edits `model_history.go` on top of Phase 3's edits. Revert this phase before
Phase 3, never after. Goldens revert with the commit.
