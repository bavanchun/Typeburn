---
phase: 2
title: "Settings + CLI config surface (TDD)"
status: completed
priority: P1
dependencies: [1]
---

# Phase 2: Settings + CLI config surface (TDD)

## Overview

Add the persisted `strict_mode` boolean to `config.Settings` and expose it via
`typeburn config` (list/get/set), mirroring the existing `blink_cursor` /
`update_check` booleans. No engine wiring or UI screen changes yet.

## Requirements
- Functional:
  - `config.Settings` gains `StrictMode bool json:"strict_mode"`.
  - `Defaults()` sets it `false`.
  - `Normalize()` needs no clause (bool is always valid) — but add a regression
    test asserting an old JSON without the key loads as `false`.
  - `typeburn config list` includes a `strict_mode` row.
  - `config get strict_mode` prints `true`/`false`.
  - `config set strict_mode <v>` accepts true/false/1/0/on/off/yes/no
    (reuse `parseBool`); invalid value → usage error.
- Non-functional: backward-compatible JSON; files < 200 LOC.

## Architecture
Follow the `blink_cursor` precedent exactly:
- `internal/config/settings.go`: add field + default.
- `internal/cli/cmd_config.go`:
  - `configRows`: append `{"strict_mode", strconv.FormatBool(s.StrictMode)}`.
  - `configSet`: add `case "strict_mode":` using `parseBool` (same error text
    pattern as `blink_cursor`).

## Related Code Files
- Modify: `internal/config/settings.go`
- Modify: `internal/config/settings_test.go` (default + back-compat load test)
- Modify: `internal/cli/cmd_config.go`
- Modify: `internal/cli/cmd_config_test.go` (list/get/set + invalid value)

## Implementation Steps (TDD)
1. **Red:** settings test — `Defaults().StrictMode == false`; unmarshal a JSON
   blob lacking `strict_mode` → field is `false` (zero value), `Normalize()`
   leaves it untouched.
2. **Red:** cmd_config tests — `config list` output contains `strict_mode`;
   `get` returns the value; `set strict_mode on` then `get` → `true`;
   `set strict_mode bogus` → usage error.
3. **Green:** add the field + default; extend `configRows`/`configSet`.
4. `go test ./internal/config/ ./internal/cli/ -race -count=1`.
5. **Commit** (e.g. `feat(config): add strict_mode setting + config CLI key`).
6. **On completion run `/vchun-git prc`** (branch `feat/strict-mode-p2-config`).
   Then `ck plan check phase-02-settings-cli-config-surface-tdd`.

## Success Criteria
- [x] `StrictMode` field + default `false`; old JSON loads as `false`.
- [x] `config list/get/set strict_mode` work; invalid value rejected.
- [x] Config tests green; lint clean.
- [x] Phase committed; PR squash-merged via `/vchun-git prc`; CI green.

## Risk Assessment
- **Risk:** forgetting to keep `configRows` and `configSet` key sets in sync.
  **Mitigation:** test asserts `get` works for every key shown by `list`.
- **Risk:** none for `Normalize` (bool). Documented so the implementer does not
  add a needless clause.
