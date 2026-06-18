---
title: "UI/UX Animation System"
description: "Stdlib-only terminal motion system for Typeburn: pure anim package, self-stopping frame driver, animated caret, result reveal, new-best celebration, and screen transitions — always-on, NO_COLOR auto-adapting."
status: pending
priority: P2
branch: "main"
tags: [ui, animation, motion, tui]
blockedBy: []
blocks: []
created: "2026-06-18T08:01:27.729Z"
createdBy: "ck:plan"
source: skill
---

# UI/UX Animation System

## Overview

Add a motion layer to Typeburn's Bubble Tea v2 / Lip Gloss v2 TUI to make it feel
alive without breaking its strict architecture. Four moments: an animated caret
(blink + new-cell fade + leave-trail), a result reveal (WPM count-up + sparkline
draw-in + stat stagger), a new-best celebration burst, and screen transitions
(crossfade in color / wipe under NO_COLOR).

**Hard constraints carried from brainstorm (`docs/mdocs/20260618/01-20260618.md`):**
- **Stdlib only** — hand-rolled easing/interpolation; no harmonica, no bubbles.
- **Always on** — no reduced-motion toggle; motion **auto-adapts** under
  `NO_COLOR`/`mono` to attribute-only variants so layout stays **layout-identical
  (line count + rune width preserved)**. NOTE: this is *layout*-identical, not
  literally *byte*-identical — the celebration (P5) overlays glyphs onto blank
  padding cells, changing rune *content* of those cells while preserving width.
  Flagged for user confirmation (see Open decisions).
- **Pure-logic layering preserved** — new `internal/anim` package is UI-free
  (joins `typing`/`metrics`/`words`); UI deps stay in `ui`/`app`/`theme`.
- **<200 LOC per file**, allowed deps unchanged (stdlib + charm.land/* + cobra + x/*).

**Key enabling facts (from research):**
- Bubble Tea v2's cell-diff "Cursed Renderer" emits writes only for changed cells
  → full-frame redraw at 30fps is cheap; an unchanged `View` costs ~zero.
- `tea.Tick(d, fn)` fires **once**; loops are built by re-arming in `Update` →
  a self-stopping frame loop is idiomatic and burns zero CPU when idle.
- Color interp: `color.Color.RGBA()` (0–65535) → `/257` → `lipgloss.Color("#RRGGBB")`.
- `theme.Color(Role)` returns `nil` under NO_COLOR → the auto-adapt detection hook.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Anim core package](./phase-01-anim-core-package.md) | Pending |
| 2 | [Frame driver](./phase-02-frame-driver.md) | Pending |
| 3 | [Caret animation](./phase-03-caret-animation.md) | Pending |
| 4 | [Result reveal](./phase-04-result-reveal.md) | Pending |
| 5 | [New-best celebration](./phase-05-new-best-celebration.md) | Pending |
| 6 | [Screen transitions](./phase-06-screen-transitions.md) | Pending |
| 7 | [Hardening and docs](./phase-07-hardening-and-docs.md) | Pending |

## Build order & dependencies

```
P1 (anim pkg) ──► P2 (frame driver) ──┬─► P3 (caret)
                                       ├─► P4 (result reveal) ──► P5 (celebration)
                                       └─► P6 (transitions)
                                                  └─► P7 (hardening + docs)
```

- P1 → P2 are the foundation; P3/P4/P6 are independent consumers of P2.
- P5 depends on P4 (celebration overlays the result reveal).
- P7 closes out: benchmarks, golden/NO_COLOR layout-identical verification, docs.

## Cross-cutting design decisions

1. **Two independent tick loops, by design.** The existing 100ms `timer.go` tick
   (WPM/Time-mode completion) stays **untouched** to preserve tested behavior. A
   **new** `FrameTickMsg` anim tick (~33ms) drives visuals and is **self-stopping**:
   it re-arms only while ≥1 animation is live. Caret *blink* (slow, ~530ms cycle)
   can ride the 100ms tick via a time-threshold; the fast 33ms frames are needed
   only during the ~150ms fade/trail windows, so idle/paused typing falls back to
   cheap cadence automatically.
2. **`nowMs` is passed down, never read ad-hoc.** Animations are pure functions of
   `(startMs, nowMs, durationMs)` so they stay deterministic and unit-testable
   (mirrors how `metrics.Compute` replays a log post-hoc).
3. **NO_COLOR auto-adapt is centralized.** A single helper resolves "do we have
   color?" via `theme.Color(Role) != nil`; every animation asks it and swaps to
   attribute-only (reverse/faint/bold, wipe instead of fade, no extra cells).
4. **No layout mutation, ever.** Count-up reserves max digit width; confetti
   overlays existing blank cells; transitions never reflow. Guarded by golden tests.

## Open decisions deferred to validation/red-team

- Frame cadence: 33ms (30fps) default vs 50ms (20Hz) SSH-safer. Benchmark gate in P7.
- Caret trail jitter risk (UX researcher flagged) — verify legibility at speed in P3.
- **"byte-identical" wording**: the brainstorm constraint said "byte-identical layout".
  Resolved as **layout-identical** (line count + rune width). Celebration glyphs change
  rune content of blank cells but not layout — confirm this reading is acceptable to user.

## Red-team resolutions (applied to phases)

Deep-mode adversarial review surfaced and these are now folded into the phases:
- **FrameTick routing (P1):** root adds an explicit `case ui.FrameTickMsg` returning
  `tea.Batch(subCmd, maybeFrameCmd())`; `TypingModel`/`ResultModel.Update` add a
  `FrameTickMsg` case that stores `nowMs` — without it nothing animates. (P2, P3, P4)
- **Bootstrap-on-edge (P1):** frame loop is bootstrapped only on the idle→active edge
  via an explicit `frameLoopArmed bool`, never per-keystroke — otherwise overlapping
  `tea.Tick` timers multiply and never self-stop. (P3)
- **Transition clear (P1):** `View()` is a value receiver and cannot mutate state, and
  the self-stopping loop could deadlock its own cleanup. Transition expiry is therefore
  **derived** (`animNowMs >= startMs+durMs` → ignore it in View) and nil-out happens on
  the next Update message — no reliance on a final cleanup tick. (P6)
- **Prefix-token cache is mandatory in P3** (not deferred): cell-diff saves terminal
  bytes but NOT the per-frame `Render`/alloc cost upstream in `word_stream_renderer.go`.
- **Clock seam:** `applyText` gets an injectable `nowFn` (mirrors existing `seed`) so
  caret goldens are deterministic; teatest cannot pin `time.Now()` otherwise. (P3)
- **Celebration overlay** restricted to ASCII width-1 glyphs on full-width blank margin
  rows (not surgical column splicing into SGR-styled lines). (P5)
- **Edge cases added:** resize cancels active transition; abort/ctrl+r reset caret +
  transition state; every `ResultMsg` unconditionally resets reveal + new-best state. (P4, P5, P6)
- **Spring cut (YAGNI):** no consumer uses it; removed from P1.
- **Pre-start blink:** typing screen bootstraps a tick on entry so the caret blinks
  before the first keystroke. (P3)

## Dependencies

No cross-plan dependencies. All prior Typeburn plans in `plans/` are shipped/independent.
