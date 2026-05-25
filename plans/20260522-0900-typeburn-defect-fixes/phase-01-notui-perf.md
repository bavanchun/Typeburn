---
phase: 1
title: "notui-perf"
status: pending
priority: P1
effort: 1h
dependencies: []
---

# Phase 1: notui-perf (MEDIUM-1)

## Overview

`internal/cli/notui/runner.go:50` calls `metrics.Compute(session.Engine.Log(), session.Mode, nowMs)` on **every keystroke** purely to read `.NetWPM` for the live status line. `Compute` replays the entire keystroke log (O(n)) and additionally calls `TrimAFK` + `bucketPerSecond` + `Consistency` — heavy, fully wasted work for a live counter. Per keystroke this is O(n); across a test it is O(n²). On long Time-mode runs this causes visible lag in `--no-tui`.

The interactive TUI already avoids this: `internal/ui/screen_typing.go:108` uses the cheap unexported `liveWPM` helper (`internal/ui/typing_log_helpers.go:8`). notui **cannot** import that helper — `internal/cli/notui` importing `internal/ui` would violate the strict dependency layering (and create a cycle risk).

**Fix:** move the live-WPM formula *down* into the pure-logic `internal/metrics` package as an exported `LiveWPM`, then have both `internal/ui` (delegate) and `internal/cli/notui` (direct call) consume it.

## Requirements

### Functional
- notui live status WPM must match the value `liveWPM` produced before (identical formula).
- `LiveWPM` returns 0 when `elapsedMs < 500` OR `len(log) == 0` (noise guard, preserves current behavior including the `elapsed <= 0` case at `runner.go:51-53`).
- Counts only forward keystrokes (`Typed != 0`), same as `liveWPM` and consistent with `Compute`'s `totalTyped` definition.
- Final-result computation in notui (`runner.go:56`) is unchanged — still uses full `metrics.Compute`.

