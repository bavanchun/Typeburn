# Phase 6 — Result redesign

Status: DONE. Gate 0 **passed**; no dedicated Result height change needed.

## Gate 0

Re-measured on the branch base (`26702a87`), matching the numbers supplied:

```
BigDigits: (7)=8 (12)=13 (87)=17 (96)=18 (100)=24 (106)=24 (120)=23 (200)=28 (999)=26
layoutFor: 60→Inner 46 | 61→47 | 72→58 | 80→66 | 88→74 | 120→88 | 200→88 (capped)
old panel height: 28 rows at every width; old View frame: 29 lines
```

### Configuration tested

`NetWPM=100`, `Accuracy=100`, `updateHint = v2.9.0`, populated rail
(`PB 104 / avg10 81 / rank #2 of 6`), `IncorrectChars=8`, `ExtraChars=1`,
2 missed keys, 30 s of per-second samples with errors, `NO_COLOR` theme.

### Result

| size | panel rows | budget (h−4) | every line == PanelW | View frame |
|---|---|---|---|---|
| 80×24 | **20** | 20 | yes (72) | 24 lines, last row blank |
| 60×20 | **16** | 16 | yes (52) | 20 lines, last row blank |
| 120×24 | 20 | 20 | yes (94) | 24 |
| 200×50 | 20 | 46 | yes (94) | 50 |

The budget is `h−4`, not `h−2`: `View` spends a blank spacer row, the update-hint
row and the footer row, and the root writes the persistence notice onto the
terminal's **last** row — which it can only do without pushing the frame over if
that row is blank. The gate therefore asserts both `panelRows <= h-4` and
"last frame row is blank". Both are permanent tests
(`TestResultPanel_FitsTheHeightBudget`, plus the app case `result/notice+hint`
which stacks panel + hint + footer + notice for real).

Row budget by tier (fixed by construction, `resultLayout.ContentRows`):

```
normal (h>=24): 1 label + 6 hero + 1 chart hdr + 4 plot + 2 axes + 2 meta = 16 content
                +2 vpad +2 border = 20 panel rows
compact (h<24): 1 + 6 + 1 + 3 + 2 + 1 = 14 content, no vpad, +2 border = 16 panel rows
```

## Ladder rungs — which are reachable, and where

`railMinW` is **not** a constant. It is `railNaturalW(rows, short)`, measured from
the label/value strings about to be rendered. Five rungs, all reached by ordinary
runs (`TestResolveHeroZones_EveryRungIsReachable`):

| rung | arrangement | reached by |
|---|---|---|
| 0 `heroRungBigAcc` | both zones block art, gutter 6 | 87 wpm @ 96% on 120 cols (also 88 cols) |
| 1 `heroRungTightGutter` | both block art, gutter 4 | 87 wpm @ **100%** on 88 cols |
| 2 `heroRungTextAcc` | accuracy demoted to a text block | 106 wpm @ 97% on 80 cols |
| 3 `heroRungShortRail` | rail switches to `pb` / `vs pb` / `avg10` / `cons` | 106 wpm @ 97% on 72 cols |
| 4 `heroRungNoRail` | no rail column | 106 wpm @ 97% on 60 cols |

Deviation from the phase text, deliberate: the spec's rung 3 was "collapse the
rail to the two-column meta block". A two-column meta block costs vertical rows
the 60×20 budget does not have (measured: it needs 3 more rows than exist), so
rung 3 is instead an abbreviated-label rail, which costs nothing vertically and
keeps the comparison visible down to 72 columns. Rung 4 drops the rail entirely;
the run's standing still shows on the closing meta row (`#2 of 6 runs`).

A sixth, defensive step exists inside `heroWithoutRail`: a WPM too wide for block
art at any gutter falls back to text (`TestHeroZones_AbsurdValueFallsBackToText`,
reached at 6-digit values). It cannot wrap the border.

## Layout

```
row 0     wpm [★ new best]            acc                (zone labels)
rows 1-6  [big wpm]  g  [acc zone]  g  [rail, right-flushed]
row 7     wpm over time                    words 10 · english · 8s
rows 8-11 chart plot (3 rows when compact)
row 12    baseline
row 13    second ticks
row 14    most missed  e ×3  t ×2                    (dropped when compact)
row 15    220 correct · 8 wrong · 1 extra          #2 of 6 runs
```

- Zone widths are all derived at render time (`heroZonesFor` → `heroDemand`).
  There is no hardcoded digit width anywhere; `TestHeroZones_WidthsComeFromTheDigits`
  fails if one is introduced (verified by mutation).
