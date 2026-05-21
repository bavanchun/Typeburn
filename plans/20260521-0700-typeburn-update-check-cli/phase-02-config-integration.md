---
phase: 2
title: "Config Integration"
status: pending
priority: P1
effort: "2h"
dependencies: [1]
---

# Phase 2: Config Integration (`update_check` key)

## Overview

Add a single new config key `update_check` (bool, **default `false`**) to
`config.Settings`, wire it through the existing `typeburn config set/get/list`
switch-based registry in `cmd_config.go`. Phase 4 will read this field.

## Requirements

**Functional**
- New field `Settings.UpdateCheck bool` with JSON tag `update_check`.
- `Defaults()` returns `UpdateCheck: false` (LOCKED — scout §3 suggested `true`;
  reversed per user decision in brainstorm).
- `typeburn config list` shows the new key in the table.
- `typeburn config get update_check` returns `false` (or current value).
- `typeburn config set update_check true|false` accepts canonical bool inputs;
  rejects others with a clear error.
- Backwards-compat: existing `settings.json` files on disk without the field
  load successfully — missing field defaults to `false`.

**Non-functional**
- No new go.mod entries.
- No `Normalize()` changes (bool has no invalid values).

## Architecture

Scout §2 confirms `cmd_config.go` uses explicit `switch` on key string —
not reflection. Adding a key is mechanical: struct field + Defaults() entry
+ row in `configRows()` + case in `configSet()` switch. `configGet()` is
loop-based over `configRows()`, so no separate change.

## Related Code Files

- **Modify:** `internal/config/settings.go` (lines 35-40 add field; lines 44-51 add default).
- **Modify:** `internal/cli/cmd_config.go` (lines 86-102 add row; lines 104-136 add set case).
- **Modify:** `internal/config/settings_test.go` — add cases for default + missing-field load.
- **Modify:** `internal/cli/cmd_config_test.go` — add cases for set/get of `update_check`.

## Implementation Steps

1. Add `UpdateCheck bool \`json:"update_check"\`` to `config.Settings` struct.
2. Add `UpdateCheck: false` to `Defaults()` return value. **Comment in code
   must NOT reference plan/finding labels** (per user CLAUDE.md rule §5):
   write the *reason* if any (e.g., `// opt-in network: off by default to
   preserve offline-first posture`), not a phase number.
3. Add `{"update_check", strconv.FormatBool(s.UpdateCheck)}` to `configRows()`.
4. Add `case "update_check":` to `configSet()` switch with a `parseBool(value)`
   call (existing helper at `cmd_config.go:138-147`).
   <!-- SUPERSEDED by red-team-finding-3 — existing parseBool accepts ONLY true/false/1/0; pick resolution (a) extend to accept on/off/yes/no, OR (b) drop on/off from docs and success criteria. See Red Team Review Updates. -->
   If resolution (a): also update `parseBool` to accept `on/off`/`yes/no`.
5. **Tests:**
   - `settings_test.go`: assert `Defaults().UpdateCheck == false`; load a JSON
     file missing the field, assert result `.UpdateCheck == false`.
   - `cmd_config_test.go`: `set update_check true` → `get update_check` returns
     `true`; invalid value (e.g., `"maybe"`) returns a parse error.
6. Run `make test-race && make lint`. Confirm `go.mod` unchanged.

## Todo List

- [ ] Add `UpdateCheck bool` field to `Settings` struct
- [ ] Add `UpdateCheck: false` to `Defaults()`
- [ ] Add row to `configRows()`
- [ ] Add `case "update_check":` to `configSet()` switch
- [ ] settings_test.go cases
- [ ] cmd_config_test.go cases
- [ ] Backwards-compat load test (missing field → false)
- [ ] All gates green

## Success Criteria

- [ ] `typeburn config list` includes `update_check  false` row.
- [ ] `typeburn config set update_check on` works (`strconv.ParseBool` accepts `on/off/true/false/1/0`).
- [ ] `typeburn config set update_check maybe` returns error.
- [ ] Loading a pre-v2.1 settings file (no `update_check` key) yields `false`.
- [ ] `go test ./internal/config/... ./internal/cli/...` passes.

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Default-true drift (scout/auditor pressure) | Plan §"Locked decisions" explicit; code comment justifies; CI does not enforce default — code review must catch flips |
| `parseBool` helper name collides with existing | Search before adding; reuse if present |
| JSON tag mismatch (camelCase vs snake_case) | Existing keys use snake_case (`default_mode`); follow that |

## Security Considerations

None — bool config field, no PII, no network.

## Next Steps

- Phase 3 consumes this field indirectly via `update.Check()`; explicit flag
  bypasses the gate (always runs when user passes `--check-update`).
- Phase 4 gates the opportunistic check on `cfg.UpdateCheck`.

## Red Team Review Updates (2026-05-21)

<!-- red-team-finding-3 --> **H1 — `parseBool` does NOT accept `on/off`.** Verified at
`internal/cli/cmd_config.go:138-147`: existing `parseBool` accepts ONLY
`true/false/1/0`. Plan's Success Criteria #2 ("`strconv.ParseBool` accepts
`on/off/true/false/1/0`") and Phase 4's manual-smoke `typeburn config set update_check on`
are both factually wrong — these commands would fail today.

**Resolution — locked (a) by user 2026-05-21:**

**(a)** Extend `parseBool` in `cmd_config.go` to accept `on/off` (also `yes/no` while we're at it for unix conventions):
```go
func parseBool(value string) (bool, bool) {
    switch strings.ToLower(strings.TrimSpace(value)) {
    case "true", "1", "on", "yes":  return true, true
    case "false", "0", "off", "no": return false, true
    default:                          return false, false
    }
}
```
Add tests for all new accepted forms AND backwards-compat (previous values still work).
Update existing `blink_cursor` set behavior — same parser, so it inherits this.

**(b)** ~~Keep `parseBool` as-is; rewrite plan + docs + Phase 4 smoke to use only
`true/false`. Update README/cli-reference accordingly. Reject `on/off` in docs.~~
**Not chosen.**

**Implementation Step update** — Step 4 now reads: "Add `case \"update_check\":`
to `configSet()` switch, calling the (possibly extended per resolution (a)) `parseBool`."

**Success Criteria update** — replace criterion 2 with:
- [ ] `typeburn config set update_check true` works.
- [ ] `typeburn config set update_check on` works (only if resolution (a) chosen).
- [ ] `typeburn config set update_check maybe` returns a clear error listing valid forms.
