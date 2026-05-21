---
phase: 4
title: "history and config subcommands plus output renderers"
status: completed
priority: P2
effort: "5h"
dependencies: [2]
---

# Phase 4: `history` + `config` subcommands + output renderers

## Overview

Scriptable reads/writes of persisted state without launching TUI:
- `typeburn history` — print last N records (table) or `--json`
- `typeburn config get|set|list` — manage `settings.json`

Adds shared output package (`internal/cli/output`) for table + JSON rendering.

## Requirements

- Functional:
  - `typeburn history` → human-readable table (when / mode / len / WPM / acc / cons), newest first, default last 20
  - `typeburn history -n 5 --json` → JSON array, newest first
  - `typeburn config list` → table of all 4 keys with current values
  - `typeburn config list --json` → JSON object
  - `typeburn config get theme` → prints value, exit 0; unknown key exits 1
  - `typeburn config set theme nord` → strict allow-list validation → persists + exits 0
  - `typeburn config set theme zzz` (invalid) → exits 1 BEFORE write, stderr lists valid options, settings.json unchanged (F6 + validate-1)
  - `typeburn history` with empty/missing history file → table prints "no history yet"; `--json` prints `[]` (valid empty JSON array) — both exit 0 (validate-6)
- Non-functional: pipe-safety (no ANSI in non-TTY output by default)

## Architecture

```
internal/cli/
  cmd_history.go
  cmd_config.go
  output/
    table.go    # plain-text aligned table (no colors)
    json.go     # MarshalIndent helper, output to stdout
```

`output/table.go` is intentionally minimal — no external lib, just `tabwriter` from stdlib.

Config key registry (in `cmd_config.go`):

```go
var configKeys = map[string]struct {
    Get   func(s config.Settings) string
    Set   func(s *config.Settings, value string) error
}{
    "theme":          {...},
    "default_mode":   {...},
    "default_length": {...},
    "blink_cursor":   {...},
}
```

**Validation policy (F6 resolution — supersedes warn-and-coerce):** CLI `config set` is STRICT.
Per-key `Validate(value string) error` runs BEFORE `Settings.Normalize()`:
- `theme` → value must be in `theme.Names()` (canonical list); else error listing valid options
- `default_mode` → value must be in {time, words, quote, code}
- `default_length` → value must parse as positive int AND be in `LengthsFor(currentMode)` (or any mode's list if mode-agnostic; document)
- `blink_cursor` → value must be {true, false, 1, 0}

`Normalize()` is then a no-op safety net for the now-validated value. This intentionally
differs from the TUI Settings screen, which uses Normalize's silent coercion — the CLI is
scriptable and refuses bad input.

## Related Code Files

- Create: `internal/cli/cmd_history.go`, `internal/cli/cmd_history_test.go`
- Create: `internal/cli/cmd_config.go`, `internal/cli/cmd_config_test.go`
- Create: `internal/cli/output/table.go`, `internal/cli/output/table_test.go`
- Create: `internal/cli/output/json.go`, `internal/cli/output/json_test.go`

## Implementation Steps

1. Write `output/table.go`: `func Render(w io.Writer, headers []string, rows [][]string) error` using `text/tabwriter`. Test with snapshot.
2. Write `output/json.go`: `func Render(w io.Writer, v any) error` using `json.MarshalIndent(v, "", "  ") + "\n"`. Test with snapshot.
3. Implement `cmd_history.go`:
   - Flags: `-n / --limit <int>` (default 20), `--json` (bool)
   - Load via `storage.LoadHistory()`, reverse, truncate, format times as RFC3339 (table) or default (JSON)
4. Implement `cmd_config.go` with `get`, `set`, `list` subcommands and the key registry.
5. Tests for `cmd_config_test.go`:
   - Set valid value → file written; Get returns the value
   - Set invalid value (`theme zzz`) → exit 1, settings.json BYTE-IDENTICAL pre/post (file not touched)
   - Set invalid `default_mode`, `default_length`, `blink_cursor` → each exits 1
   - Use a `t.TempDir()` + override `XDG_CONFIG_HOME` env to isolate
6. Tests for `cmd_history_test.go`:
   - Missing history file + `--json` → stdout = `[]\n`, exit 0
   - Missing history file + table mode → stdout includes "no history yet", exit 0
   - Multiple records → newest-first order; `-n 1` truncates
7. Manual smoke against real `~/.config/typeburn/settings.json`.

## Success Criteria

- [ ] `history` table renders with correct columns + newest-first order
- [ ] `history --json` produces valid JSON; record fields = storage.Record JSON tags
- [ ] `config get/set/list` all 4 keys work
- [ ] Invalid `config set` rejected with helpful message listing valid options
- [ ] All tests pass; `output/` package has ≥6 tests
- [ ] No ANSI escape codes in non-TTY pipe output

## Risk Assessment

- **Risk:** Race condition: `config set` and TUI both writing settings.json concurrently.
  **Mitigation:** Existing `atomicWrite` (rename) is safe. Last-write-wins is acceptable for v2.
- **Risk:** Table rendering breaks on narrow terminals.
  **Mitigation:** Use `tabwriter` with minimum padding; no wrapping. Document min width as ≥80 chars.
- **Risk:** `--json` includes time zone surprises.
  **Mitigation:** Storage already uses RFC3339 with explicit zone; pass through unchanged.
- **Risk:** Validating `theme` against the canonical list duplicates state.
  **Mitigation:** Sync test in Phase 7 asserts CLI list == `theme.Names()` == `Settings.Normalize` accept-set.
