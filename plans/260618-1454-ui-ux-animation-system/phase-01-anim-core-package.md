---
phase: 1
title: "Anim core package"
status: completed
priority: P1
effort: "4h"
dependencies: []
---

# Phase 1: Anim core package

## Overview
Create `internal/anim`, a **pure, UI-free** package holding all motion math:
easing curves, RGB color interpolation, a generic tween, and a clock that tracks
whether any animation is still live. No Bubble Tea / Lip Gloss imports — it joins
`typing`/`metrics`/`words` as testable pure logic. Everything downstream depends on this.

**Spring removed (red-team YAGNI):** no consumer in P3–P6 uses a spring; all use
tweens + easing. Do **not** build `spring.go`. Add it back only if a real consumer appears.

## Requirements
- Functional: easing fns, `LerpColor`, `Tween`/`LerpInt`/`LerpFloat`, `Clock.Active()`.
- Non-functional: zero UI deps; deterministic (pure fns of time); 100% table-tested;
  each file < 200 LOC.

## Architecture
Time model: every animation is a pure function of `(startMs, nowMs, durMs) → progress∈[0,1]`,
then `progress → eased → value`. No goroutines, no internal clocks — the caller
supplies `nowMs` (epoch-ms), mirroring `metrics.Compute`'s post-hoc replay.

Color interpolation works on `image/color.Color` (stdlib) so the package never
imports lipgloss. Callers convert the returned RGB to a lipgloss color at the
UI boundary. `LerpColor(from, to, t)` returns `color.RGBA`; if either input is
`nil` (NO_COLOR), it returns `nil` so callers can branch to attribute-only.

## Related Code Files
- Create: `internal/anim/easing.go` — `EaseOutCubic`, `EaseInOutQuad`, `EaseOutQuad`, `Clamp01`.
- Create: `internal/anim/color.go` — `LerpColor(from, to color.Color, t float64) color.Color`.
- Create: `internal/anim/tween.go` — `Tween{StartMs, DurMs, Ease}` + `Progress(nowMs)`, `Done(nowMs)`, `LerpInt`, `LerpFloat`.
- Create: `internal/anim/clock.go` — `Clock` aggregating active tweens; `Active(nowMs) bool`.
- Create: `internal/anim/easing_test.go`, `color_test.go`, `tween_test.go`, `clock_test.go`.

## Implementation Steps
1. `easing.go`: implement (research-confirmed formulas):
   - `EaseOutCubic(t) = (t-1)^3 + 1`
   - `EaseInOutQuad(t)` piecewise (accelerate→decelerate)
   - `EaseOutQuad(t) = t*(2-t)` (count-up curve)
   - `Clamp01(t)` guard. All take/return `float64` in [0,1].
2. `tween.go`: `Tween{StartMs int64; DurMs int64; Ease func(float64) float64}`.
   - `Progress(nowMs) float64` = `Ease(Clamp01((nowMs-StartMs)/DurMs))`; returns 0 before start, 1 after end.
   - `Done(nowMs) bool` = `nowMs >= StartMs+DurMs`.
   - `LerpFloat(from, to, t)`, `LerpInt(from, to, t)` helpers (count-up uses `LerpInt`).
3. `color.go`: `LerpColor` — `from.RGBA()`/`to.RGBA()` give 0–65535; lerp each channel,
   `/257` to 0–255, return `color.RGBA{R,G,B,255}`. `nil` in either → `nil` out.
4. `clock.go`: `Clock` holds a slice of `Tween`; `Active(nowMs)` true if any `!Done(nowMs)`;
   `Add`, `Prune(nowMs)`. This is the signal the frame driver (P2) polls to self-stop.
5. Tests: table-driven, real values — boundary t=0/0.5/1, monotonicity, `LerpColor` channel
   math + `nil` passthrough, `LerpInt` endpoints, `Done`/`Active` transitions.

## Success Criteria
- [x] `go test ./internal/anim/ -race -count=1` green; table-driven, no mocks.
- [x] `go list -deps ./internal/anim` shows **no** `charm.land`/`charmbracelet` imports.
- [x] Every file < 200 LOC; `gofmt -l` empty; `go vet ./internal/anim/` clean.
- [x] `LerpColor(nil, x, t)` and `LerpColor(x, nil, t)` both return `nil`.

## Risk Assessment
- RGBA `/257` rounding: acceptable (research-confirmed); assert exact endpoints in tests.
- Keep the surface minimal (YAGNI): only ship what P3–P6 consume (easing, tween/lerp,
  color, clock). No spring, no speculative helpers.
