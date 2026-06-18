---
phase: 4
title: "Result reveal"
status: pending
priority: P2
effort: "5h"
dependencies: [2]
---

# Phase 4: Result reveal

## Overview
Animate the Result screen entrance: the big WPM digits **count up** 0→final, the
sparkline **draws in** left→right, and stat cards **stagger-fade** in. The Result
screen is not perf-critical, so this is the lowest-risk visible win.

## Requirements
- Functional: WPM count-up (~600ms easeOutQuad), sparkline progressive reveal,
  staggered stat-card appearance; all settle to the exact final frame.
- Non-functional: no layout jitter (reserve max digit width); NO_COLOR → cards/sparkline
  appear via attribute step (faint→normal) not color fade; always-on.

## Architecture
**Entry trigger:** Result screen begins animating when it becomes active. The root sets a
`revealStartMs` when routing `ResultMsg` (`model.go:97` path / `handleResultMsg`) and bootstraps
the frame loop with `frameTickCmd()`. **Every `ResultMsg` must unconditionally (re)set
`revealStartMs = animNowMs`** (and P5's `isNewBest`/`celebrateStartMs`) so a second result never
inherits an already-elapsed window from the prior result — the `ResultModel` is reused on the
root (`model.go:21`), so stale state would otherwise suppress the animation or fire spurious confetti.

**State (on `ResultModel`):** `revealStartMs int64`, `nowMs int64` (stamped from `FrameTickMsg`).
All reveal values derive purely from `(revealStartMs, nowMs)`.

**Count-up:** display value = `anim.LerpInt(0, finalWPM, EaseOutQuad(progress))` with
`countUpMs=600`. **Reserve width** for the final digit count (e.g. right-align in a fixed cell)
so 9→10→100 never shifts the big-digit block. Big digits render via existing `ascii_big_digits.go`.

**Sparkline draw-in:** reuse `sparkline.go`; reveal the first `n=int(barCount*progress)` bars,
render remaining positions as blank cells of equal width (no reflow). `drawInMs=500`.

**Stat-card stagger:** each `StatCard` (`stat_card.go`) has an offset start
(`card[i].start = revealStartMs + i*staggerMs`, `staggerMs≈120`); before its start the card
renders faint/placeholder at full width, then fades to normal. NO_COLOR: faint→normal attribute step.

**HasActiveAnim:** `true` while `nowMs < revealStartMs + totalRevealMs`
(`totalRevealMs = max(countUpMs, drawInMs, lastCardStart+cardFadeMs)`), else `false` → self-stop.

**Update hint / new-best badge:** unchanged in this phase except that the new-best badge’s
appearance time is exposed so P5 can hang the celebration off it.

## Related Code Files
- Create: `internal/ui/screen_result_reveal.go` — reveal state + pure helpers
  (`countUpValue`, `sparkReveal`, `cardVisible`), NO_COLOR-aware.
- Create: `internal/ui/screen_result_reveal_test.go` — count-up endpoints/monotonic, spark bar count,
  card stagger gating, width-reservation invariant, NO_COLOR attr path.
- Modify: `internal/ui/screen_result.go` — add `revealStartMs`/`nowMs`; `HasActiveAnim`;
  stamp `nowMs` on `FrameTickMsg`.
- Modify: `internal/ui/screen_result_view.go` + `result_render_helpers.go` — consume reveal helpers.
- Modify: `internal/app/model_result.go` (handleResultMsg site) — set `revealStartMs`, bootstrap frame cmd.

## Implementation Steps
1. `screen_result_reveal.go`: pure functions of `(revealStartMs, nowMs)`; constants `countUpMs`,
   `drawInMs`, `staggerMs`, `cardFadeMs`.
2. Wire `revealStartMs` at the ResultMsg handling site; return `frameTickCmd()` to start the loop.
3. Stamp `nowMs` from `FrameTickMsg` in `ResultModel.Update`; implement `HasActiveAnim`.
4. Replace the static WPM render with `countUpValue(...)`; ensure fixed-width digit slot.
5. Gate sparkline + stat cards through reveal helpers; verify equal-width placeholders.
6. Tests pin `nowMs`; assert final frame equals the pre-animation static render exactly.

## Success Criteria
- [ ] WPM counts 0→final over ~600ms with easeOut; lands exactly on final (no off-by-one blur).
- [ ] Sparkline fills left→right; stat cards appear staggered; no horizontal jitter at any frame.
- [ ] Final settled frame is **byte-identical** to the current static Result render (golden test).
- [ ] Under `NO_COLOR`, reveal uses faint→normal only; layout-identical (line count + rune width).
- [ ] `HasActiveAnim` false after `totalRevealMs`; loop self-stops.
- [ ] A second consecutive result animates fresh (no inherited elapsed window from the prior result).
- [ ] `make test-race`, `go vet`, `gofmt -l` clean; files < 200 LOC.

## Risk Assessment
- **Width jitter on digit growth (UX flag):** mitigated by reserved fixed-width digit slot; test asserts
  constant rendered width across the count-up.
- **Golden churn:** Result goldens become time-dependent; pin `nowMs` and add a dedicated
  "settled frame == static" golden to lock the end state.
- **Double-counting with update hint:** ensure the muted update-hint footer is excluded from reveal timing.
