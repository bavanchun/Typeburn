---
phase: 7
title: "Hardening and docs"
status: pending
priority: P1
effort: "4h"
dependencies: [3, 4, 5, 6]
---

# Phase 7: Hardening and docs

## Overview
Close out: benchmark the typing hot path, prove the NO_COLOR layout-identical
invariant across every animated screen, refresh goldens deterministically, retune
frame cadence if needed, enforce size/lint gates, and update docs. This is the
gate that decides the cadence and whether any flagged risk (trail, faint-nesting)
gets cut.

## Requirements
- Functional: full CI-equivalent green; settled-state goldens equal pre-animation output.
- Non-functional: typing hot path shows no meaningful regression; binary-size cap respected; docs current.

## Architecture
**Determinism harness:** all animation is a pure function of `nowMs`, so tests inject fixed
timestamps. Add a tiny test helper to render a screen at an explicit `nowMs` for golden capture
at chosen frames (start / mid / settled).

**Benchmark gate:** add `internal/ui` benchmark(s) that render the typing word-stream for a
100-word and a Code-mode buffer with an active caret animation, comparing against a static render.
The static-prefix token cache is **already built in P3** (mandatory), so the benchmark here
*verifies* it works: animated allocs/op must stay close to static (only the ≤2 animated cells
re-`Render`ed). If the benchmark shows the cache isn't engaging, fix P3's cache before proceeding.
Record the chosen `frameInterval` (33ms vs 50ms) here based on the benchmark + a manual SSH smoke check.

**Invariant proof:** a table test renders every animated screen (typing, result, celebration,
transition) with `NO_COLOR=1` at several `nowMs` and asserts each line’s **rune width and line
count** match the static render. NOTE this is the **layout-identical** guarantee (line count +
rune width), which is the architecturally meaningful invariant. It is NOT literal byte-identity:
the celebration overlays glyphs onto blank margin cells (rune *content* changes, width does not).
The plan/docs use "layout-identical" wording accordingly.

## Related Code Files
- Create: `internal/ui/anim_bench_test.go` — static vs animated render benchmarks.
- Create: `internal/ui/nocolor_layout_invariant_test.go` — per-screen rune-width/line-count equality under NO_COLOR.
- Create: `internal/app/anim_edge_cases_test.go` — resize mid-transition (snap), abort mid-reveal
  (state cleared), ctrl+r mid-caret-fade (caret reset), two consecutive results (second animates
  fresh; non-best after best → no confetti), min-terminal 60×20 (transition/celebration skip/clamp).
- Modify: golden files under `internal/ui` / `internal/app` testdata — re-capture with pinned `nowMs`.
- Modify: `docs/codebase-summary.md` — add `internal/anim` package + per-screen animation notes.
- Modify: `docs/system-architecture.md` — document the dual-tick model (100ms timer + 33ms self-stopping frame loop) and the NO_COLOR auto-adapt seam.
- Modify: `docs/project-roadmap.md` — record shipped animation feature.
- Modify: `README.md` — one line under Features (e.g., "subtle motion: animated caret, result reveal, celebration; auto-adapts to NO_COLOR").
- Modify: `CHANGELOG`/release notes as the repo convention dictates (no version bump here unless asked).

## Implementation Steps
1. Add deterministic golden helper; re-capture animated-screen goldens at start/mid/settled.
2. Write the NO_COLOR layout-invariant table test across all four moments.
3. Write benchmarks; run `go test -bench` for the typing path; decide on prefix-cache + cadence.
4. Run the full gate: `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` (empty), `make size-check`.
5. Update all docs listed above; ensure no plan-artifact references leak into code comments
   (explain the *why*, e.g. "self-stopping frame loop bounds idle cost", not "per phase 2").
6. Final whole-plan consistency pass: confirm constants (`frameInterval`, `blinkHalfMs`, `countUpMs`,
   `celebrateMs`, transition `durMs`) are consistent across phases and docs.

## Success Criteria
- [ ] `go test ./... -race -count=1` green; `go vet` clean; `gofmt -l .` empty; `make size-check` passes.
- [ ] Settled frame of every animated screen is byte-identical to its pre-animation static render
      (the *settled* end state IS byte-identical; only mid-animation differs, and celebration only by content).
- [ ] NO_COLOR layout-invariant test (rune width + line count) passes for typing, result, celebration, transition.
- [ ] Edge-case tests pass: resize/abort/ctrl+r/double-result/min-terminal.
- [ ] Typing hot-path benchmark confirms the P3 prefix-cache engages (animated allocs/op ≈ static).
- [ ] Chosen `frameInterval` recorded with rationale; idle app schedules zero frame ticks.
- [ ] Docs (`codebase-summary`, `system-architecture`, `roadmap`, `README`) updated; no plan refs in code.

## Risk Assessment
- **Golden flakiness:** any non-pinned timestamp reintroduces flakiness — assert all animated goldens
  inject `nowMs`; fail the phase if any golden reads wall-clock.
- **Binary-size creep:** new package + render code may approach the cap; if `size-check` fails, trim
  the spring (if unused) and dead helpers before relaxing the cap.
- **Scope-cut escalation:** if benchmark/UX review argues for dropping trail or color-crossfade, present
  the data to the user (these were confirmed choices) — do not silently remove.