### Non-functional
- O(n) per keystroke replaced by a single O(n) sum with no AFK trim / per-second bucketing / consistency math → constant-factor and asymptotic win (no longer reruns `Compute`'s extra passes).
- No new dependency. `internal/metrics` stays UI-free.
- New file `live_wpm.go` keeps `compute.go` under 200 LOC (currently 163).

## Architecture

### Current data flow (broken)
```
notui keystroke loop
  └─ metrics.Compute(log, mode, nowMs)   // O(n): replay + AFKTrim + bucket + consistency
        └─ .NetWPM                        // only field used; rest discarded
```

### Target data flow
```
internal/metrics/live_wpm.go
  └─ func LiveWPM(log []typing.Keystroke, elapsedMs int64) float64   // single O(n) pass

internal/ui/typing_log_helpers.go  → liveWPM delegates to metrics.LiveWPM
internal/cli/notui/runner.go       → metrics.LiveWPM(log, elapsed)
```

`elapsed` is already computed at `runner.go:49` (`elapsed := nowMs - session.Engine.StartMs()`). Note: when `StartMs()==0` (no keystroke yet) `elapsed` would be huge/garbage, but the loop only reaches line 49 *after* an `Apply`/`Backspace` set `startMs`; and `LiveWPM`'s `len(log)==0` + `<500` guards make a pre-start call return 0 anyway. The existing `if elapsed <= 0 { live = 0 }` guard (lines 51-53) becomes redundant and is removed.

### Formula (verbatim from `liveWPM`, `typing_log_helpers.go:8-19`)
```
if elapsedMs < 500 || len(log) == 0 { return 0 }
forward = count of k where k.Typed != 0
return forward / 5.0 / (elapsedMs / 60000.0)
```

## Related Code Files

**Create**
- `internal/metrics/live_wpm.go` — exported `LiveWPM`.
- `internal/metrics/live_wpm_test.go` — table-driven `TestLiveWPM_*`.

**Modify**
- `internal/ui/typing_log_helpers.go` — `liveWPM` body delegates to `metrics.LiveWPM`; add `metrics` import.
- `internal/cli/notui/runner.go` — replace line 50 + remove guard lines 51-53.

**Delete**
- None.

## Implementation Steps

1. **Create `internal/metrics/live_wpm.go`:**
   ```go
   package metrics

   import "github.com/bavanchun/Typeburn/internal/typing"

   // LiveWPM estimates current net WPM from forward keystrokes in the log.
   // Returns 0 when elapsedMs < 500ms (too noisy) or the log is empty.
   // This is the cheap O(n) live-display estimate; full metrics come from Compute.
   func LiveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
       if elapsedMs < 500 || len(log) == 0 {
           return 0
       }
       var forward int
       for _, k := range log {
           if k.Typed != 0 {
               forward++
           }
       }
       return float64(forward) / 5.0 / (float64(elapsedMs) / 60000.0)
   }
   ```

2. **Delegate in `internal/ui/typing_log_helpers.go`:** replace the `liveWPM` body with a one-line delegate, keep the unexported wrapper so existing call site `screen_typing.go:108` and the test comment at `screen_typing_test.go:235` stay valid (no churn in ui callers).
   ```go
   import (
       "github.com/bavanchun/Typeburn/internal/metrics"
       "github.com/bavanchun/Typeburn/internal/typing"
   )

   func liveWPM(log []typing.Keystroke, elapsedMs int64) float64 {
       return metrics.LiveWPM(log, elapsedMs)
   }
   ```
   Verify no import cycle: `internal/ui` already depends on `internal/metrics` (e.g. `screen_result.go:7` imports `metrics.Result`), so this edge exists — safe.

3. **Fix `internal/cli/notui/runner.go`:** replace lines 50-53.
   ```go
   // before
   live := metrics.Compute(session.Engine.Log(), session.Mode, nowMs).NetWPM
   if elapsed <= 0 {
       live = 0
   }
   // after
   live := metrics.LiveWPM(session.Engine.Log(), elapsed)
   ```
   `metrics` is already imported (`runner.go:11`). `elapsed` already in scope (line 49). `session.Mode` no longer needed for the live call but is still used by `completed`/`endMs` — keep the import and field usage as-is.

4. **Create `internal/metrics/live_wpm_test.go`** — table-driven (repo convention: real data, no mocks). Cover:
   - `empty_log` → 0
   - `below_500ms_guard` (elapsedMs=400, some keystrokes) → 0
   - `zero_elapsed` (elapsedMs=0) → 0 (subsumes old `elapsed<=0` guard)
   - `negative_elapsed` (elapsedMs=-100) → 0
   - `forward_only` (e.g. 10 forward keystrokes, 60000ms → 10/5/1 = 2.0 WPM) exact value
   - `with_backspaces` — backspace entries (`Typed==0`) excluded from count; assert count matches forward-only.
   Build keystrokes as `[]typing.Keystroke{{Typed:'a'},{Typed:0},...}` — struct verified at `internal/typing/engine.go:11` (`{TimeMs, Typed, Target, Correct}`).

5. **Optional consistency check (no behavior change):** assert `LiveWPM(log, elapsedMs)` for a forward-only log within a single 1s bucket approximates `Compute(...).NetWPM` when all keystrokes are correct. Keep loose (formulas differ on correctness weighting) — informational only; skip if it adds flakiness.

6. Run `gofmt -w`, `go vet ./...`, `go test ./internal/metrics/ ./internal/ui/ ./internal/cli/notui/ -race -count=1`, then full `make test-race`.

7. Update `docs/codebase-summary.md` `internal/metrics` entry to mention `LiveWPM` (public surface change).

## Success Criteria

- [ ] `internal/metrics/live_wpm.go` exports `LiveWPM` with the verified formula.
- [ ] `internal/ui` `liveWPM` delegates; `screen_typing.go:108` unchanged and compiles.
- [ ] `runner.go` no longer calls `metrics.Compute` in the per-keystroke loop; only the completion branch (line 56) calls `Compute`.
- [ ] Old `if elapsed <= 0` guard removed; `LiveWPM` covers it.
- [ ] `TestLiveWPM_*` pass, including backspace-exclusion and guard cases.
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.
- [ ] `internal/metrics` still has no UI imports; no import cycle introduced.
- [ ] `compute.go` and `live_wpm.go` each < 200 LOC.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Import cycle ui↔metrics | Low | High (won't compile) | ui→metrics edge already exists (`metrics.Result` used in ui); only adding a call, not a new direction. Verify with `go build ./...`. |
| Behavior drift in live WPM display | Low | Low (cosmetic) | Formula copied verbatim; delegate keeps ui identical; notui now matches ui. |
| notui `elapsed` garbage when `StartMs()==0` | Low | Low | `len(log)==0` guard is the protection — NOT the loop entry condition. `EventNone` and no-op Backspace (`engine.go:77`: returns if `len(typed)==0`) both reach line 49 with `startMs==0`; `LiveWPM`'s `len(log)==0` guard returns 0 regardless. |
| `screen_typing_test.go:235` comment references 400ms→0 | Low | Low | Behavior preserved (400ms < 500 guard intact); test still valid. |

### Rollback
`git revert` the phase commit. notui reverts to calling `Compute`; ui `liveWPM` reverts to inline formula; `metrics.LiveWPM` removed. No other phase imports `LiveWPM`.
