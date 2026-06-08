---
phase: 2
title: Typing Snapshot Optimization
status: completed
priority: P2
effort: 1.5h
dependencies:
  - 1
---

# Phase 2: Typing Snapshot Optimization

## Context Links

- Hot path: `internal/ui/screen_typing_view.go`
- Engine copy API: `internal/typing/engine.go`
- Replay helper: `internal/ui/typing_log_helpers.go`
- Live WPM path: `internal/ui/screen_typing.go`, `internal/metrics/live_wpm.go`

## Overview

Reduce typing-screen allocation/time only if Phase 1 proves a meaningful issue.
Expected fix is a small immutable snapshot API on `typing.Engine`, not a
renderer rewrite.

## Requirements

- Functional: rendered output and completion behavior unchanged.
- Non-functional: lower `TypingModel.View` and/or live WPM hot-path allocations
  for Code 10k or equivalent worst case.
- Guardrail: skip production optimization if benchmarks show current cost is
  already negligible.

## Architecture

Candidate approach:

```go
type Snapshot struct {
    States []CharState
    Typed  []rune
    Log    []Keystroke // optional; include only if needed
}

func (e *Engine) Snapshot() Snapshot
```

The snapshot may copy data to preserve encapsulation, but should avoid multiple
copies/replays per render. UI receives states + typed runes from one engine
call. If only typed runes are needed, prefer narrower `Typed()` plus existing
`States()`.

## File Inventory

| File | Action | Rough size | Test impact |
|---|---:|---:|---|
| `internal/typing/engine.go` | Modify | 161 LOC now | Add snapshot/typed API, keep <200 |
| `internal/typing/engine_test.go` | Modify | 187 LOC now | May need split to stay <200 |
| `internal/typing/snapshot_test.go` | Create | <120 LOC | Snapshot immutability/regression |
| `internal/ui/screen_typing_view.go` | Modify | 76 LOC | Use snapshot/typed API |
| `internal/ui/typing_log_helpers.go` | Modify/Delete helper | 30 LOC | Remove replay if obsolete |
| `internal/ui/screen_typing_test.go` | Modify | 256 LOC now | Consider new focused test file |
| `internal/ui/render_benchmark_test.go` | Modify | from Phase 1 | Before/after comparison |

## Test Scenario Matrix

| Scenario | Criticality | Protection |
|---|---|---|
| Snapshot cannot mutate engine internals | Critical | New unit test |
| Apply/backspace state unchanged | Critical | Existing engine tests |
| Word typing view unchanged semantically | High | Existing UI tests |
| Code typing view with newlines/tabs unchanged | High | Existing code renderer tests |
| Paste multi-rune behavior unchanged | Medium | Existing paste tests |
| Benchmarks improve or justify skip | Critical | Phase 1 benchmarks |

## Dependency Map

```text
typing.Engine snapshot API
└── ui.TypingModel.View consumes snapshot
    ├── RenderWordStream unchanged
    └── RenderCodeStream unchanged
```

## Implementation Steps

1. Review Phase 1 benchmark output.
2. If no meaningful bottleneck, mark this phase as skipped/deferred in plan
   notes and do not change code.
3. If bottleneck exists, write tests first:
   - snapshot/typed API returns expected runes after apply/backspace
   - modifying returned slices does not mutate engine
4. Add the narrowest engine API needed:
   - prefer `Typed()` if only replay removal is needed
   - use `Snapshot()` only if it removes multiple hot-path calls
5. Update `TypingModel.View()` to avoid `typedFromLog(m.eng.Log())`.
6. Remove or limit `typedFromLog` if no longer used by production.
7. Re-run benchmarks from Phase 1 and compare before/after.
8. Run targeted tests:
   ```sh
   go test ./internal/typing ./internal/ui -count=1
   ```

## Success Criteria

- [ ] Production optimization is backed by benchmark data, or explicitly skipped.
- [ ] No render behavior regressions.
- [ ] Returned engine data cannot mutate engine internals.
- [ ] Benchmark before/after shows lower allocations or time for the measured
      hot path if code changed.
- [ ] `go test ./internal/typing ./internal/ui -count=1` passes.

## Risk Assessment

- Risk: overexposing engine internals. Mitigation: copies or immutable contract
  with tests.
- Risk: optimizing away useful log API consistency. Mitigation: keep `Log()` for
  metrics/post-hoc replay; add narrow UI helper only.
- Risk: file-size violation in `engine.go` or tests. Mitigation: create
  `snapshot_test.go` and split helpers.
