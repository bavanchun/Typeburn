---
phase: 2
title: Wiring Call Sites (TDD)
status: completed
effort: small
priority: P1
dependencies:
  - 1
---

# Phase 2: Wiring Call Sites (TDD)

## Overview

<!-- Updated: Validation Session 1 - corrected call chain, verified against codebase -->

Thread `settings.Punctuation`/`settings.Numbers` through the VERIFIED call
chain: `words.ForMode` ← `runner.NewSession` ← `ui.newTypingWithSeed`
(via `ui.NewTyping`, called from `app/model.go`, AND via the ctrl+r restart
path in `ui/screen_typing_actions.go`) ← and separately `cli.runSession` in
`cli/cmd_run_notui.go`. Two new positional bool params (`punctuation, numbers bool`),
not a bundled struct — matches existing style. Quote/Code modes must NOT
receive the transform — verified by ForMode's mode switch, not by caller
discipline. `NewCodeSession` is untouched (Code mode excluded by design).

## Requirements

- Functional:
  - `words.ForMode(g *Generator, m mode.Mode, length int, ql QuoteLen, punctuation, numbers bool) string`
    — apply `g.ApplyOptions(...)` only in the `ModeWords`/default-`ModeTime`
    branches; `ModeQuote` branch returns `g.Quote(ql).Text` untouched.
  - `runner.NewSession(m mode.Mode, length int, ql words.QuoteLen, seed int64, strict, punctuation, numbers bool) Session`
    — thread the two new bools into its `words.ForMode(...)` call.
    `runner.NewCodeSession` is UNCHANGED (Code mode excluded).
  - `ui.NewTyping(...)` and `ui.newTypingWithSeed(...)` (both in
    `screen_typing.go`) gain `punctuation, numbers bool` params, passed to
    `runner.NewSession(...)`. `TypingModel` gains `punctuation`/`numbers`
    fields alongside its existing `strict` field, so the ctrl+r restart path
    in `screen_typing_actions.go:30` can pass them into its own
    `newTypingWithSeed(...)` call (currently passes `m.strict`).
  - `app/model.go`'s `StartTestMsg` handler (lines ~86,89) updates its
    `ui.NewTyping(...)` call to also pass `m.settings.Punctuation,
    m.settings.Numbers`. Its `ui.NewTypingCode(...)` call is UNCHANGED (Code
    mode excluded).
  - `internal/cli/cmd_run_notui.go`'s `runSession` func (line ~54) updates its
    `runner.NewSession(...)` call to pass `settings.Punctuation,
    settings.Numbers`. Its `runner.NewCodeSession(...)` call is UNCHANGED.
- Non-functional: no behavior change when both flags are false (regression
  guard — this is the majority of existing tests). No `cmd_run.go` flag
  changes — settings-only control surface (Validation Session 1, Q4).

## Architecture

Same fan-out shape as `StrictMode`: one config field pair, one
generator-facing transform, each layer of the call chain (`ForMode` →
`NewSession` → `newTypingWithSeed`/`NewTyping` → `app/model.go` StartTestMsg
handler, and separately → `cli/cmd_run_notui.go`) threads the two bools
through as plain positional params — no new struct type (Validation Session 1,
Q3). `TypingModel` carries the two bools the same way it already carries
`strict`, so ctrl+r restart is symmetric with existing behavior by
construction, not a special case.

## Related Code Files

- Modify: `internal/words/for_mode.go` (signature + ModeWords/ModeTime branch)
- Modify: `internal/words/for_mode_test.go` (extend for new params — TDD: RED
  first)
- Modify: `internal/runner/session.go` (`NewSession` signature; `NewCodeSession`
  untouched)
- Modify: `internal/runner/session_test.go`
- Modify: `internal/ui/screen_typing.go` (`NewTyping`, `newTypingWithSeed`
  signatures; `TypingModel` struct gains `punctuation`, `numbers` fields)
- Modify: `internal/ui/screen_typing_test.go`
- Modify: `internal/ui/screen_typing_actions.go` (ctrl+r restart call at
  line ~30: `newTypingWithSeed(m.mode, m.length, m.ql, m.th, m.keys, m.blink,
  m.strict, ...)` → add `m.punctuation, m.numbers`)
- Modify: `internal/ui/screen_typing_actions_test.go`
- Modify: `internal/ui/phase09_polish_test.go` (calls these constructors —
  verify exact call before editing)
