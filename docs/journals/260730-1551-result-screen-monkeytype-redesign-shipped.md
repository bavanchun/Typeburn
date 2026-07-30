# Result Screen Monkeytype-Style Redesign — Shipped

**Date:** 2026-07-30
**Branch:** `feat/result-ui-redesign` → PR [#61](https://github.com/bavanchun/Typeburn/pull/61)
**Plan:** `plans/260730-1442-result-ui-redesign/` (4 phases, all completed)

## What changed

Rebuilt the post-test Result screen to Monkeytype fidelity, keeping Typeburn's
heatmap and animation/NO_COLOR invariants:

1. **Hero** — accuracy promoted to a second big number beside the ASCII
   big-digit WPM; `raw` + `consistency` demoted to a secondary card row.
   Count-up reveal, stagger fade, `★ new best` unchanged.
2. **Graph** — new `result_graph.go` + `result_graph_axes.go`: braille
   sub-cell WPM line (RoleAccent, left Y) + red `x` error markers (RoleError,
   right Y). First consumer of `PerSecond.Errors` (computed since v1, unused
   until now). Draw-in reveal reuses `sparkVisibleBars`.
3. **Stats grid** — 2-col (test type / raw / characters | consistency / time),
   stacks below 60 inner columns. Replaced the old char-stats row + meta line.
4. **Deleted** `sparkline.go` (0 external callers); History keeps its own
   `sparklineInline` with `sparkBars` moved there.

## Decisions & why

- **Braille line over block bars:** true continuous line, closest to the
  Monkeytype reference; 2×4 sub-cell resolution is adequate at chartH=5.
- **`x` replaces braille cell at error columns:** keeps the two series
  unambiguous without color — critical for mono/NO_COLOR.
- **Downsampling added beyond the plan letter:** the plan's one-cell-per-second
  mapping would overflow the panel on 120 s tests; equal buckets (mean WPM,
  summed errors) keep any run inside the panel. Width-sweep tested.
- **No 4th `missed` char stat, no session timestamp:** per brainstorm
  decisions (YAGNI; field was always 0 / not stored).

## Difficulties

- **Grid mid-entry wrap (caught in review):** the 2-col threshold at
  `innerW >= 56` gave the left lipgloss column 28–29 cols while the longest
  line ("test type  words 100 · english") is 30 chars — `english` wrapped
  onto its own row at terminal widths 68–71. Only surfaced by probing widths
  between the tested 60 and 80. Fixed by raising the stack threshold to 60
  inner cols + a 36–100 width-sweep regression test. Lesson: test the widths
  *between* the canonical sizes, not just the canonical sizes.
- **Shared helpers vs deletion ordering:** `minMax`/`xAxisLabels` had to move
  to `result_graph.go` in Phase 1 while `sparkline.go` still compiled, then
  the deletion in Phase 3 flushed out `sparkBars`' hidden second consumer
  (History) — resolved by relocating the var, not resurrecting the file.

## Verification

`go test ./... -race -count=1` green (all packages), `go vet` clean,
`gofmt -l` empty, `make size-check` pass. CI green on ubuntu + macos across
both pushes. NO_COLOR invariant + reveal byte-stability covered by 14 new
graph tests, hero layout test, grid tests.
