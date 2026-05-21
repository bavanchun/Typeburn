---
phase: 3
title: "Version --check-update Flag"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 3: `typeburn version --check-update` Flag

## Overview

Add an explicit, synchronous update-check trigger as a flag on the existing
`version` subcommand. Always runs (config-independent — user explicitly asked).
Supports `--json`. Honors the same 1.5s timeout + silent-degrade rules but
prints an honest "could not check" line on errors (or `{"error":"..."}` for JSON)
because the user actively requested information.

## Requirements

**Functional**
- New flag: `--check-update` (BoolVar, long-only, no short form).
- When set, after the existing version output, call
  `update.Check(ctx, version.Resolve().Version, force=true)`.
- Human output:
  - Up-to-date: `you are on the latest version (v2.1.0).`
  - Upgrade available: full 3-line block (release URL + 3 verbatim commands).
  - Dev/pseudo-version skip: `version check skipped: build has no release version.`
  - Error: `could not check for updates: <reason>` to stderr; exit code 0
    (do not fail the user's command).
- `--json` output:
  ```json
  {
    "current": "v2.0.0",
    "latest": "v2.1.0",
    "upgrade_available": true,
    "release_url": "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
    "checked_at": "2026-05-21T07:00:00Z"
  }
  ```
  Error JSON: `{"error":"..."}` with non-zero exit code (scripting needs to detect).
- Combined `--json` + `--check-update`: emit a single JSON object that includes
  the existing version fields PLUS an `update_check` object — proposed wrapper:
  ```json
  {
    "version": {"version":"v2.0.0","commit":"abc","date":"..."},
    "update_check": {...}
  }
  ```
  Decide between this and "two JSON objects on consecutive lines" — propose
  **single object** for jq-friendliness.
- Explicit flag bypasses the cache (`force=true`).

**Non-functional**
- New code path < 50 LOC delta in `cmd_version.go`.
- Stdlib only.

## Architecture

Scout §1: existing `cmd_version.go:24` registers `--json` flag adjacent;
add `--check-update` in the same block. The `runVersion` function gains a
parameter and a conditional tail.

For the JSON wrapper decision: changing the existing `--json` shape would
break v2.0.0 scripts. **Decision:** when `--check-update` is set with `--json`,
emit the wrapper. Without `--check-update`, the existing `--json` shape stays
identical (backwards-compatible).

## Related Code Files

- **Modify:** `internal/cli/cmd_version.go` (lines 24, 29-31, 33-51).
- **Modify:** `internal/cli/cmd_version_test.go` — add flag tests.
- **Create:** Possibly `internal/cli/cmd_version_check.go` if `cmd_version.go`
  exceeds 200 LOC after the addition. **First check current LOC; modularize
  only if needed.**

## Implementation Steps

1. Add `var checkUpdate bool` and `cmd.Flags().BoolVar(&checkUpdate, "check-update", false, "check GitHub for a newer release")`.
2. Refactor `runVersion(cmd, asJSON, checkUpdate)` signature.
3. If `checkUpdate`:
   - `ctx := cmd.Context()` (cobra provides; respects parent timeout if any).
   - `currentVer := version.Resolve().Version`
   - `result, err := update.Check(ctx, currentVer, true)`
   - Branch on `result == nil && err == nil` (dev-skip), `err != nil` (error path),
     `result.UpgradeAvailable` (upgrade path), else (up-to-date path).
4. Render:
   - Human path: `fmt.Fprintln(cmd.OutOrStdout(), ...)` for the 4 scenarios above.
   - Upgrade hint block (locked verbatim):
     ```
     typeburn <latest> is available (you have <current>).
     Release notes: <url>
     Upgrade with one of:
       brew upgrade typeburn
       curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
       go install github.com/bavanchun/Typeburn@latest
     ```
   - JSON wrapper path: build the `{version, update_check}` struct, pass to
     `output.RenderJSON`.
5. **Tests** (`cmd_version_test.go`):
   - Inject a stub for `update.Check` via a package-level function var (test seam).
     Pattern: `var checkFn = update.Check` at file top; tests overwrite.
   - Cases: stable-upgrade, up-to-date, dev-skip, error, JSON wrapper shape.

## Todo List

- [ ] Register `--check-update` flag
- [ ] Refactor `runVersion` signature + body
- [ ] Add 4-branch rendering (skip/error/upgrade/up-to-date)
- [ ] JSON wrapper struct
- [ ] Test seam (`var checkFn = update.Check`)
- [ ] Tests: stable-upgrade, up-to-date, dev-skip, error, JSON
- [ ] LOC check (split if cmd_version.go ≥ 200)
- [ ] Manual smoke: `./bin/typeburn version --check-update` against real API

## Success Criteria

- [ ] `typeburn version --check-update` against real `api.github.com` returns
  ≤1.6s on cold, ≤100ms on warm cache (Phase 1 cache shared with Phase 4).
- [ ] `typeburn version --check-update --json` emits valid JSON parseable by `jq`.
- [ ] Existing `typeburn version --json` output is **unchanged** when
  `--check-update` is NOT passed (backwards-compat).
- [ ] Network failure prints to stderr, exits 0 (human) / non-zero (JSON
  with `--json`).

## Risk Assessment

| Risk | Mitigation |
|---|---|
| `cmd_version.go` exceeds 200 LOC | Split into `cmd_version_check.go` |
| Test seam breaks production wiring | Seam is a `var` assignment at file top — compile-checked |
| JSON wrapper breaks v2.0.0 scripts | Only emitted when `--check-update` is set; default JSON shape unchanged |
| Real-API call in test suite (flake/rate-limit) | Tests use the seam, NOT the real API |
| User confusion between explicit flag and config | Help text + docs (Phase 5) make it explicit |

## Security Considerations

- Stderr output may include URLs/error strings; sanitize: no user input
  flows into error messages, so injection risk is nil.
- Exit code policy: 0 for success/dev-skip/network-fail in human mode;
  non-zero for `--json` errors to support `set -e` scripts.

## Next Steps

- Phase 4 reuses `update.Check` (with `force=false`) for the opportunistic
  path; cache built here on warm-up benefits TUI launch.

## Red Team Review Updates (2026-05-21)

<!-- red-team-finding-15 --> **M12 — `Resolve()` signature pin.** Verified at `internal/version/version.go:28-32, 40`:
`Resolve()` returns the `Info` struct `{Version, Commit, Date string}`. The Phase 3
pseudo-code `currentVer := version.Resolve().Version` is correct (no change needed
here). The contradiction was in Phase 1 prose only — corrected in that phase's
red-team updates. Add a one-line comment in implementation to anchor for future readers:
```go
// version.Resolve() returns Info{Version, Commit, Date}; .Version is the resolved tag.
```
