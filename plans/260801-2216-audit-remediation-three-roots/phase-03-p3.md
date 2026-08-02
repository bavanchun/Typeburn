---
phase: 3
title: "Metrics And Typing Correctness"
status: done
priority: P1
effort: "1.5d"
dependencies: [1]
---

# Phase 3: Metrics And Typing Correctness

## Overview

Stop impossible results being computed and persisted. Covers the AFK policy, the
zero-duration fabrication, the strict-mode replay desync, and the `Progress()`
semantics — everything where the *number* is wrong, as opposed to where an input
is unbounded (Phase 4) or a write races (Phase 5).

## Requirements

- Functional
  - No physically impossible metric is persisted or counted as a best.
  - AFK-trimmed runs are withheld from history and best, **visibly**.
  - Strict mode's replay agrees with the engine's actual buffer.
  - `Progress()` returns a word count in every mode.
- Non-functional
  - No formula changes beyond the partial-final-second correction.
  - `internal/typing/engine.go` stays under 200 LOC (it is at **197** — the split
    is part of this phase, not an afterthought).

## Architecture

### AFK policy — user-confirmed: withhold, do not rescale

`TrimAFK` (`afk_trim.go:21`) moves `endMs` back to the last forward keystroke;
nothing floors the result, so `netWPM = chars/5/minutes` divides by the *burst*:

```
30s Time test, 3 keys at t=1000/1050/1100, then AFK:
  DurationMs=100  NetWPM=360.0  RawWPM=360.0
5 keys at 1ms spacing: DurationMs=4, NetWPM=3000, RawWPM=15000
```

Chain traced: `model_history.go:49` persists unconditionally; `EligibleForBest`
(`new_best.go:6`) excludes only `code` and `strict`, so a Time-mode AFK burst is
eligible and becomes a permanent best in `time/30`.

`metrics.Result` gains `AFKTrimmed bool`, set by `Compute` when `TrimAFK` actually
moved `endMs`. `EligibleForBest` returns false for it; `handleResultMsg` skips
`AppendHistory`.

**The withholding must be visible.** Silently discarding a result is worse than
the bug being fixed — the user would think the app lost their run. Reuse the
persistence-notice seam with copy like `paused mid-test — not saved to history`.

**Fix the notice's placement here**, since this phase makes it load-bearing:
`model_view.go:62-69` writes into `lines[len(lines)-1]`, which at h≤24 is a
clipped row (invisible exactly when it matters) and at h≥30 is the *footer*
(destroying the keybindings). Overlay at `min(len(lines)-1, m.h-1)` and require
that row to be blank, else insert.

**Coordination with Phase 6:** a notice that needs its own row costs Result a row
it does not have. Phase 6's budget gate accounts for this; do not add a dedicated
row without telling that phase.

### Zero duration fabricates 100%

`compute.go:52-57`: after trimming, `startMs == endMs`, so `durationMs <= 0`
returns `Result{Accuracy: 100, KeystrokeAccuracy: 100}`.

```
target "hello world", ONE wrong key 'q' at t=1000, end t=16000:
  DurationMs=0  Accuracy=100.0  KeystrokeAcc=100.0
```

Via `new_best.go:86-88` (`best := -1.0`) that becomes the first-ever best. Report
the keystroke-level truth (0% here) or mark the result invalid. The
`len(log) == 0` case may legitimately stay 100% — nothing was typed wrong. The
zero-*duration* case may not.

### Strict-mode replay desync — two sites, not one

`compute.go:142-166` vs `engine.go:72-81`: a strict-blocked keystroke is appended
to the log with `Typed != 0` but is NOT applied to `e.typed`. `replayFinalState`
skips only `Typed == 0`, so blocked keystrokes enter its reconstruction;
backspaces then pop slots the engine never had.

```
target "abcdef" strict: abc, z×5 (blocked at pos 3), backspace×3, retype abcdef
  engine final text = "abcdef" exactly
  metrics: CorrectChars=9 IncorrectChars=2 NetWPM=67.50 Accuracy=81.82
  non-strict, same pattern: CorrectChars=6 (correct)
```

Fix: `Blocked bool` on `typing.Keystroke`, set at `engine.go:72`, skipped in
`replayFinalState` while still counted in `totalTyped`, `correctForward`, and
`KeyHeatmap`.

**Red-team found a second desync site the original plan missed:**
`internal/ui/typing_log_helpers.go:18 typedFromLog` is a byte-for-byte
reimplementation of the same buggy loop (`if k.Typed == 0 { pop } else { append }`).
It will push blocked keystrokes into its buffer too. **This phase owns that file**
— the fix is incomplete without it. Consider collapsing the duplicate rather than
fixing it twice; that is a DRY violation that caused this exact double-bug.

`Keystroke` is serialised into replay JSON — add the field with `omitempty` so
existing logs still parse to `false`, i.e. today's behaviour.

### `Progress()` returns milliseconds as a word count

`engine.go:182-196`: `runner.wordTarget` (`session.go:53-57`) stores `length*1000`
for `ModeTime`, and `Progress`'s `default` branch returns it verbatim. A 30s
engine reports `(0, 30000)`. `ModeCode` returns `(words, 0)` — a zero denominator.

