---
phase: 2
title: "Typing engine & metrics (TDD)"
status: pending
priority: P1
effort: ~7h
dependencies: [1]
---

# Phase 2: Typing engine & metrics (TDD)

## Overview

Pure, UI-free typing engine + metrics. **Tests first** for every formula and state transition. The engine maintains target buffer, cursor, char-state, keystroke log; metrics derive everything post-hoc from the log + per-second snapshots. No Bubble Tea imports in either package.

Refs: researcher-02 (all sections — formulas, data model, AFK); design §5.2 (char states).

## Requirements

### Functional
- Char-state classify: `untyped, correct, incorrect, incorrect-space, extra, current` (design §5.2). `current-error` NOT used in v1 (allow-continue only) — note in code comment.
- Apply printable keystroke; `backspace` deletes last typed rune (within current word / buffer; allow-continue).
- Completion detection per mode: Time (caller signals timeout), Words (N words typed), Quote (full text matched).
- Per-keystroke log `{time_ms, typed, target, correct}`; per-second raw-WPM snapshots.
- Metrics: NetWPM, RawWPM, Accuracy%, Consistency, error count, CPS, duration — all derived post-hoc.
- Clock starts on FIRST keystroke. AFK trailing-trim (>7s gap at end) ONLY in Time mode.

### Non-functional
- Zero UI deps; deterministic; rune-safe (iterate runes, multi-byte safe).
- Files <200 lines; table-driven tests; no mocks/fakes.

## Architecture

Data flow: caller feeds runes/backspace + monotonic timestamps → `Engine` mutates buffer state + appends `Keystroke` → on completion `metrics.Compute(log, mode, duration)` → `Result`.

```go
// internal/typing
type CharState int
const ( Untyped CharState = iota; Correct; Incorrect; IncorrectSpace; Extra; Current )

type Keystroke struct { TimeMs int64; Typed, Target rune; Correct bool }

type Engine struct {
    target  []rune
    typed   []rune
    log     []Keystroke
    startMs int64   // 0 until first keystroke
    mode    Mode    // Time|Words|Quote
    wordTarget int   // Words mode
}
func New(target string, mode Mode, wordTarget int) *Engine
func (e *Engine) Apply(r rune, nowMs int64)        // sets startMs if first
func (e *Engine) Backspace(nowMs int64)
func (e *Engine) States() []CharState              // per target+extra index, cursor=Current
func (e *Engine) Complete(nowMs int64) bool        // mode-aware
func (e *Engine) Log() []Keystroke
func (e *Engine) Progress() (done, total int)      // for header

// internal/metrics
type PerSecond struct { Sec int; RawWPM float64; Errors, CorrectChars, TotalChars int }
type Result struct {
    NetWPM, RawWPM, Accuracy, Consistency, CPS float64
    CorrectChars, IncorrectChars, ExtraChars, MissedChars, Errors int
    DurationMs int64
    PerSecond  []PerSecond
}
func Compute(log []Keystroke, mode Mode, endMs int64) Result
func bucketPerSecond(log []Keystroke) []PerSecond
func consistency(raw []float64) float64 // 100*tanh(1-CV); CV=stddev/mean; clamp[0,100]
func trimAFK(log []Keystroke, mode Mode) ([]Keystroke, int64) // Time-only, >7s trailing
```

Final char-state for accuracy: last keystroke per target index decides correct/incorrect (corrected errors don't penalize). NetWPM numerator = correct chars in final state; RawWPM = all keystrokes. CPS = totalChars / (durationMs/1000). Consistency pinned to `100*tanh(1-CV)` — minor upstream coefficient uncertainty is an **accepted v1 decision**, documented in code, not an open question.

## Related Code Files

Create:
- `internal/typing/engine.go`, `internal/typing/char-state.go`, `internal/typing/completion.go`
- `internal/typing/engine_test.go`, `internal/typing/char-state_test.go`
- `internal/metrics/compute.go`, `internal/metrics/per-second.go`, `internal/metrics/consistency.go`, `internal/metrics/afk-trim.go`
- `internal/metrics/compute_test.go`, `internal/metrics/consistency_test.go`, `internal/metrics/afk-trim_test.go`
- `internal/typing/mode.go` (Mode enum, shared)

Modify: none. Delete: none.

## Implementation Steps

1. **Write tests FIRST** (`*_test.go`) — they must fail/compile-error before impl:
   - char-state table: each of 6 states from a crafted typed/target pair; cursor index → Current; extra runes past word → Extra; wrong char at space slot → IncorrectSpace.
   - backspace: removes last typed rune; log still records the deletion-causing keystrokes; corrected error → final Correct.
   - completion: Words (type exactly N words → Complete true at Nth), Quote (full text), Time (Complete only when caller passes endMs ≥ limit).
   - metrics formulas: NetWPM=correct/5/min, RawWPM=all/5/min, Accuracy=100·c/(c+i) on final state (include "corrected error → 100%" case), CPS, duration, error count.
   - **consistency worked example** (researcher-02 §3): per-sec raw `[80,85,82,78,90]` → mean=83, stddev≈4.2, CV≈0.051, `100*tanh(1-0.051)` ≈ 74 (±1 tolerance). Add a uniform-samples case → consistency≈100; high-variance case → low.
   - AFK trim: Time mode with >7s trailing gap → trailing seconds removed, endMs adjusted to last keystroke; Words/Quote mode → NO trim even with gap.
   - clock-start: first Apply sets startMs; pre-first-keystroke duration = 0.
2. Implement `typing` (engine, char-state, completion, mode) to pass typing tests. Rune slices only.
3. Implement `metrics` (per-second bucket [0,1)…, compute, consistency, afk-trim) to pass metrics tests.
4. `go test ./internal/typing/... ./internal/metrics/... -race` → all green.
5. `go build ./...`, `go vet`, `gofmt -l`.

## Success Criteria

- [ ] Tests written before implementation (commit order or note in PR).
- [ ] All 6 char-states correctly classified incl. extra & incorrect-space.
- [ ] Backspace + corrected-error → final accuracy 100% for that char.
- [ ] Consistency worked example yields ≈74 (±1); formula pinned `100*tanh(1-CV)` with code comment on accepted uncertainty.
- [ ] AFK trim applies Time-only; Words/Quote untouched.
- [ ] `go test ./... -race` passes; build/vet/gofmt clean.
- [ ] Zero Bubble Tea imports in `typing`/`metrics`.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Consistency coefficient wrong vs upstream | M×L | Pinned & documented as accepted v1 decision; table test locks behavior |
| Per-second bucket boundary ambiguity | M×M | Define [0s,1s) half-open, document; test with on-boundary keystroke |
| Rune vs byte bug on multi-byte input | L×H | Rune slices throughout; explicit multi-byte test case |
| AFK trim leaking into non-Time modes | L×M | Mode guard + explicit Words/Quote negative tests |
