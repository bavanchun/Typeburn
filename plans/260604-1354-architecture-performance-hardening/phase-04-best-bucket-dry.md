---
phase: 4
title: Best Bucket DRY
status: completed
priority: P2
effort: 1h
dependencies:
  - 1
---

# Phase 4: Best Bucket DRY

## Context Links

- Storage source: `internal/storage/new_best.go`
- UI duplicate: `internal/ui/history_table.go`
- History view call site: `internal/ui/screen_history_view.go`
- Existing tests: `internal/storage/history_store_test.go`,
  `internal/ui/screen_history_test.go`

## Overview

Remove duplicated best-bucket/effective-WPM logic between storage and UI. Keep
the accepted legacy `NetWPM==0` fallback exactly as-is.

## Requirements

- Functional: history stars and `IsNewBest` behavior unchanged.
- Non-functional: one source of truth for bucket key and effective WPM logic.
- Compatibility: no storage schema change.

## Architecture

Preferred approach: export storage helpers with domain names:

```go
func EffectiveWPM(r Record) float64
func BestBucketKey(mode string, length int) string
func BestWPMPerBucket(records []Record) map[string]float64
```

UI history rendering consumes these helpers. `IsNewBest` also uses them. Keep
storage as owner because best-bucket semantics belong to persisted history, not
rendering.

## File Inventory

| File | Action | Rough size | Test impact |
|---|---:|---:|---|
| `internal/storage/new_best.go` | Modify | 82 LOC | Export helpers, keep old behavior |
| `internal/storage/history_store_test.go` | Modify | 370 LOC now | Consider new `new_best_test.go` split |
| `internal/storage/new_best_test.go` | Create | <140 LOC | Helper-level tests if splitting |
| `internal/ui/history_table.go` | Modify | 149 LOC | Remove duplicate helpers |
| `internal/ui/screen_history_view.go` | Modify | small | Use storage helpers |
| `internal/ui/screen_history_test.go` | Existing | 234 LOC | Existing star tests should pass |

## Test Scenario Matrix

| Scenario | Criticality | Protection |
|---|---|---|
| First bucket result is best | High | Existing storage tests |
| Higher NetWPM wins | Critical | Existing storage tests |
| Equal WPM not new best | High | Existing storage tests |
| Legacy `NetWPM==0` falls back to `WPM` | Critical | Existing storage test |
| Code mode never best | High | Existing storage test |
| History star marks same bucket best | High | Existing UI test |

## Dependency Map

```text
storage.BestWPMPerBucket
├── storage.IsNewBest
└── ui.screen_history_view / history_table
```

## Implementation Steps

1. Write or move tests for exported helper names before changing UI:
   - `EffectiveWPM`
   - `BestBucketKey`
   - `BestWPMPerBucket`
2. Rename unexported helpers or add exported wrappers in `storage`.
3. Update `IsNewBest` to use exported helpers.
4. Remove duplicate `effWPM`, `histBucketKey`, `bestWPMPerBucket` from UI.
5. Update UI history rendering to call storage helpers.
6. Run:
   ```sh
   go test ./internal/storage ./internal/ui -count=1
   ```

## Success Criteria

- [ ] Best-bucket logic has one source of truth in `internal/storage`.
- [ ] Legacy fallback behavior unchanged and tested.
- [ ] History stars render unchanged.
- [ ] No storage schema migration.
- [ ] Targeted tests pass.

## Risk Assessment

- Risk: exported helper names become public-ish API. Mitigation: package is
  internal; names should be clear and stable.
- Risk: UI imports more storage semantics. Mitigation: UI already consumes
  `storage.Record`; this reduces duplication.
- Risk: accidental schema change. Mitigation: do not touch `Record` fields or
  JSON tags.
