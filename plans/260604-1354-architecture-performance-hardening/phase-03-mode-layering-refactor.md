---
phase: 3
title: Mode Layering Refactor
status: completed
priority: P1
effort: 1.5h
dependencies:
  - 1
---

# Phase 3: Mode Layering Refactor

## Context Links

- Current mode definitions: `internal/config/settings.go`
- Bubble Tea keymap: `internal/config/keymap.go`
- Pure packages importing config: `internal/typing`, `internal/metrics`,
  `internal/words`, `internal/storage`, `internal/runner`
- Architecture doc drift: `docs/system-architecture.md`

## Overview

Move test mode definitions out of `internal/config` so typing/metrics/words and
runner do not depend on the UI keymap package for `Mode`. This is architecture
cleanup, not a behavior change. It deliberately keeps `config.Keymap` in
`internal/config` per current repo rules.

## Requirements

- Functional: all existing mode strings and JSON values stay identical.
- Non-functional: packages that only need `Mode` import `internal/mode`, not
  `internal/config`.
- Compatibility: persisted settings/history JSON remains backward-compatible.

## Architecture

Create a small domain package:

```text
internal/mode
├── mode.go       # Mode enum: time, words, quote, code
└── lengths.go    # LengthsFor
```

Then `config.Settings.DefaultMode` becomes `mode.Mode`. Existing callers import
`internal/mode` directly when they only need mode constants/options. `config`
continues to expose settings/keymap/XDG; keymap stays the Bubble Tea input
boundary unless a later user-approved plan changes that.

## File Inventory

| File | Action | Rough size | Test impact |
|---|---:|---:|---|
| `internal/mode/mode.go` | Create | <80 LOC | New mode tests |
| `internal/mode/mode_test.go` | Create | <120 LOC | LengthsFor + known modes |
| `internal/config/settings.go` | Modify | 87 LOC | Settings uses `mode.Mode` |
| `internal/config/settings_test.go` | Modify | 248 LOC now | Consider split if touched heavily |
| `internal/config/mode_seam_sync_test.go` | Move/modify | 48 LOC | Move to `mode` or adapt imports |
| `internal/typing/*.go` | Modify imports | small | Existing tests |
| `internal/metrics/*.go` | Modify imports | small | Existing tests |
| `internal/words/*.go` | Modify imports | small | Existing tests |
| `internal/runner/session.go` | Modify imports | 54 LOC | Existing tests |
| `internal/storage/*.go` | Usually unchanged | small | Existing tests |
| `internal/ui`, `internal/app`, `internal/cli` | Modify imports | broad | Existing tests |

## Test Scenario Matrix

| Scenario | Criticality | Protection |
|---|---|---|
| Mode string values unchanged | Critical | `mode` tests |
| Settings JSON still round-trips | Critical | `config` + `storage` tests |
| Unknown mode normalizes to time | High | Existing settings tests |
| Code mode has no length selector | High | Existing mode seam tests |
| Mode-only packages stop importing config | Critical | `go list`/grep check |
| CLI flag parsing accepts all modes | High | Existing CLI tests |

## Dependency Map

```text
mode (new pure domain)
├── config.Settings
├── typing.Engine
├── metrics.Compute
├── words.ForMode
├── runner.Session
├── ui/app/cli call sites
└── storage only if mode helpers are genuinely needed

config.Keymap -> Bubble Tea remains allowed
```

## Implementation Steps

1. Add `internal/mode` with `Mode` constants and `LengthsFor`.
2. Move/adapt mode seam tests to `internal/mode`.
3. Update `config.Settings` to use `mode.Mode`; keep JSON tags identical.
4. Replace `config.Mode*` call sites with `mode.Mode*` where the caller does not
   need settings/keymap.
5. Keep `config.DefaultKeymap()` and `config.Keymap` unchanged unless a trivial
   import cleanup is needed.
6. Run:
   ```sh
   go test ./internal/mode ./internal/config ./internal/typing ./internal/metrics ./internal/words ./internal/runner -count=1
   ```
7. Verify mode-only packages no longer import config:
   ```sh
   go list -f '{{.ImportPath}} {{join .Imports " "}}' ./internal/typing ./internal/metrics ./internal/words ./internal/runner | rg 'internal/config'
   ```
   Expected: no matches after the refactor.

## Success Criteria

- [ ] New `internal/mode` package owns mode enum and length options.
- [ ] Mode-only packages use `internal/mode`, not `internal/config`, when they
      only need mode data.
- [ ] Persisted settings/history compatibility is unchanged.
- [ ] Existing CLI/UI behavior remains unchanged.
- [ ] Targeted tests and import-edge check pass.

## Risk Assessment

- Risk: import cycles. Mitigation: `mode` imports nothing project-local.
- Risk: widespread call-site churn. Mitigation: mechanical import change, no
  behavior changes.
- Risk: docs/tests still refer to `config.Mode` as source of truth. Mitigation:
  Phase 5 docs sweep and rg for stale references.
- Risk: reviewer asks to move `Keymap` too. Mitigation: reject in this plan
  unless user explicitly approves reversing the current `CLAUDE.md` boundary.
