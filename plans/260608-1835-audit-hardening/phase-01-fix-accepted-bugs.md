---
phase: 1
title: "Fix Accepted Bugs"
status: pending
priority: P1
effort: "30m"
dependencies: []
---

# Phase 1: Fix Accepted Bugs

## Overview

Fix 2 accepted findings from the adversarial code review. Both are
correctness bugs — one latent (itoa infinite loop), one UX-visible
(modeIdx reset on settings change).

## Related Code Files

- Modify: `internal/storage/new_best.go` — replace `itoa` with `strconv.Itoa`
- Modify: `internal/app/model_settings.go` — use `WithSettings` instead of `NewHome`
- Modify: `internal/ui/screen_home.go` — add `WithSettings` method

## Implementation Steps

### Bug 1: `itoa` infinite loop on negative input

**File:** `internal/storage/new_best.go`

1. Add `import "strconv"` to the package
2. In `BestBucketKey`, replace:
   ```go
   // Inline int-to-string to avoid importing fmt/strconv in this tiny file.
   key += itoa(length)
   ```
   with:
   ```go
   key += strconv.Itoa(length)
   ```
3. Delete the entire `itoa` function (lines 20–34)

**Rationale:** Hand-rolled `itoa` only handles `n >= 0`. A corrupt JSON
record with `"length": -1` causes `for n > 0` to never terminate because
dividing a negative int by 10 stays negative. `strconv.Itoa` handles all
int values safely.

### Bug 2: `applySettings` resets `modeIdx`

**File 1:** `internal/ui/screen_home.go`

Add a new method after `WithCodeText`:
```go
// WithSettings returns a copy with theme and keymap updated from the new
// settings, preserving modeIdx and lenIdx so the user's current selection
// is not reset when a setting changes (same preservation pattern as
// WithCodeText).
func (m HomeModel) WithSettings(s config.Settings, th theme.Theme, km config.Keymap) HomeModel {
    m.th = th
    m.km = km
    return m
}
```

**File 2:** `internal/app/model_settings.go`

In `applySettings`, replace line 40:
```go
m.home = ui.NewHome(s, m.theme, m.keys, m.codeText, m.codeHint).SetSize(m.w, m.h)
```
with:
```go
m.home = m.home.WithSettings(s, m.theme, m.keys).SetSize(m.w, m.h)
```

**Rationale:** `NewHome` rebuilds `modeIdx` from `s.DefaultMode`, snapping
the user's tab selection back to default whenever any setting is toggled.
The new `WithSettings` only updates theme/keymap, preserving modeIdx/lenIdx.
Same pattern as `WithCodeText` (which already documents this exact risk).

## Success Criteria

- [ ] `go test ./internal/storage/... -count=1` passes
- [ ] `go test ./internal/app/... -count=1` passes
- [ ] `go test ./... -race -count=1` passes
- [ ] `go vet ./...` clean
- [ ] `gofmt -l .` returns no output
- [ ] `BestBucketKey("time", -1)` returns `"time/-1"` (not infinite loop)
- [ ] Changing a setting while on Code mode tab does NOT reset to Time mode

## Risk Assessment

- **Low risk.** Both changes are minimal (net ~5 LOC each). `strconv.Itoa`
  is stdlib, zero new dependencies. `WithSettings` follows the exact same
  pattern as `WithCodeText` which has been stable since v1.1.0.