**Cannot be fixed by changing `wordTarget`:** `completion.go:21` uses it as the
Time deadline in ms. Fix `Progress()`'s Time branch instead, which changes what
these display:

```
internal/ui/screen_typing_view.go:25   ← Phase 2 owns this file
internal/ui/mode_header.go:23-30       ← this phase owns it
internal/cli/notui/runner.go:117       ← this phase owns it
```

`screen_typing_view.go:25` is the collision: Phase 2 owns it and runs in the same
group. **Resolution: this phase does not touch that file.** It changes
`Progress()`'s contract and updates `mode_header.go` and `notui/runner.go`; Phase 2
adapts its one call site as part of its own viewport work. Both phases must know
this — see open question 4 on whether the *displayed* value may change.

### Consistency: partial final second

`per_second.go:42,71` size buckets by the last keystroke, then scale every bucket
as a full second. A *perfectly even* 5 c/s typist over 3.2 s scores **51.31**
against a formula maximum of 76.16. Drop or prorate the final bucket below 1000 ms.

**This changes every user's consistency score.** Document it in the CHANGELOG as
an intentional correction with a worked before/after; do not migrate stored history.

### Also here (same files)

- Strict does not block runes past the end of the target (`engine.go:72` guard is
  `pos < len(e.target)`): strict `"ab"`, type `ab` then `z`×10 → `ExtraChars=10`.
- `completion.go:46,53,65,67` dead `_ = wordStart`; `:39-40` false comment.

### The cross-package assertion (R1's actual point)

The audit's C9 finding is that the AFK defect spans
`TrimAFK → Compute → Record.NetWPM → IsNewBest → AppendHistory` — four packages —
and every test is within-package. Phase 1's `assertPlausible` lives in
`internal/metrics`, which is *also* within-package and does not close it.

**This phase adds the seam test** in `internal/app`: construct an AFK-shaped
`ResultMsg`, run it through `handleResultMsg`, and assert nothing was persisted
and no best was recorded. That is the assertion whose absence let the defect ship.

## Related Code Files

- Modify: `internal/metrics/afk_trim.go`, `compute.go`, `per_second.go`
- Modify: `internal/typing/engine.go` (+ split — it is at 197 LOC), `completion.go`
- Modify: `internal/ui/typing_log_helpers.go`, `internal/ui/mode_header.go`
- Modify: `internal/cli/notui/runner.go`, `internal/runner/session.go`
- Modify: `internal/storage/new_best.go`
- Modify: `internal/app/model_history.go`, `internal/app/model_view.go`
- Create: `internal/app/result_persistence_test.go` (the cross-package seam)
- Modify: `internal/metrics/plausibility_test.go` (delete the AFK allowlist entry)

## Implementation Steps

1. Reproduce each defect as a failing test first, using the inputs above.
2. `AFKTrimmed` → `EligibleForBest` → persistence gate → visible notice, and fix
   the notice's row placement.
3. Zero-duration path reports truth.
4. `Blocked` marker; fix **both** `replayFinalState` and `typedFromLog`, or
   collapse them into one.
5. `Progress()` Time branch + the two display sites this phase owns.
6. Consistency partial-bucket correction; capture before/after numbers for the
   CHANGELOG.
7. Past-end strict guard; dead-code cleanup.
8. Cross-package seam test in `internal/app`.
9. Split `engine.go` under 200 LOC.

## Success Criteria

- [ ] Every defect has a test verified to fail against current code
- [ ] AFK run: not persisted, not a best, notice **visible at 80×24**
- [ ] Cross-package seam test in `internal/app` covers `TrimAFK → … → AppendHistory`
- [ ] Single wrong keystroke no longer reports 100%
- [ ] Strict and non-strict agree on `CorrectChars`; `typedFromLog` agrees with the engine
- [ ] Old replay fixtures (no `Blocked` field) still parse and behave as before
- [ ] `Progress()` returns a word count in Time and Code modes
- [ ] Consistency before/after recorded for a worked example
- [ ] `engine.go` and every touched file under 200 LOC
- [ ] `assertPlausible` passes with zero allowlist entries
- [ ] `go test ./... -race -count=1` green

## Risk Assessment

**Risk:** withholding AFK runs silently — user believes data was lost.
**Mitigation:** the visible notice is a success criterion, and its placement bug
is fixed in the same phase.

**Risk:** `Progress()`'s contract change collides with Phase 2 in
`screen_typing_view.go`.
**Mitigation:** explicit split of ownership above — this phase does not touch
that file; Phase 2 adapts its call site.

**Risk:** the consistency correction reads as a regression to users.
**Mitigation:** CHANGELOG documents it as a correction with numbers; stored
history untouched.

**Risk:** `Blocked` breaks replay of older logs.
**Mitigation:** `omitempty`; absent unmarshals to `false` = old behaviour. Test
against a fixture written before the change.

## Rollback

Revert the commit — but note **Phase 6 edits `model_history.go` on top of this
phase**, so a revert after Phase 6 merges will conflict. If this phase must be
rolled back late, revert Phase 6 first or cherry-pick the inverse of the
eligibility gate only. Behaviour changes (consistency values, AFK withholding)
are not feature-flagged; that is deliberate, but it means rollback is all-or-nothing.
