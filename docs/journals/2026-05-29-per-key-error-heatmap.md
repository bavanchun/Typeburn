# Per-Key Error Heatmap

**Date:** 2026-05-29
**Branch:** `feat/key-error-heatmap` → PR #31
**Plan:** `plans/20260529-key-error-heatmap/`

## What shipped

After each test, Typeburn now surfaces the keys the user fumbled most — a
post-hoc per-key miss tally. Pure replay over the existing keystroke log; nothing
persisted, no schema change. Rides the ephemeral `metrics.Result`.

Built TDD across four phases:

1. **Metrics** — `KeyHeatmap(log) []KeyMiss` + `Result.KeyMisses`, populated in
   `Compute`. A miss = every wrong forward keystroke (`Typed!=0`) vs a real
   target (`Target!=0`) where `!Correct` — corrected fumbles still count.
   Case-folded (`unicode.ToLower`), keys with ≥1 miss only, sorted
   misses↓ → attempts↓ → key↑, capped to top 8. Single pass, timing-independent.
2. **Result screen** — `renderKeyHeatmap` adds a `most missed:  e ×4   t ×3 …`
   line between char-stats and meta. Theme-roles only (NO_COLOR/mono layout
   identical), width-capped to the panel; faint `no missed keys` on clean runs.
3. **CLI** — additive `key_misses` JSON array (empty → `[]`, not `null`) and
   top-5 `most_missed_*` table rows. Single mapping layer (`newMetricOutput` /
   `metricTableRows`), so both `run --no-tui` and `replay` inherit it. Existing
   JSON keys unchanged.
4. **Docs + gate** — `codebase-summary.md` + `project-roadmap.md`; full CI gate
   green (`go test ./... -race`, `go vet`, `gofmt -l`, `make size-check`).

## Key decision: honor the locked plan over weakening docs

The mandatory code review caught a real contract drift. The brainstorm had
**locked "Top N = 8"**, and phase-03 documented JSON as "the full (top-8) list".
But the first implementation left `KeyHeatmap` *uncapped* — the top-8 bound lived
only in the UI layer (`heatmapMaxEntries`). So JSON `key_misses` was actually
unbounded (a 200-distinct-key Code-mode paste would emit 200 entries).

Two ways to resolve: (a) weaken the docs to "uncapped", or (b) move the cap into
`KeyHeatmap` so every surface inherits it. Per the project rule that *verified/
locked user decisions are sticky*, the cap was the user's locked design choice —
so the fix aligned the code to the plan, not the other way around. The cap now
lives in `KeyHeatmap`; the UI `heatmapMaxEntries` became defensive belt-and-
suspenders. Added a `TestKeyHeatmap_CapsAtTopN` to lock it.

## Notes

- All new files <200 LOC; pure-logic packages (`metrics`, `cli`) keep zero UI deps.
- Test robustness lesson: asserting `×4` against the raw styled Result view failed
  because lipgloss splits a single `Render` with ANSI resets between runes — the
  view-level substring checks had to run through the package's `stripANSI` helper.
- Deferred (future): lifetime/aggregate heatmap across history, per-key error-*rate*
  coloring, finger/row grouping, configurable N, ASCII space-glyph fallback.
