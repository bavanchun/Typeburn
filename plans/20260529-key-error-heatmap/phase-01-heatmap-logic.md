---
phase: 1
title: "Heatmap Logic"
status: completed
priority: P1
effort: "2h"
dependencies: []
---

# Phase 1: Heatmap Logic

## Overview

Add a pure per-key miss tally to `internal/metrics`, exposed via a new
`KeyMisses []KeyMiss` field on `metrics.Result`, populated inside `Compute`.
This is the foundation every surface (Phase 2, 3) reads.

## Requirements

- **Functional:**
  - `KeyHeatmap(log []typing.Keystroke) []KeyMiss` tallies, per case-folded
    target rune: total attempts and total misses.
  - A miss = forward keystroke (`Typed != 0`) with a real target
    (`Target != 0`) where `!Correct`. Corrected fumbles still count.
  - Skip backspace events (`Typed == 0`) and extra chars (`Target == 0`).
  - Case-fold the target via `unicode.ToLower` so `a`/`A` merge into one key.
  - Return only keys with ≥1 miss, sorted deterministically.
  - `Compute` populates `Result.KeyMisses` from the full log.
- **Non-functional:**
  - No UI deps (stays pure-logic). Only stdlib `unicode` + `sort` added.
  - File < 200 LOC. Deterministic output (golden/table-test safe).

## Architecture

`Keystroke` already carries everything needed (`engine.go:11-16`):
```go
type Keystroke struct {
    TimeMs  int64
    Typed   rune   // 0 == backspace
    Target  rune   // 0 == extra (past target end)
    Correct bool
}
```

`KeyHeatmap` does a single pass over the log into a `map[rune]*KeyMiss`
keyed by folded target rune, then flattens + sorts. **Timing-independent** —
it runs over the full untrimmed log (AFK trim only affects duration/rate
metrics, not which keys were mistyped), so order vs `TrimAFK` in `Compute`
does not matter.

```go
type KeyMiss struct {
    Key      rune   `json:"-"`     // case-folded target rune
    Label    string `json:"key"`   // display label: "e", "t", "␣" for space
    Misses   int    `json:"misses"`
    Attempts int    `json:"attempts"`
}
```

Sort order (deterministic): `Misses` desc → `Attempts` desc → `Key` asc.

Label mapping (`keyLabel(r rune) string`):
- `' '` → `"␣"` (visible space glyph; ASCII-safe in any terminal? — `␣` is
  U+2423, renders in UTF-8 terminals. Mono/NO_COLOR unaffected, it is a glyph
  not a color. If ASCII-only fallback is wanted, that is a deferred follow-up.)
- printable rune → `string(r)`
- other control/whitespace (tab, newline — only reachable in Code mode) →
  Go-quoted form trimmed of quotes (e.g. `\t`).

In `Compute`, after the existing per-second/consistency block, add:
```go
return Result{
    ...
    KeyMisses: KeyHeatmap(log),
}
```
Note: `log` inside `Compute` is the AFK-trimmed slice; that is fine — trim only
drops *trailing idle* keystrokes (which by definition were not mistypes that
matter), and matches the keys the user actually engaged. Keep it simple: pass
the same `log` already in scope.

## Related Code Files

- Create: `internal/metrics/key_heatmap.go` (~70 LOC)
- Create: `internal/metrics/key_heatmap_test.go`
- Modify: `internal/metrics/compute.go` (add `KeyMisses` field to `Result`
  struct + one line in the returned literal)

## Implementation Steps

1. Create `internal/metrics/key_heatmap.go`:
   - `KeyMiss` struct (with JSON tags as above).
   - `keyLabel(r rune) string` helper.
   - `KeyHeatmap(log []typing.Keystroke) []KeyMiss`: single-pass tally → flatten
     → `sort.Slice` with the deterministic comparator → return.
2. In `internal/metrics/compute.go`:
   - Add `KeyMisses []KeyMiss` to the `Result` struct (document it in the
     struct comment block alongside the other fields).
   - Populate it in the returned `Result` literal: `KeyMisses: KeyHeatmap(log)`.
   - The early-return paths (`len(log)==0`, `durationMs<=0`) leave `KeyMisses`
     nil — correct (no misses to show).
3. Create `internal/metrics/key_heatmap_test.go` (table-driven, real data):
   - Empty log → nil.
   - Only-correct log → nil (no misses).
   - Corrected fumble (wrong → backspace → right) still counts as 1 miss.
   - Case folding: errors on `a` and `A` merge into key `a`.
   - Extra chars (`Target==0`) and backspaces (`Typed==0`) excluded.
   - Space miss → label `␣`.
   - Sort determinism: equal misses break by attempts desc then key asc.
   - Attempts counts all forward keystrokes against that target (correct + wrong).

## Success Criteria

- [ ] `KeyHeatmap` returns deterministic, correctly-sorted misses.
- [ ] Corrected errors counted; backspaces/extras excluded; case folded.
- [ ] `Result.KeyMisses` populated by `Compute`; early-returns leave it nil.
- [ ] `go test ./internal/metrics/ -race -count=1` GREEN.
- [ ] `gofmt -l internal/metrics/` empty; `go vet ./internal/metrics/` clean.
- [ ] `key_heatmap.go` < 200 LOC, no UI imports.

## Risk Assessment

- **Low.** Pure additive; no change to existing metric values. Existing
  `metrics` tests assert specific fields and will not break from a new field.
- **Glyph risk:** `␣` (U+2423) requires UTF-8 terminal. Acceptable — the whole
  TUI already assumes a modern terminal. ASCII fallback deferred.
