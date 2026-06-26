---
phase: 1
title: "Engine strict-letter policy + keystroke-accuracy (TDD)"
status: completed
priority: P1
dependencies: []
---

# Phase 1: Engine strict-letter policy + keystroke-accuracy (TDD)

## Overview

Pure-logic core of the feature. Teach `typing.Engine` to block wrong keystrokes
in strict mode (cursor frozen, error still logged), and add a mode-agnostic
`KeystrokeAccuracy` to `metrics.Result` so blocked errors are honestly counted.
No UI, no settings, no wiring yet.

## Requirements

- Functional:
  - Engine carries a `strict` flag set at construction.
  - Strict + position has a target + wrong rune → append error keystroke
    (`Typed`=r, `Target`=target rune, `Correct=false`) to the log; do **not**
    append to the typed buffer (cursor stays).
  - Strict + correct rune → normal `Apply` (advance + log).
  - Non-strict → behavior byte-identical to today.
  - `metrics.Result.KeystrokeAccuracy` = `100 * correctForward / totalForward`
    over non-backspace log entries; `0` forward keystrokes → `100`.
- Non-functional: `internal/typing` and `internal/metrics` stay UI-free; files
  < 200 LOC; snake_case.

## Architecture

`internal/typing/engine.go` — add field and branch:

```go
type Engine struct {
    // ...existing...
    strict bool
}

// New keeps its current signature for non-strict callers; add a variadic
// option OR a dedicated constructor to avoid breaking the single call site.
// Prefer an explicit option to keep call sites readable:
func NewStrict(target string, m mode.Mode, wordTarget int, strict bool) *Engine { ... }
// and have New(...) delegate with strict=false (back-compat).
```

In `Apply`:

```go
func (e *Engine) Apply(r rune, nowMs int64) {
    if e.startMs == 0 { e.startMs = nowMs }
    pos := len(e.typed)
    var target rune
    var correct bool
    if pos < len(e.target) {
        target = e.target[pos]
        correct = (r == target)
    }
    if e.strict && pos < len(e.target) && !correct {
        // blocked: log the error, do NOT advance the typed buffer
        e.log = append(e.log, Keystroke{TimeMs: nowMs, Typed: r, Target: target, Correct: false})
        return
    }
    e.typed = append(e.typed, r)
    e.log = append(e.log, Keystroke{TimeMs: nowMs, Typed: r, Target: target, Correct: correct})
}
```

Edge cases to honor:
- **Overflow/Extra:** strict only gates positions with a target (`pos <
  len(e.target)`). At/after end, behave as today (extra runes allowed). A wrong
  key cannot reach overflow anyway because earlier errors are blocked.
- **Backspace:** unchanged. In strict the buffer holds only correct runes, so
  backspace deletes a correct char (allowed — lets the user go back).
- **Space-as-target:** a wrong non-space where a space is expected is still a
  mismatch → blocked, same rule.

`internal/metrics/compute.go` — add `KeystrokeAccuracy` to `Result` and compute
it from the log (total non-backspace keystrokes vs those with `Correct==true`).
Do **not** alter the existing final-state `Accuracy`.

## Related Code Files
- Modify: `internal/typing/engine.go` (strict field + Apply branch + constructor)
- Modify: `internal/typing/char_state.go` (update the NOTE comment — strict is now
  supported; keep `Current`/no `current-error` state since the cursor simply
  freezes)
- Modify: `internal/metrics/compute.go` (+`KeystrokeAccuracy` field + calc)
- Create/Modify tests: `internal/typing/engine_test.go` (or new
  `strict_engine_test.go`), `internal/metrics/compute_test.go`

## Implementation Steps (TDD)

1. **Red:** add engine tests:
   - strict rejects a wrong rune: `Typed()` length unchanged, `States()` shows
     `Current` at the same cursor, `Log()` gained one `Correct:false` entry.
   - strict accepts the correct rune afterward: cursor advances.
   - non-strict unchanged: existing tests still green (regression guard).
2. **Red:** add metrics test: a log with N forward keystrokes, k correct →
   `KeystrokeAccuracy == 100*k/N`; empty → 100.
3. **Green:** implement the engine `strict` field + `Apply` branch + constructor.
4. **Green:** implement `KeystrokeAccuracy`.
5. Run `go test ./internal/typing/ ./internal/metrics/ -race -count=1`.
6. **Commit** (conventional, e.g. `feat(typing): letter-strict keystroke
   blocking + keystroke accuracy metric`).
7. **On completion run `/vchun-git prc`** (branch `feat/strict-mode-p1-engine` →
   PR → CI green → squash-merge; never push main). Then
   `ck plan check phase-01-engine-strict-letter-policy-tdd`.

## Success Criteria
- [x] Strict engine blocks wrong runes (cursor frozen) and logs the error.
- [x] Correct rune advances; non-strict path byte-identical (regression tests green).
- [x] `metrics.Result.KeystrokeAccuracy` correct incl. empty-log → 100.
- [x] `internal/typing` + `internal/metrics` import no UI packages.
- [x] Phase committed; PR squash-merged via `/vchun-git prc`; CI green.

## Risk Assessment
- **Risk:** changing `New` signature breaks the call site. **Mitigation:** keep
  `New` delegating to a strict-aware constructor with `strict=false`; only
  `runner/session.go` changes (Phase 3).
- **Risk:** metrics field unused now → dead code until Phase 3. Acceptable; it is
  pure and tested. Keeps Phase 1 self-contained pure-logic.
- **Risk:** hidden assumptions in `metrics.Compute` replay about log/position
  coupling. **Mitigation:** `KeystrokeAccuracy` reads the log directly and does
  NOT touch the final-state replay, so existing `Accuracy` is unaffected.
