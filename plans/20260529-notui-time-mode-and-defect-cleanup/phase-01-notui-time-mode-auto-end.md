---
phase: 1
title: "Notui Time-Mode Auto-End"
status: completed
priority: P1
effort: "3-4h"
dependencies: []
---

# Phase 1: Notui Time-Mode Auto-End

## Overview
Make `--no-tui` Time mode complete on the clock even with zero trailing input, and add the missing `runner_test.go`. Approach A (chosen in brainstorm): async read on a goroutine + `select` over {event channel, timer}. Words/Quote/Code paths unchanged.

## Requirements
- **Functional:** In Time mode (once started), `runLoop` must *terminate* at wall-time `StartMs + Length*1000` regardless of further input — i.e. the loop ends on the clock, not on a keystroke. The *reported* `DurationMs` from `Compute` may be smaller than `Length*1000` when AFKTrim fires (trailing idle >7s) — that is intended TUI parity, not a bug. No keystroke arriving after the timer fires may enter the engine log.
- **Functional:** Words/Quote/Code still complete via `Engine.Complete(nowMs)` exactly as today (event-driven).
- **Non-functional:** Tests deterministic (no real sleeps); reuse/extend the `now func() time.Time` seam. `-race` clean. New file < 200 LOC.

## Architecture

**Current (buggy) flow** — `internal/cli/notui/runner.go:30-61`:
`runLoop` calls `ReadEvent(rd)` (blocking) at the top of every iteration; the `completed()` check runs only after a read returns. Time mode has no clock source → hangs.

**Target flow:**
- Spawn a reader goroutine that loops `ReadEvent(rd)` and sends `Event`/`error` into a buffered channel.
- `runLoop` becomes a `select`:
  - `case ev := <-events:` apply to engine, render status, then check `completed()`.
  - `case <-timer:` (Time mode only) → completed by clock; build result with `endMs = start + Length*1000`; render/write; return.
  - reader error/EOF flows through the channel as a terminating event.
- Timer source: inject a seam alongside `now` — e.g. `after func(d time.Duration) <-chan time.Time` (defaults to `time.After`) so tests fire it synchronously.
- **Timer arms on the first-keystroke transition only** (`StartMs() 0 → non-zero`), matching TUI where the clock starts on first input. By construction the timer-fire instant then equals `StartMs + Length*1000`, which is exactly what `endMs()` returns — fire-instant and `endMs` are the *same clock event*, so there is **no desync** (red-team Axis 2). Do NOT arm at loop entry.
- **Zero-input Time test (no keystroke ever):** no timer is ever armed → `runLoop` stays blocked on the event channel until Ctrl-C (SIGINT → `ErrAbort` via `lifecycle.go`) or stdin EOF (ReadEvent error → loop returns). This is the accepted behavior ("no input, no test"); we do NOT add a separate idle timer. (User decision.)
- **AFKTrim parity (red-team Axis 5):** `Compute` already runs `TrimAFK` for Time mode — if trailing idle >7s, it rewrites `endMs` down to `lastKeyMs`. We KEEP this (TUI parity, user decision). Consequence: the reported `DurationMs` is `min(Length*1000, lastKeyMs-StartMs)` after trim, NOT always `Length*1000`. Test assertions MUST account for this (see Steps).
- **endMs already correct:** `endMs()` returns `start + Length*1000` for Time mode — keep it; the timer-fire path calls `Compute` with it (and `Compute` may AFK-trim it). The expiry keystroke does not exist on the timer path (timer fires independently of input), so no post-expiry keystroke can enter the log.

**Concurrency invariants (red-team Axis 1/4) — HARD RULES, enforced in review + `-race`:**
- `session.Engine` (`Apply`/`Backspace`/`Log`/`Progress`/`StartMs`) is mutated/read **only** from the `runLoop` goroutine. The reader goroutine touches **only** `bufio.Reader` and the send channel — never the engine, never `out`.
- All `RenderPrompt`/`RenderStatus`/`write` output happens **only** in `runLoop`.
- The reader goroutine **MUST NOT** `close` the channel (no `defer close`); it only sends. `runLoop` only receives. → no send-on-closed panic.
- Channel is buffered cap-1 so the first post-return send does not block. After `runLoop` returns (timer/abort/ctx), one reader goroutine may remain blocked on `ReadEvent` or on a second channel send — **intentionally abandoned at process teardown**, same as today's `runRaw` `done` goroutine (`lifecycle.go:69`). Tests MUST tolerate this and **must not** use goroutine-leak assertions on this path.

