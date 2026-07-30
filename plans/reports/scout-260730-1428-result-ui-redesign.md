# Scout Report — Result-Screen UI Redesign (Monkeytype-style)

## Project State (snapshot 2026-07-30)

- **Release:** `v2.5.1` (2026-07-11) = current public stable. Roadmap marks it SHIPPED; no in-flight release.
- **Health:** `go test ./...` GREEN (all 18 pkgs). `gofmt -l .` empty. `go vet ./...` clean.
- **Git:** `main` clean. **Only 1 untracked file** — `docs/journals/260702-1857-punctuation-numbers-toggle-feature-shipped.md` (journal for already-shipped PR #57 / v2.5.0). Not committed because `main` is protected (PR-only). It is finished docs of finished work, not dangling code.
- **No stale/old docs found.** Roadmap, wireframe, codebase-summary all at v2.5.1 truth. Backlog = additive/cosmetic only; no active plan for a Result redesign.

## Current Result Screen (`internal/ui/screen_result_view.go:85` renderPanel)

Renders inside one rounded-border `result` panel, vertical stack:

1. **Hero** (`screen_result_hero.go`): big-digit WPM (ASCII art `ascii_big_digits.go`) + right column of 3 stat cards (`acc`, `raw`, `consistency`). Count-up reveal + stagger fade.
2. **Sparkline** (`sparkline.go`): single-series **bar** chart "wpm over time" using `PerSecond.RawWPM` only. 3 rows, y-ticks + x-axis seconds. Draw-in reveal.
3. **Char stats**: `correct N  incorrect N  extra N` (single row).
4. **Key heatmap** (`renderKeyHeatmap`): "most missed" top-8 (Typeburn extra; Monkeytype lacks).
5. **Meta**: `30s · time 30 · english`.

## Target (image — Monkeytype result)

1. **Top:** TWO big numbers side-by-side — `wpm 66` (left) + `acc 92%` (right), both large/accent.
2. **Graph:** dual-axis **line** chart — WPM-over-time line (left Y-axis) + Errors-over-time line w/ red `x` markers (right Y-axis); X-axis = seconds.
3. **Stats grid (2 cols):** left = `test type` (time 15 / english), `raw 70`, `characters 82/4/1/0`; right = `consistency 55%`, `time 15s` + `00:00:15 session`.

## Gap Analysis (current → target)

| Area | Need | Data ready? | Work |
|---|---|---|---|
| Hero | Promote `acc` to 2nd big-digit block beside WPM | ✅ `res.Accuracy` / `KeystrokeAccuracy` | Rework `renderHero` layout (2 big blocks, drop acc from stat-card col) |
| Graph | Replace single bar sparkline → dual-axis line (WPM + Errors + red X markers, 2 Y-axes) | ✅ **`PerSecond.Errors` already computed but UNUSED today** — `internal/metrics/per_second.go:12` | **Biggest new component.** New renderer (line, not bar). Dual Y-axis scale (WPM 0..max, Errors 0..max). |
| Char stats | Keep, maybe add 4th `missed` (image shows `82/4/1/0`) | ⚠️ `MissedChars` removed in v1.0.1 (m4); only 3-tuple exists | Decision: keep 3-tuple or re-introduce missed. Recommend keep 3 (YAGNI; field was always 0). |
| Stats grid | Re-arrange into 2-col (test type / raw / characters \| consistency / time / session) | ✅ all present except `session` timestamp (not stored in `metrics.Result`) | Layout re-arrange + optional session-time field |
| Heatmap | Keep as-is (Typeburn extra) | ✅ | None |

## Files Involved (likely touch)

- `internal/ui/screen_result_view.go` (158 LOC) — panel assembly, section order
- `internal/ui/screen_result_hero.go` (64 LOC) — hero 2-big-number layout
- `internal/ui/sparkline.go` (173 LOC, near 200 cap) — **replace or add sibling** for dual-axis line chart. New file e.g. `result_graph.go` advised (<200 LOC rule).
- `internal/ui/screen_result_reveal.go` — reveal animation must stay byte-identical when settled; new graph needs reveal-compatible path
- `internal/ui/result_render_helpers.go` — shared render helpers
- Golden tests: `screen_result_test.go`, `screen_result_reveal_test.go`, `screen_result_heatmap_test.go` — **will need re-goldening**
- `docs/wireframe/mockups.md` §3 — update mockup to new design
- `docs/project-roadmap.md` — add release entry

## Constraints / Risks

- **NO_COLOR + mono invariants:** `nocolor_layout_invariant_test.go` asserts layout byte-identical under attribute-only render. New graph must be role-based (`theme.RoleAccent` etc.), never literal hex.
- **<200 LOC/file:** sparkline.go at 173 → new chart logic goes in a new file.
- **Reveal animation:** every frame layout-identical; settled frame byte-identical to static. New line-chart must honor this (animate draw-in like current `sparklineVisible`).
- **teatest goldens** will break → intentional re-goldening, not test weakening.
- **`min terminal 60×20`:** dual-axis chart must degrade gracefully on narrow widths (existing `width_tier.go`).

## Unresolved Questions

1. **`missed` 4th char-stat:** re-introduce (match Monkeytype `82/4/1/0`) or keep Typeburn 3-tuple `correct/incorrect/extra`? Recommend 3 (field was always-zero, removed deliberately).
2. **`session` timestamp:** image shows `00:00:15 session`. Typeburn `metrics.Result` has no start clock-time field (only `DurationMs`). Add one, or omit?
3. **Graph style:** true dual-axis line (WPM line + Errors line + red X) — confirm full Monkeytype fidelity vs a lighter "WPM line + error markers only" variant.
4. **Heatmap:** keep below the new stats grid (current), or drop for Monkeytype parity?
