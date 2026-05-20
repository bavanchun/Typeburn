---
phase: 1
title: "Extract internal/runner shared session driver"
status: completed
priority: P1
effort: "4h"
dependencies: []
---

# Phase 1: Extract internal/runner shared session driver

## Overview

Move the session-construction logic currently in `internal/ui/screen_typing.go:newTypingWithSeed`
into a new pure-logic package `internal/runner`. This is a behavior-preserving refactor.
TDD: write `internal/runner/session_test.go` first; TUI continues to pass `teatest` goldens.

## Requirements

- Functional: TUI tests still pass; identical session for given (mode, length, ql, seed)
- Non-functional: `internal/runner` has no UI imports (no `bubbletea`, no `lipgloss`)

## Architecture

```
internal/runner/
  session.go        # NewSession, NewCodeSession; tiny (~80 LOC)
  session_test.go   # table-driven, deterministic via fixed seeds
```

```go
type Session struct {
    Engine   *typing.Engine
    Target   string
    Mode     config.Mode
    Length   int
    QuoteLen words.QuoteLen
    CodeText string  // ModeCode only
}

func NewSession(mode config.Mode, length int, ql words.QuoteLen, seed int64) Session
func NewCodeSession(snippet string) Session

// RebuildEngine recomputes the wordTarget math and returns a fresh engine
// for the SAME target text (used by ctrl+r restart paths).
func RebuildEngine(target string, mode config.Mode, length int) *typing.Engine
```

`internal/ui/screen_typing.go:newTypingWithSeed` calls `runner.NewSession` and wraps with UI state.
`internal/ui/screen_typing_actions.go:restartSame` calls `runner.RebuildEngine(...)` to eliminate
the duplicate `wordTarget = length*1000 if Time` math currently inlined there.

## Related Code Files

- Create: `internal/runner/session.go`, `internal/runner/session_test.go`
- Modify: `internal/ui/screen_typing.go` (delegate to `runner.NewSession`)
- Modify: `internal/ui/screen_typing_code.go` (delegate to `runner.NewCodeSession`)
- Modify: `internal/ui/screen_typing_actions.go` (`restartSame` and `newTest` must call `runner.RebuildEngine` instead of inlining `length*1000 if Time` math — eliminates known-duplication bug source)

## Implementation Steps

1. Write `internal/runner/session_test.go` with table cases: (mode, length, ql, seed) → expected target prefix + wordTarget. Use existing seeds known from `screen_typing_test.go` to lock behavior.
2. Implement `internal/runner/session.go` with `NewSession` and `NewCodeSession`. Move the wordTarget math (`length*1000` for Time) here. No I/O.
3. Refactor `internal/ui/screen_typing.go:newTypingWithSeed` to `runner.NewSession(...)` then attach UI fields. Same for `NewTypingCode`.
4. Run `go test ./internal/runner/ -race -count=1` — must pass.
5. Run `go test ./... -race -count=1` — TUI teatest goldens must still pass byte-identical.
6. `go vet ./...` + `gofmt -l .` clean.

## Success Criteria

- [ ] `internal/runner/session.go` created, ≤120 LOC, no UI imports
- [ ] `internal/runner/session_test.go` table-driven; covers Time, Words, Quote, Code modes + `RebuildEngine`
- [ ] `internal/ui/screen_typing.go` shorter (~40 LOC removed)
- [ ] `internal/ui/screen_typing_actions.go` `restartSame` + `newTest` no longer inline `length*1000` math; delegate to `runner.RebuildEngine`
- [ ] grep confirms `length * 1000` literal does not exist outside `internal/runner/`
- [ ] All existing tests pass unchanged (teatest goldens byte-identical)
- [ ] `go test ./... -race -count=1` green
- [ ] `gofmt -l .` empty

## Risk Assessment

- **Risk:** `words.NewGenerator(seed)` non-determinism creeps in.
  **Mitigation:** Tests use fixed non-zero seeds; assert exact target string.
- **Risk:** Behavior drift in `newTypingWithSeed`'s seed=0 fallback.
  **Mitigation:** Preserve the `seed==0 → time-based` branch inside `NewSession`; existing TUI tests cover both paths.
