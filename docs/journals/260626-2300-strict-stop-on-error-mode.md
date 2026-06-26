---
title: Strict stop-on-error typing mode
date: 2026-06-26
type: journal
---

# Strict stop-on-error typing mode

## Context

Implemented plan [plans/260626-2213-strict-stop-on-error-mode/plan.md](file:///Users/vchun/Codes/My-projects/Typeburn/plans/260626-2213-strict-stop-on-error-mode/plan.md) to add an optional, letter-strict typing mode that blocks the cursor from advancing past errors while continuing to record fumbled keystrokes for accurate metrics.

## What Happened

- **Phase 1: Logic & Metrics**:
  - Implemented the letter-strict engine policy in `internal/typing/engine.go` via a new `NewStrict` constructor. Correct runes advance the cursor, while incorrect runes are logged but keep the cursor frozen at its current position.
  - Implemented the additive, mode-agnostic `KeystrokeAccuracy` metric (`100 * correctForward / totalForward`) in `internal/metrics/compute.go` to capture error rates honestly in strict mode where the final buffer has no errors.
  - Verification: `strict_engine_test.go` and `keystroke_accuracy_test.go`.

- **Phase 2: Settings & CLI Config**:
  - Added `StrictMode` to `config.Settings` (default `false`) with backward-compatible JSON deserialization.
  - Exposed `strict_mode` via CLI configuration: list, get, and set commands (`typeburn config set strict_mode on|off`).
  - Verification: `strict_mode_settings_test.go` and `strict_mode_config_test.go`.

- **Phase 3: Wiring, TUI, Bests Exclusion**:
  - Wired `StrictMode` through `NewSession` and `NewCodeSession` down to `typing.Engine` construction in `internal/runner/session.go`.
  - Added the "Strict mode" setting row to `internal/ui/settings_rows.go` and live-apply logic in `internal/ui/screen_settings.go`.
  - Updated the Result screen to show `KeystrokeAccuracy` instead of final-state `Accuracy` when the run is strict.
  - Updated `internal/storage/new_best.go` to exclude strict runs from personal bests (★) records.
  - Verification: Updated TUI tests in `screen_typing_test.go`, `screen_typing_actions_test.go`, `screen_settings_test.go`, and added `strict_new_best_test.go`.
  - Merged PR #54 into `main`.

- **Phase 4: Docs Sync & Verification**:
  - Updated `README.md`, `CLAUDE.md`, and all `docs/` files (architecture, summary, roadmap, overview) to fully document the strict mode feature and `KeystrokeAccuracy` metric.
  - Merged PR #55 into `main`.

## Verification

- Executed `make build`, `make test-race`, and `make lint` locally and confirmed that the full CI gate is green.
- Verified that strict runs are stored in history with `strict: true` and excluded from ★ records.
- Verified Settings TUI row correctly toggles and live-applies strict mode.

## Decisions

- **Accuracy Mapping:** Mapped keystroke accuracy directly to the persistent `Record.Accuracy` field for strict runs to avoid changing the history table column layout.
- **★ Exclusion:** Excluded strict runs from personal bests (★) because they are not comparable to standard run timings (errors block cursor).

## Unresolved Questions

None.