- The rail renders at its **own** natural width inside the column it is given and
  is right-flushed, so extra columns become gutter rather than label-to-value
  distance. `railLabelValueSpan` is asserted `<= 20` cells; stretching the block
  to the column produces 42–46-cell gaps and fails the test (verified by mutation).
- `resultMaxContentW` stays 88; the cap comment records the measured reason.
- `220/8/1` is now `220 correct · 8 wrong · 1 extra`; the delta is `▲ 6 wpm` /
  `▼ 4 wpm` / `= 0 wpm` — a glyph, never a bare tinted number.
- First-run rail: `first run` / `no history yet` / blank / `raw` / `consistency` /
  blank. A run withheld from history (AFK-trimmed) keeps PB and avg10 — both true
  facts about the history — but its rank reads `not ranked`, because it took no
  place.
- Rank is bucket-scoped (same mode + length), per the recorded decision.

## Chart

- Y-axis fits the observed range ±10% (`wpmAxisRange`), floored at a span of 2 so
  a flat run still scales. The 8-second fixture went from `0…95` (curve in the
  top two rows) to `56…98` (curve across the whole plot).
- `errAxisCeiling` lifts the error scale to at least 4. This fixes both defects at
  once: a lone error no longer sits at `errors/maxErr == 1` on the top row, and
  the axis ticks stop being non-monotonic (`maxErr=1 → 1/0/0`).
- Plot height is 4 rows (3 when compact), down from 5, to buy the meta rows.

## Fill ratio

Measured at 120 columns over the panel's inner area (88 cells wide), counting
non-space cells, ANSI stripped, using the `shortRunResult` fixture:

| region | before | after |
|---|---|---|
| chart rows excluded | 208 / 1496 = **13.9%** | 331 / 968 = **34.2%** |
| whole inner area | 404 / 2288 = **17.7%** | 547 / 1584 = **34.5%** |
| chart excluded, rail populated | — | 367 / 968 = **37.9%** |

(The audit's 22.6% used a different row window; both before/after here are
measured the same way, so the ratio is the comparable number: density roughly
2.5×, on a panel that is also 8 rows shorter.)

## Golden diff

Regenerated once, at the end, all eight files (`-update`), and read. Every change
is explained by the redesign:

- panel height 28 → 20 rows at every width;
- hero label row moved above the digits, `wpm` label no longer under them;
- comparison rail added on the digit rows (first-run form for this fixture);
- `raw 106 wpm   consistency 59%` moved out of a secondary card row into the rail;
- `test type` / `characters` / `time` grid replaced by the mode line on the chart
  header and the labelled character row at the bottom;
- `52/0/0` → `52 correct · 0 wrong · 0 extra`;
- closing row gained `first run` at the right edge;
- chart y-axis labels `95/48/0` → `98/78/56` and the plot is 4 rows;
- 60-column panel lost its vertical padding (compact tier).

No unexplained change. `TestRenderHarness_ReproducesRecordedBaseline` still gates
the harness against the recorded bytes.

## Reveal

`TestResultReveal_ExhaustiveSweepHoldsGeometry`: 7 configurations × 5 sizes ×
17 ms steps from t=0 to t=1200 ms (past the celebration window) = ~2 500 frames.
Every frame keeps the settled line count and per-line rune width; the settled
frame is byte-identical to the static frame at every size. Zero failures.

Configurations: two-digit first run, three-digit at 100% ranked, new best,
letter-strict, no per-second samples, withheld-from-history, long run
(downsampled chart). Sizes: 60×20, 72×24, 80×24, 120×24, 200×50.

Two invariants that per-line width alone cannot see, added as their own tests:

- `TestResultReveal_ZonesDoNotMoveDuringTheCountUp` — the band's columns are
  resolved from the final values, so the accuracy zone and the rail do not slide
  while the number climbs.
- `TestResultReveal_CountUpKeepsTheDigitsAnchored` — the counting digits stay
  right-aligned in their zone.

A mid-reveal case (`result/mid-reveal`, plus `result/ranked`) was added to
`screenCases`, so every frame-fits and theme-independence assertion now covers
the count-up geometry as well as the settled one. Ordering note: the case was
added after the layout change, not before it as the phase text asked — the
coverage is identical, but the sequencing claim in the phase file was not met.

## Mutations run (falsifiability)

