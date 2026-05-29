---
phase: 1
title: Fix in-app update hint
status: completed
priority: P1
effort: 30m
dependencies: []
---

# Phase 1: Fix in-app update hint

## Overview

The result-screen "update available" footer nudge points at the check-only
`typeburn version --check-update`, which never upgrades. Repoint it at
`typeburn update`. This is the highest-value fix — the stale string defeated
feature discovery.

## Requirements

- Functional: when an opportunistic check finds a newer release, the result
  screen's muted footer line tells the user to run `typeburn update`.
- Non-functional: NO_COLOR + mono layouts identical (attributes only); semver
  injection guard and width-capping behavior preserved.

## Architecture

- `internal/ui/screen_result_view.go:75` — `renderUpdateHint()` builds the hint
  string. Only the literal command text changes; the surrounding `Latest` semver
  guard (suppress non-semver) and `theme.RoleTextMuted` styling stay as-is.
- New string is shorter than the old, so the existing width cap cannot newly
  truncate — no layout regression.

## Related Code Files

- Modify: `internal/ui/screen_result_view.go` (line ~75, hint string)
- Modify: `internal/ui/screen_result_test.go` (assertion at line ~351)

## Implementation Steps (TDD)

1. **Red:** in `screen_result_test.go`, change the assertion
   `strings.Contains(view, "typeburn version --check-update")` to expect
   `"typeburn update"`. Run the test → fails against current code.
2. **Green:** in `screen_result_view.go`, change the hint format string from
   `↑ %s available — run "typeburn version --check-update"` to
   `↑ %s available — run "typeburn update"`.
3. Run `go test ./internal/ui/ -run Result -count=1` → green.
4. Confirm `TestRenderUpdateHint_InjectionGuard` still passes (non-semver
   `Latest` still suppressed — unaffected by the string change).

## Success Criteria

- [ ] Result-screen hint reads `run "typeburn update"`.
- [ ] `screen_result_test.go` assertion updated and green.
- [ ] Injection-guard test still passes.
- [ ] `go test ./internal/ui/ -race -count=1` green; `gofmt -l` clean.

## Risk Assessment

- Low. Single string + single assertion. Risk = a *second* test or golden also
  pins the old string → grep `check-update` under `internal/ui/` before editing
  (current scan shows only the one assertion; re-verify).