- Modify: `internal/app/anim_driver_test.go` (calls these constructors —
  verify exact call before editing)
- Modify: `internal/app/model.go` (StartTestMsg handler, `ui.NewTyping(...)`
  call only — NOT `ui.NewTypingCode(...)`)
- Modify: `internal/cli/cmd_run_notui.go` (`runSession` func,
  `runner.NewSession(...)` call only — NOT `runner.NewCodeSession(...)`)
- Do NOT modify: `internal/cli/cmd_run_validate.go` (verified not a call site
  — only computes `runLengthForMode`, unrelated to this chain)

## Implementation Steps

1. **RED**: Extend `for_mode_test.go` — add cases asserting `ForMode` with
   `punctuation=true`/`numbers=true` on `ModeWords` and `ModeTime` produces
   transformed output (reuse phase 1's `ApplyOptions` test assertions at this
   integration level), and `ModeQuote`/default fallback ignores the flags
   entirely (byte-identical to pre-change behavior for a fixed seed+quote index).
2. Confirm RED (signature mismatch compile failure is expected first).
3. **GREEN**: Update `ForMode` signature and implementation.
4. **GREEN**: Update `runner.NewSession` signature + its `words.ForMode(...)`
   call; update `session_test.go` call sites.
5. **GREEN**: Update `ui.NewTyping`/`newTypingWithSeed` signatures, add
   `punctuation`/`numbers` fields to `TypingModel`, update
   `screen_typing_test.go`.
6. **GREEN**: Update `screen_typing_actions.go`'s ctrl+r restart call to pass
   `m.punctuation, m.numbers`; update `screen_typing_actions_test.go`.
7. **GREEN**: Update `app/model.go`'s `ui.NewTyping(...)` call (StartTestMsg
   handler) to pass `m.settings.Punctuation, m.settings.Numbers`.
8. **GREEN**: Update `cli/cmd_run_notui.go`'s `runSession` func's
   `runner.NewSession(...)` call to pass `settings.Punctuation,
   settings.Numbers`.
9. **GREEN**: Run `go build ./...` — compile errors surface any remaining
   missed call site immediately (signature change is a forcing function);
   fix `phase09_polish_test.go` and `anim_driver_test.go` call sites as
   surfaced by the build.
10. Run full suite: `go test ./... -race -count=1` — confirm no regressions.
11. `gofmt -l .` and `go vet ./...` clean.

## Success Criteria

- [x] `ForMode`, `NewSession`, `NewTyping`/`newTypingWithSeed` all accept and
      correctly thread `punctuation, numbers bool`
- [x] `TypingModel` carries `punctuation`/`numbers`; ctrl+r restart preserves
      them (verified by a restart-path test asserting fields survive)
- [x] `NewCodeSession`/`NewTypingCode` signatures UNCHANGED (Code mode excluded)
- [x] `cmd_run_validate.go` untouched; no new `cmd_run.go` flags added
- [x] `go build ./...` compiles clean (proves no call site was missed)
- [x] `ModeQuote`/`ModeCode` paths byte-identical to pre-change
- [x] Full `go test ./... -race -count=1` green
- [x] `gofmt -l .` and `go vet ./...` clean

## Risk Assessment

- **Risk:** A forgotten call site silently keeps old behavior (masks the
  feature) instead of failing loudly. The original plan draft missed 2 real
  call sites (`screen_typing_actions.go` restart, `cmd_run_notui.go`) and
  named a wrong file (`cmd_run_validate.go`) — caught only by tracing the
  actual call chain during validation (Validation Session 1), not by grep.
  **Mitigation:** Signature change forces a compile error at every call site —
  `go build ./...` catches all of them mechanically; Step 9 explicitly budgets
  for fixing test files surfaced by the build rather than assuming the
  Related Code Files list above is exhaustive.
- **Risk:** Accidentally routing the transform into `ModeQuote`/`ModeCode`.
  **Mitigation:** Transform application lives inside `ForMode`'s existing mode
  switch, gated to the same branch that already returns `TimeBuffer`/`Words`
  output — not a wrapper applied unconditionally at call sites.
- **Risk:** ctrl+r restart silently drops the setting (regression vs.
  StrictMode's existing restart-preserving behavior).
  **Mitigation:** `TypingModel` stores `punctuation`/`numbers` fields
  explicitly (Validation Session 1, Q2); a dedicated test asserts they survive
  restart.