| # | mutation | caught by | outcome |
|---|---|---|---|
| 1 | `errAxisCeiling` floor 4 → 1 | 3 graph tests | FAIL as intended |
| 2 | hardcode `WPMW: 17` in `heroZonesFor` | 6 tests incl. derived-width + goldens | FAIL |
| 3 | `spacer -= 1` | nothing | **survived** — `lipgloss.Place` re-pads, so the blank row is structural |
| 3b | chart rows 4→7 / 3→6 (panel fills the terminal) | height budget + celebration-row test | FAIL |
| 4 | rail stretched to its column instead of natural width | label/value span test | FAIL (42–46-cell gaps) |
| 5 | `BigDigitsFixed` → `BigDigits` during count-up | anchored-digits test | FAIL |
| 6 | `revealLine` returns `""` instead of spaces at p=0 | nothing | **survived** — `padCell` re-pads every row |
| 7 | zone widths taken from the count-up value | zones-do-not-move test | FAIL |

Mutations 3 and 6 surviving is a property of the design, not a gap: every row is
emitted through `padCell` at an exact width and the frame is always shorter than
the terminal, so those two failure modes cannot occur. 3b is the mutation that
actually removes the slack, and it fails.

## Celebration

**Kept, with the row made explicit.** It is not dead: `View` always leaves
`spacer >= 1`, and the frame is always shorter than the terminal, so
`lipgloss.Place` adds padding — a blank band adjacent to the panel exists at
every supported size. `TestResultView_KeepsARowForTheCelebration` now asserts
that directly (`blankBand` non-empty, and a mid-burst frame differs from its
settled self) across 7 sizes × 2 run shapes. Mutation 3b proves the assertion
bites the moment the budget stops leaving the row.

## Files

Created: `internal/ui/result_comparison_rail.go`, `internal/ui/result_context.go`
(+ their `_test.go`), `internal/ui/result_hero_zones_test.go`.
Deleted: `internal/ui/stat_card.go` (`StatCard` had no remaining caller once
`renderStatsGrid` went).
Modified as scoped, plus two outside the phase's list:

- `internal/ui/result_render_helpers.go` — took `padCell` / `cutCells` / `maxInt`
  so `screen_result_hero.go` stays under 200 LOC. No behaviour change.
- `internal/ui/render_harness_cases_test.go` / `internal/app/frame_fits_cases_test.go`
  — the new harness cases. Neither is owned by a concurrent phase.

`knownOverflow`: all 14 Result entries deleted from `internal/ui`. In
`internal/app`, all `result@*` (14) and `transition/early@*` (14) entries deleted
and the 6 narrow `result/persist-notice@*` entries corrected from
`{Lines:29, Width:78}` to `{Lines:0, Width:78}` — the height debt is gone, the
notice's unbounded width belongs to another phase and was left alone. Nothing
else in either map was touched.

Every non-test Go file is under 200 LOC (largest: `screen_result_view.go` at 197).
Test files exceed it, matching existing repo practice.

## Gate

```
gofmt -l .            empty
go vet ./...          clean
go test ./... -race -count=1   all packages ok
make lint             ok
make build            ok
```

## CHANGELOG copy

```
### Changed
- The result screen is rebuilt around a comparison rail. Beside the big WPM
  number it now answers "was that any good?" from history already in memory:
  personal best, how this run compares to it, your average over the last ten runs
  in the same mode and length, raw speed, consistency, and where the run ranks.
  A first run says so instead of showing blanks.
- The result panel is 8 rows shorter and fits an 80x24 terminal with the update
  hint and a save notice on screen at the same time. It degrades to 60x20.
- The WPM chart's vertical axis now fits the run's own range instead of starting
  at zero, so the curve uses the whole plot rather than the top two rows.
- A single mistyped second no longer draws its error marker at the top of the
  chart, where it read as a burst of speed. The error axis labels no longer count
  backwards.
- The character counts are spelled out - "220 correct - 8 wrong - 1 extra" - and
  the comparison against your best uses an up or down arrow. Nothing on the
  result screen depends on colour alone any more, so NO_COLOR and the mono theme
  lose no information.
```

## Unresolved questions

1. **Accuracy's typographic weight.** Gate 0 proves two block-art zones and a
   comparison rail cannot coexist below 88 inner columns, so on an 80-column
   terminal with history, accuracy renders as a bold coloured text block rather
   than block art. The requirement "accuracy carries real typographic weight" is
   met at 88+ columns and at 80 for a first run; below that it is weight-by-
   emphasis, not by size. Accept, or spend the rail at 80 columns?
2. **Leftover slack placement.** When the rail's natural width is much smaller
   than its column (e.g. the first-run rail at 120 columns), the slack currently
   sits between the accuracy zone and the rail. Pushing it into the gutters
   instead would centre the composition. Deliberately not changed this late.
3. **Compact tier drops the missed-key line** (h < 24). The character breakdown
   and the run's standing were judged more valuable. Confirm.
4. `docs/` is untouched — Phase 9 owns documentation. The Result sections of
   `codebase-summary.md` and `system-architecture.md` now describe the old
   layout.