## Related Code Files
- Modify: `internal/cli/notui/runner.go` (runLoop → select; add `after` seam; arm timer for Time mode)
- Create: `internal/cli/notui/runner_test.go` (deterministic Time-mode + Words completion coverage)
- Read for context: `internal/cli/notui/reader.go`, `lifecycle.go`, `render.go`, `internal/runner/session.go`, `internal/metrics/compute.go`

## Implementation Steps
1. **Test-first** — write `runner_test.go` with these deterministic cases (injected `now` + `after`; fire timer via manual channel send, never real sleep):
   - **(a) Auto-end, no trailing idle:** Time/30s, feed runes whose last keystroke is within 7s of the limit (so AFKTrim does NOT fire), then fire the timer. Assert `runLoop` returns a result with `DurationMs == Length*1000` and char counts matching the typed runes.
   - **(b) Auto-end with trailing idle >7s (AFKTrim path):** feed runes ending well before the limit, fire timer. Assert `DurationMs == lastKeyMs - StartMs` (AFK-trimmed), NOT `Length*1000`. This is the core parity case — getting the assertion right here is the main correctness check.
   - **(c) Words-mode regression:** feed enough runes to complete; assert event-driven completion unchanged, no timer involved.
   - **(d) Backspace/EventNone-only log:** feed only deletions then fire timer; assert no panic, valid (accuracy=100, WPM=0) result.
2. Add an `after func(time.Duration) <-chan time.Time` seam to `runLoop` (default `time.After` in `Run`), alongside the existing `now`.
3. Extract the read into a goroutine feeding a cap-1 buffered `chan readMsg{ev Event; err error}`. Goroutine: `for { ev,err := ReadEvent(rd); ch <- readMsg{ev,err}; if err != nil { return } }`. **No `close(ch)`.**
4. Rewrite the loop as `select` over the read channel vs the (lazily-armed) timer channel:
   - On `readMsg` with err → return err. On event → `Apply`/`Backspace` in runLoop; **arm the timer on the first keystroke that makes `StartMs() != 0`** if Time mode and not yet armed; render status; `completed()` check (unchanged for Words/Quote/Code).
   - On timer fire (Time mode) → `metrics.Compute(Log(), Mode, endMs(session, nowMs))`; write/summary; return. **Skip the final `RenderStatus`** on this path (avoid a stale display frame — red-team Axis 6).
5. Verify the zero-input path: with no keystroke, timer is never armed → loop blocks on the channel until EOF/SIGINT. Confirm `runRaw`'s ctx/signal `select` (lifecycle.go) still interrupts it. No code needed beyond not-arming; document it.
6. Run `go test ./internal/cli/notui/ -race -count=1 -v`, then full `go test ./... -race`, `gofmt -l .`, `go vet ./...`.

## Success Criteria
- [ ] `runner_test.go` exists; cases (a)-(d) from Step 1 all pass, deterministic, `-race` green (no real sleeps).
- [ ] Timer arms on first keystroke only; fire-instant == `endMs` by construction (no loop-entry arming).
- [ ] AFKTrim parity preserved: trailing-idle Time test reports trimmed duration (case b), not `Length*1000`.
- [ ] Zero-input Time test does not auto-end; terminates only on EOF/SIGINT (documented, not a separate timer).
- [ ] **Concurrency invariants hold:** engine + all output touched only in `runLoop`; reader goroutine never closes the channel; tests use no goroutine-leak assertions. `-race` green.
- [ ] Words/Quote/Code completion byte-identical to before (existing tests + case c pass).
- [ ] `runner.go` and `runner_test.go` each < 200 LOC; gofmt empty; `go vet` clean.

## Risk Assessment
- **Timer/endMs desync (was CRITICAL):** resolved by arming on the first-keystroke transition so fire-instant == `endMs` is structural, not coincidental. Reviewer must reject any loop-entry arming.
- **AFKTrim vs duration assertion (was MAJOR):** the naive `DurationMs == Length*1000` assertion is FALSE under trailing idle >7s. Split into case (a) no-trim and case (b) trim. This is the most likely first-write test failure — write both.
- **Data race (was CRITICAL-leaning):** only safe if the engine is mutated solely in `runLoop`. Hard rule in Success Criteria; `-race` catches violations.
- **Goroutine leak:** one blocked reader may remain at teardown — accepted (matches `runRaw`). Cap-1 buffer prevents deadlocking `runLoop`'s return. Never `close` the channel.
- **Determinism:** real `time.After` in tests = flaky. The `after` seam is mandatory.
- **Scope creep:** do NOT refactor `reader.go` event parsing; only wrap its call site.
