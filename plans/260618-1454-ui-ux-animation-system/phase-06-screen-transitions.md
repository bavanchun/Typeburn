---
phase: 6
title: "Screen transitions"
status: completed
priority: P2
effort: "5h"
dependencies: [2]
---

# Phase 6: Screen transitions

## Overview
Soften hard screen cuts with a short transition, primarily **Typing → Result**.
Color themes get a **dim-curtain crossfade**; `NO_COLOR`/`mono` get a **directional
wipe** (attribute-only, identical cell count) — this is the auto-adapt that preserves
the layout-identical invariant (line count + rune width). Transitions are root-owned because they span two screens.

## Requirements
- Functional: ~250ms transition on Typing→Result (others may stay instant cuts);
  crossfade in color, wipe under NO_COLOR.
- Non-functional: never reflow; root `View()` stays the single chokepoint; self-stopping;
  files < 200 LOC.

## Architecture
**Why root-owned:** a transition shows the *outgoing* frame and the *incoming* frame
together, which only `app.Model.View()` can compose (it already routes all screens at
`model_view.go:48`). Screen sub-models can’t see each other.

**State (`app.Model`):** `transition *transitionState` with
`{ fromFrame string; toScreen Screen; startMs, durMs int64 }`. `nil` = no transition.

**Capture-then-animate:** when a transition-worthy nav occurs (e.g. `ResultMsg`), snapshot the
**already-composed, placed** outgoing frame (the post-`lipgloss.Place` string that
`model_view.go:60-63` produced for the user's *last* render — carry it forward rather than
re-rendering the typing model after completion, which could paint a different frame) into
`fromFrame`, set `toScreen`, `startMs=animNowMs`, bootstrap `frameTickCmd()`.

**Render (`transition.go`):** given `progress = anim.EaseInOutQuad(...)`:
- **Color (crossfade):** dim the `fromFrame` toward background and the incoming frame up from dim.
  Implemented as an attribute-layer crossfade: outgoing wrapped in `Faint(true)` while
  `progress<0.5`, incoming wrapped in `Faint(true)` while `progress<1`, swapping at the midpoint.
  (Honest limitation: true per-cell alpha isn’t possible; faint-curtain is the terminal equivalent.)
- **NO_COLOR (wipe):** reveal the incoming frame row-by-row over `from`:
  `visibleRows = int(totalRows*progress)`; top `visibleRows` lines come from `toFrame`, the rest
  from `fromFrame`. Pure string-line slicing — same width, same line count.
- Resolve color vs NO_COLOR via `th.Color(RoleBg)==nil`.

**Completion (derived expiry — red-team P1):** `View()` is a value receiver and **cannot** mutate
`transition`; and a self-stopping loop must not depend on a "final cleanup tick" that may never be
scheduled (the loop stops *because* the transition ended). So expiry is **derived, not mutated in
View**: `View` treats the transition as active only while `m.animNowMs < startMs+durMs`, otherwise
it ignores `fromFrame` and renders `toScreen` normally. The actual nil-out of `transition` happens
lazily in `Update` on the next message (any KeyPress/tick), which is harmless because View already
stopped using it. This removes the stall hazard with zero reliance on a trailing tick.

**Cancellation (red-team P2):** `transition` is invalidated when its geometry or target becomes
stale: `WindowSizeMsg` (`model.go:131`) cancels it (snap to `toScreen`) — the snapshot was taken
at the old width and would mismatch; `AbortMsg` (`model.go:109`) clears it. These prevent old-width
`fromFrame` bleeding over a new-width `toFrame`.

**Scope guard (YAGNI):** implement Typing→Result only. Home→Typing stays an instant cut (user just
pressed enter; immediacy is correct). Other navs stay instant unless the user later asks.

## Related Code Files
- Create: `internal/app/transition.go` — `transitionState`, `renderTransition(from, to string, progress float64, noColor bool) string`.
- Create: `internal/app/transition_test.go` — wipe row math, crossfade midpoint swap, line-count/width invariants, NO_COLOR branch.
- Modify: `internal/app/model_view.go` — when `transition != nil`, return `renderTransition(...)` (still via `altView`, single chokepoint).
- Modify: `internal/app/model_result.go` (ResultMsg site) — capture `fromFrame`, set transition, bootstrap frame cmd.
- Modify: `internal/app/anim_driver.go` — include `transition` liveness in `animActive` (declared P2; populated here).

## Implementation Steps
1. `transition.go`: define state + `renderTransition`; split both frames to `[]string` lines, pad
   to equal line counts/widths defensively, compose per mode.
2. At the ResultMsg site, snapshot outgoing frame (call the typing View used by `model_view.go:60`),
   set `transition`, return `frameTickCmd()`.
3. In `model_view.go`, branch to `renderTransition` only while `animNowMs < startMs+durMs`
   (derived expiry); otherwise fall through to the normal per-screen switch. Do NOT mutate in View.
4. Nil-out `transition` lazily in `Update` (next message after expiry); cancel it on
   `WindowSizeMsg` and `AbortMsg`.
5. Tests: wipe reveals exactly `floor(rows*progress)` rows; crossfade swaps at 0.5; both preserve
   line count and per-line rune width; NO_COLOR path never emits color; expired transition is
   ignored by View even if not yet nil-ed; resize/abort cancel cleanly.

## Success Criteria
- [x] Typing→Result shows a ~250ms crossfade (color) / wipe (NO_COLOR), then the live Result reveal.
- [x] Transition never changes total line count or any line’s rune width (asserted both modes).
- [x] `model_view.go` remains the single return chokepoint (all paths via `altView`).
- [x] Transition expiry is derived in View (no value-receiver mutation); no stale-frame stall.
- [x] Resize during a transition snaps to `toScreen` (no old-width bleed); abort clears it.
- [x] `make test-race`, `go vet`, `gofmt -l` clean; files < 200 LOC.

## Risk Assessment
- **Faint-nesting reliability:** wrapping an already-styled frame in `Faint(true)` may not override
  inner SGR on every terminal. Mitigation: if unreliable in manual check, fall back to the wipe for
  color themes too (wipe is robust everywhere) — note the decision in P7 docs.
- **Snapshot timing:** capturing the outgoing frame must happen before state flips to Result.
  Order the ResultMsg handler to snapshot first.
- **Degraded/min-size interaction:** transition must respect the 60×20 degraded gate
  (`model_view.go:32`) — skip transition when degraded.
