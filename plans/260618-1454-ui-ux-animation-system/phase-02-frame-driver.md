---
phase: 2
title: "Frame driver"
status: pending
priority: P1
effort: "5h"
dependencies: [1]
---

# Phase 2: Frame driver

## Overview
Add the self-stopping animation frame loop and plumb a shared `nowMs` to screen
`View()`s. This is the single, app-owned driver every animated moment hooks into.
The existing 100ms `timer.go` tick is **not touched**.

## Requirements
- Functional: new `FrameTickMsg`; app re-arms a ~33ms tick only while ≥1 screen has
  live animation; screens expose `HasActiveAnim(nowMs) bool`; `nowMs` reaches View.
- Non-functional: zero ticks scheduled when fully idle; no change to existing tested
  timer/WPM/completion paths; files < 200 LOC.

## Architecture
**Message:** `FrameTickMsg{ t time.Time }` in `internal/ui/messages.go` (sibling of the
existing `tickMsg`, distinct type so routing is unambiguous).

**Driver ownership:** `app.Model` owns animation orchestration in a new
`internal/app/anim_driver.go`:
- `frameInterval = 33 * time.Millisecond` (single const; P7 may retune).
- `frameTickCmd()` → `tea.Tick(frameInterval, ...)` returning `FrameTickMsg` (single-fire;
  re-armed only when animation remains active — mirrors `timer.go:17` pattern).
- `m.animActive(nowMs) bool` = OR of the active screen's `HasActiveAnim(nowMs)` plus any
  in-flight root-owned transition (P6).
- On `FrameTickMsg`: stamp `m.animNowMs`, forward to active screen's `Update` so it can
  advance its own tween state, then return `frameTickCmd()` **iff** `animActive` else `nil`.

**Kicking the loop:** any event that *starts* an animation must return `frameTickCmd()`
once to bootstrap the loop (e.g. ResultMsg entry in P4, first keystroke in P3,
transition start in P6). Self-arming keeps it going; self-stop ends it.

**nowMs plumbing:** add `animNowMs int64` to `app.Model`. Screens that animate take
`nowMs` via their existing `View()`—**preferred**: store `nowMs` on the sub-model when
forwarding `FrameTickMsg` in `Update` (no `View()` signature churn; matches how
`TypingModel.nowMs` already works at `screen_typing.go:28`). View reads the stored field.

**Interface (kept tiny to protect <200 LOC on root):**
```go
// implemented by screens that animate (typing, result)
type Animatable interface{ HasActiveAnim(nowMs int64) bool }
```

## Related Code Files
- Create: `internal/app/anim_driver.go` — `frameInterval`, `frameTickCmd`, `animActive`, FrameTick routing.
- Create: `internal/app/anim_driver_test.go` — self-stop logic (active→re-arm, idle→nil).
- Modify: `internal/ui/messages.go` — add `FrameTickMsg`.
- Modify: `internal/app/model.go` — add `animNowMs`; handle `FrameTickMsg` in `Update`;
  `Init()` stays `nil` (idle = no tick).
- Modify: screen sub-models (P3/P4/P6) to implement `HasActiveAnim` + store `nowMs`
  (declared here, populated by later phases).

## Implementation Steps
1. Add `FrameTickMsg{ t time.Time }` to `messages.go`.
2. `anim_driver.go`: const + `frameTickCmd()`; `func (m Model) animActive(nowMs int64) bool`
   switching on `m.screen` to call the active sub-model's `HasActiveAnim`, OR transition state.
3. In `model.go Update`, add an **explicit** `case ui.FrameTickMsg:` branch **before** the
   generic delegation (the generic block at `model.go:165-189` must NOT be the path that
   handles it — otherwise the sub-model's cmd is returned and `maybeFrameCmd` is lost). The
   branch: set `m.animNowMs = msg.t.UnixMilli()`; forward to the active screen's `Update` to
   capture `subCmd` (so it stores `nowMs`); `return m, tea.Batch(subCmd, m.maybeFrameCmd())`
   where `maybeFrameCmd` returns `frameTickCmd()` if `animActive` else `nil`.
4. **Sub-models must handle the message:** add a `case ui.FrameTickMsg:` to
   `TypingModel.Update` and `ResultModel.Update` that stores `nowMs = msg.t.UnixMilli()` and
   returns `(m, nil)`. Without this case the "advance tweens" forwarding silently no-ops and
   nothing animates. (Real per-screen logic lands in P3/P4; the case + field land here.)
5. Define `HasActiveAnim(nowMs) bool` on `TypingModel` and `ResultModel` returning `false`
   for now (real logic in P3/P4) so the interface compiles.
6. Unit-test the self-stop: a model reporting active re-arms (`tea.Batch` non-nil frame cmd);
   reporting idle returns no frame cmd. Assert `FrameTickMsg` never advances WPM/completion.

## Success Criteria
- [ ] With all screens reporting idle, `FrameTickMsg` handling returns no frame re-arm (loop stops).
- [ ] When a screen reports active, handling returns `tea.Batch(subCmd, frameTickCmd())` (non-nil frame cmd).
- [ ] `FrameTickMsg` is handled by an explicit root case, not the generic delegation block.
- [ ] `TypingModel`/`ResultModel.Update` each have a `FrameTickMsg` case that stores `nowMs`.
- [ ] `Init()` still returns `nil`; launching the app schedules no frame tick until an animation starts.
- [ ] Existing `timer.go` tick + WPM/completion tests unchanged and green.
- [ ] `go test ./... -race`, `go vet`, `gofmt -l` all clean; files < 200 LOC.

## Risk Assessment
- **Double-tick coupling:** during typing both the 100ms timer tick and 33ms frame tick run.
  Acceptable (independent loops); ensure `FrameTickMsg` never advances WPM/completion (separation of concerns).
- **Root model growth:** keep all driver logic in `anim_driver.go`; `model.go` gains only a
  thin `case` + one field to stay < 200 LOC.
- **Forgotten bootstrap:** if a phase starts an animation but forgets to return `frameTickCmd()`,
  nothing animates. P7 adds an integration test asserting the loop arms on ResultMsg.
