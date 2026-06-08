---
title: Audit Hardening — Bug Fixes & Test Coverage
date: 2026-06-08
type: journal
---

# Audit Hardening — Bug Fixes & Test Coverage

## Context

Implemented plan [260608-1835-audit-hardening](file:///Users/vchun/Codes/My-projects/Typeburn/plans/260608-1835-audit-hardening/plan.md) targeting the 6 action items identified from the codebase review and git retro audit. Focus was on fixing critical bugs and increasing test coverage for packages with low coverage ([internal/app](file:///Users/vchun/Codes/My-projects/Typeburn/internal/app), [internal/storage](file:///Users/vchun/Codes/My-projects/Typeburn/internal/storage), [internal/ui](file:///Users/vchun/Codes/My-projects/Typeburn/internal/ui), [internal/typing](file:///Users/vchun/Codes/My-projects/Typeburn/internal/typing)).

## What Happened

- **Phase 1 (Bug Fixes)**:
  - Cleaned up the custom `itoa` helper in [new_best.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/storage/new_best.go). The old `for n > 0` loop returned an empty string for negative `length` (it did not loop infinitely, and negative length is not reachable in production — `Length` always comes from validated settings). Replaced the custom 15-line helper with standard Go [strconv.Itoa](https://pkg.go.dev/strconv#Itoa) for clarity and correct negative handling.
  - Fixed a `modeIdx` reset bug in [screen_home.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/ui/screen_home.go). Re-applying settings was triggering `NewHome()` which snapped the UI active tab back to the default mode. Added a `WithSettings()` method to only update theme/keymap, preserving `modeIdx` and `lenIdx` in the TUI screen model.
- **Phases 2-4 (Test Hardening)**:
  - Added 6 edge case tests to [new_best_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/storage/new_best_test.go).
  - Modified [history_store_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/storage/history_store_test.go) to set `NetWPM` in `makeRecord`.
  - Added test suites for the core app models: [model_view_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/app/model_view_test.go) (11 tests), [model_settings_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/app/model_settings_test.go) (2 tests), and [model_history_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/app/model_history_test.go) (6 tests).
  - Added tests for completion and actions: [completion_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/typing/completion_test.go) (4 tests) and [screen_typing_actions_test.go](file:///Users/vchun/Codes/My-projects/Typeburn/internal/ui/screen_typing_actions_test.go) (5 tests).
- **Validation**: All 16 packages passed tests with race detector (`go test ./... -race -count=1`), and `go vet` / `gofmt` run cleanly.

## Reflection

- **Coverage Metrics**:
  - `internal/app` coverage increased from 75.1% to **82.8%** (+7.7pp).
  - `internal/typing` coverage increased from 96.2% to **97.7%** (+1.5pp).
  - `internal/ui` coverage increased from 87.3% to **88.2%** (+0.9pp).
  - `internal/storage` coverage remained flat at 78.0% (down slightly from 78.9% due to removing dead `itoa` helper lines, though the new tests cover `new_best.go` at 100%). Further increases are gated by `atomic_write.go` error handling paths (which is at 42.1% due to OS write-error scenarios).
- Parallel subagent test writers worked effectively without file conflicts, completing the suite within minutes.

## Decisions

- Retain custom `atomic_write.go` error path testing as a future task since simulating disk full or OS-level permission failures in unit tests requires extensive mocking of the Go `os` filesystem layer which was out of scope for this sweep.
- Preserved existing UI model structures without major refactoring since coverage was successfully elevated via state assertions in TUI key event simulations.

## Next

- Propose committing the changes to `main` as all tests pass and bugs are resolved.
