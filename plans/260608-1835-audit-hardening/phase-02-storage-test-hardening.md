---
phase: 2
title: "Storage Test Hardening"
status: pending
priority: P2
effort: "1h"
dependencies: [1]
---

# Phase 2: Storage Test Hardening

## Overview

Improve `internal/storage` test coverage from 78.9% → 85%+ by filling gaps
identified in the code review: `itoa` negative input (now `strconv.Itoa`),
`atomicWrite` error paths, `makeRecord` NetWPM, and `BestBucketKey` edge cases.

## Related Code Files

- Modify: `internal/storage/new_best_test.go` — add negative length, zero length tests
- Modify: `internal/storage/history_store_test.go` — fix `makeRecord` helper, add concurrent test

## Implementation Steps

### 1. Fix `makeRecord` helper to set `NetWPM`

In `history_store_test.go`, update `makeRecord` helper:
```go
// Before:
WPM: wpm,
// After:
WPM:    wpm,
NetWPM: float64(wpm) + 0.42,
```

This exercises the primary `EffectiveWPM` code path (NetWPM > 0) instead of
always falling back to the legacy `float64(WPM)` path.

### 2. Add `BestBucketKey` edge case tests

In `new_best_test.go`, add table-driven tests:

```go
func TestBestBucketKey_EdgeCases(t *testing.T) {
    cases := []struct {
        name   string
        mode   string
        length int
        want   string
    }{
        {"time/0",     "time",  0,  "time/0"},
        {"time/neg",   "time",  -1, "time/-1"},
        {"words/0",    "words", 0,  "words/0"},
        {"quote",      "quote", 0,  "quote"},
        {"code",       "code",  42, "code"},
        {"empty mode", "",      10, ""},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := BestBucketKey(tc.mode, tc.length)
            if got != tc.want {
                t.Errorf("BestBucketKey(%q, %d) = %q, want %q",
                    tc.mode, tc.length, got, tc.want)
            }
        })
    }
}
```

### 3. Add concurrency documentation test

In `history_store_test.go`, add a test that documents the TOCTOU limitation:

```go
// TestAppendHistory_DocumentConcurrencyContract verifies AppendHistory works
// correctly in single-goroutine use. This test documents that AppendHistory
// is NOT concurrency-safe (see code review finding storage/M1).
func TestAppendHistory_DocumentConcurrencyContract(t *testing.T) {
    // ... single-goroutine append + verify
}
```

## Success Criteria

- [ ] `go test ./internal/storage/... -count=1 -cover` shows ≥85%
- [ ] All existing tests still pass
- [ ] `BestBucketKey("time", -1)` covered (was infinite loop before Phase 1 fix)
- [ ] `makeRecord` exercises NetWPM > 0 path

## Risk Assessment

- **Minimal.** Test-only changes. No production code modified.
