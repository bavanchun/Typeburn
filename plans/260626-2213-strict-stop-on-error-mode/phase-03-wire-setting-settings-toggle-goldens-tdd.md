---
phase: 3
title: "Wire setting + Settings toggle + goldens (TDD)"
status: completed
priority: P1
dependencies: [2]
---

# Phase 3: Wire setting + Settings toggle + goldens (TDD)

## Overview

Connect the pieces: thread `StrictMode` into engine construction, add a "Strict
mode" row to the Settings screen (live-applied + persisted), and surface
keystroke accuracy for strict runs on the Result screen. Regenerate teatest
goldens.

## Requirements
- Functional:
  - The engine is constructed with `strict = settings.StrictMode` at the single
    call site (`internal/runner/session.go:46`). Trace session callers and pass
    the flag down from loaded settings.
  - Settings screen shows a fixed-index "Strict mode" row (`off`/`on`) after
    "Blink cursor"; selecting toggles `s.StrictMode` via `applyRow`; live-applies
    in-session (same path that already live-applies theme/blink) and persists.
  - Result screen shows keystroke accuracy when the completed run was strict
    (otherwise final-state accuracy as today).
  - `storage.Record` gains `Strict bool json:"strict"` (back-compat: legacy → false).
  - Strict runs are saved to History but **excluded from ★**: extend
    `internal/storage/new_best.go` `IsNewBest` (line ~62) from `if r.Mode ==
    "code"` to also return `false` when `r.Strict`.
  - History table shows keystroke accuracy for strict rows (via `Record.Strict`).
- Non-functional: keymap/screens stay the only UI seam; files < 200 LOC (split
  Settings view/rows if needed).

## Architecture
- **Engine wiring:** `runner.Session` (or its constructor) must receive
  `strict`. Inspect how `session.go` obtains mode/length today and add `strict`
  alongside, sourced from `config.Settings`. Update `New`→`NewStrict` call.
- **Settings row:** in `internal/ui/settings_rows.go` add `rowStrictMode = 4`
  and a row `{label: "Strict mode", values: []string{"off","on"}, idx: …,
  help: "Block wrong keys: cursor will not advance past an error."}`. In
  `internal/ui/screen_settings.go` `applyRow`, add
  `case rowStrictMode: m.s.StrictMode = (val == "on")`.
- **Result display:** where the Result screen reads `Accuracy`, branch on whether
  the run was strict to show `KeystrokeAccuracy`. Source the strict flag from the
  in-session result for the live Result screen, and from `Record.Strict` for the
  History table (the field is added this phase — resolved decision #3).

## Related Code Files
- Modify: `internal/runner/session.go` (+ callers if signature changes)
- Modify: `internal/ui/settings_rows.go` (new fixed-index row)
- Modify: `internal/ui/screen_settings.go` (applyRow case)
- Modify: `internal/ui/screen_settings_view.go` if row rendering needs it
- Modify: `internal/ui/screen_result*.go` (accuracy display branch)
- Modify: `internal/storage/history_record.go` (+`Strict bool json:"strict"`)
- Modify: `internal/storage/new_best.go` (extend ★ exclusion with `|| r.Strict`)
- Modify: `internal/app` result-persistence path (set `Record.Strict` from the
  run; `internal/app` is the only place that builds + persists Records)
- Update goldens: `internal/ui/testdata/*` (teatest), `internal/app` smoke/golden
- Tests: `internal/ui/screen_settings_test.go`, result-screen test, an
  app-level test that strict setting reaches the engine,
  `internal/storage/new_best_test.go` (strict record never new-best)

## Implementation Steps (TDD)
1. **Red:** test that constructing a session/engine with `StrictMode=true`
   produces a strict engine (blocks wrong key end-to-end).
2. **Red:** Settings screen test — toggling the new row flips `StrictMode` and
   persists; row index constants unchanged for existing rows.
3. **Red:** Result screen test — a strict result renders keystroke accuracy.
4. **Green:** implement wiring + row + result branch.
5. Regenerate teatest goldens deliberately; **review the diff** (a new Settings
   row is expected; nothing else should shift).
6. `make test-race && make lint`.
7. **Commit** (e.g. `feat(ui): wire strict mode + Settings toggle + result
   accuracy`).
8. **On completion run `/vchun-git prc`** (branch `feat/strict-mode-p3-wire`).
   Then `ck plan check phase-03-wire-setting-settings-toggle-goldens-tdd`.

## Success Criteria
- [x] Toggling strict in Settings changes live typing behavior + persists.
- [x] Strict run shows error-aware (keystroke) accuracy on Result.
- [x] Existing Settings row indices/goldens change only by the added row.
- [x] `make test-race` + `make lint` green.
- [x] Phase committed; PR squash-merged via `/vchun-git prc`; CI green.

## Risk Assessment
- **Risk:** golden churn obscures a real regression. **Mitigation:** review
  golden diffs line-by-line; only the Settings screen + strict Result should change.
- **Risk:** `session.go` signature change ripples to multiple callers.
  **Mitigation:** scout callers first (`grep -rn "runner.NewSession\|Session{"`);
  thread the flag without altering unrelated params.
- **Risk:** the ★-exclusion + `Record.Strict` schema add are in-scope (resolved
  decisions) — keep the schema change backward-compatible (legacy JSON → false)
  and assert it with a `new_best_test.go` case. Do not add a separate best
  bucket (explicitly rejected — KISS).
