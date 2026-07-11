---
phase: 1
title: Code Mode Display and Best Marker TDD
status: completed
priority: P1
effort: 2h
dependencies: []
---

# Phase 1: Code Mode Display and Best Marker TDD

## Overview

Write regressions first, then replace two default-to-Quote formatters with one
UI formatter. Align History ★ rendering with the existing Code/Strict
personal-best exclusion without changing stored records or JSON.

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [Code/display research](./research/code-mode-display-and-best-marker.md)
- Bug sources: `internal/ui/history_table.go`, `internal/ui/result_render_helpers.go`

## Requirements

### Functional

- `time/30` → `time 30`; `words/50` → `words 50`.
- Quote ignores length and renders `quote`; Code ignores rune count and renders `code`.
- Non-empty unknown mode renders its raw identifier; empty renders `unknown`.
- Code and Strict history rows never display ★; normal eligible ties keep current behavior.

### Non-functional

- No storage schema, history migration, new dependency, or target-generation change.
- One canonical formatter serves current History and Result paths; future reuse is allowed.
- Keep touched Go files under 200 LOC; add focused test files instead of growing
  existing oversized screen test files.

## Architecture

`displayModeLabel(mode string, length int)` lives in `internal/ui` because it is
presentation logic and History owns forward-compatible raw string modes. Result
converts `config.Mode` to string. Storage exposes one eligibility predicate used
by `IsNewBest`, bucket aggregation, and History row rendering so the ★ contract
cannot drift between result celebration and history.

### Dependency Map

```text
storage.Record.Mode/Strict
  ├─> storage.EligibleForBest ─> IsNewBest
  └─> History.View ─> EligibleForBest + BestWPMPerBucket ─> renderHistoryRow

Result config.Mode ─┐
                    ├─> ui.displayModeLabel
History raw mode ───┘
```

## File Inventory

| Action | File | Change | Test impact |
|---|---|---|---|
| Create | `internal/ui/mode_label.go` | Shared explicit formatter | Unit contract |
| Create | `internal/ui/mode_label_test.go` | Formatter + History/Result regressions | New coverage |
| Modify | `internal/ui/history_table.go` | Use shared helper; delete duplicate | History label |
| Modify | `internal/ui/result_render_helpers.go` | Delete duplicate helper | Result label |
| Modify | `internal/ui/screen_result_view.go` | Call shared helper | Result meta |
| Modify | `internal/ui/screen_history_view.go` | Gate stars by eligibility | Code/Strict stars |
| Modify | `internal/storage/new_best.go` | Central eligibility helper; filter buckets | PB consistency |
| Create | `internal/storage/best_eligibility_test.go` | Code/Strict/normal eligibility | Storage contract |

## Function and Interface Checklist

- [ ] Add `displayModeLabel(string, int) string` with explicit known cases.
- [ ] Replace `modeLabel` caller and remove old definition.
- [ ] Replace `modeMetaLabel` caller and remove old definition.
- [ ] Add `storage.EligibleForBest(Record) bool`; reuse in `IsNewBest`.
- [ ] Make `BestWPMPerBucket` ignore ineligible records.
- [ ] Make `IsNewBest` ignore ineligible historical records as well as candidates.
- [ ] Require eligibility before History compares row WPM with bucket best.
- [ ] Verify `Record`, JSON tags, `buildRecord`, and best bucket keys remain unchanged.

## Tests Before

1. Add the formatter table and confirm Code/unknown rows fail against current code.
2. Add History Code and Result Code rendering tests; assert bounded metadata,
   not raw ANSI sequences.
3. Add History tests proving Code and Strict rows have no ★ while eligible
   Time/Words/Quote behavior remains.
4. Add a normal-candidate regression where a faster historical Strict/Code run
   must not suppress a valid eligible PB.
5. Add storage eligibility/bucket tests before changing implementation.

## Test Scenario Matrix

| Priority | Scenario | Expected |
|---|---|---|
| Critical | Result `ModeCode` | `· code · english`, never Quote |
| Critical | History Code record with `Length=142` | `code`, no rune count, no ★ |
| Critical | Strict Time record faster than normal | Strict row has no ★ |
| Critical | Faster historical Strict/Code then normal candidate | Ineligible history cannot suppress PB |
| High | Time 30 / Words 50 / Quote | Existing labels unchanged |
| High | Unknown `custom` / empty mode | `custom` / `unknown` |
| High | Eligible tied normal records | History marks co-record holders; Result celebrates only strict improvement |
| Medium | Time/Words length 0 | Deterministic `time 0` / `words 0` |

## Refactor

1. Implement the shared formatter.
2. Rewire both screen paths and remove duplicate helpers.
3. Centralize best eligibility and reuse it in result/history best paths.
4. Keep comments domain-based; do not mention plan phases or finding IDs.

## Tests After and Regression Gate

```sh
go test ./internal/ui ./internal/storage ./internal/app -count=1
go test ./... -race -count=1
make lint
make size-check
```

## Implementation Steps

1. Record baseline test results/workspace status and SHA-256 + metadata for the
   untracked journal without printing its contents.
2. Write failing tests from the matrix.
3. Implement formatter and eligibility changes.
4. Search for stale symbols: old formatter names must have zero matches.
5. Run focused and full gates; inspect rendered substrings manually.
6. Stage only explicit Phase 1 paths; never `git add -A`.

## Success Criteria

- [ ] All matrix cases pass.
- [ ] Exactly one canonical mode-label formatter serves both current display paths.
- [ ] Code/Strict ★ contract matches README and `IsNewBest` semantics.
- [ ] Storage JSON and public CLI output shapes are byte-compatible.
- [ ] Full gates green; journal remains untouched/untracked.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Unknown modes still lie as Quote | Corrupt display meaning | Raw fallback test |
| Ineligible row equals eligible best | False ★ despite bucket filter | Row-level eligibility guard |
| Large test files grow further | Violates repo convention | New focused test files |
| Accidental PB/schema redesign | Scope and compatibility risk | Read-only boundary checklist |

## Security Considerations

Unknown mode values originate from local JSON. Preserve current rendering model;
do not add sanitization or schema policy without a separate threat model. Accepted
risk: a hand-edited long/control-character mode ID can disturb terminal layout.

## Unresolved Questions

None.
