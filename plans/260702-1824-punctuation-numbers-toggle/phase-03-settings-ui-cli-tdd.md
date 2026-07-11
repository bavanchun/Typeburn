---
phase: 3
title: Settings UI & CLI (TDD)
status: completed
effort: medium
priority: P1
dependencies:
  - 1
  - 2
---

# Phase 3: Settings UI & CLI (TDD)

## Overview

Expose the two new settings as toggle rows in `internal/ui/screen_settings.go`
(same on/off cycle as `rowStrictMode`) and as `config get`/`config set` CLI
keys in `internal/cli/cmd_config.go` (same pattern as `strict_mode`).

## Requirements

- Functional:
  - Settings screen: two new rows, e.g. `rowPunctuation`, `rowNumbers`, cycled
    on/off via the existing `Cycle` binding, positioned after `rowStrictMode`
    in the row order (confirm exact row enum/order in `screen_settings.go`
    before editing).
  - `screen_settings_view.go`: render the two new rows following the existing
    label/value column layout — no new visual pattern.
  - CLI: `typeburn config get punctuation` / `config get numbers`, `config set
    punctuation <true|false>` / `config set numbers <true|false>`, with the
    same invalid-value rejection behavior as `strict_mode` (see
    `strict_mode_config_test.go` `TestStrictModeSetGet` for the exact
    assert shape to mirror).
  - `typeburn config list` output includes both new keys (mirror
    `TestConfigListIncludesStrictMode`).
- Non-functional: toggles must persist via existing `AppendHistory`-adjacent
  settings-save path (auto-persist on Settings screen `esc`/`1`, per README's
  documented behavior) — no new persistence mechanism.

## Architecture

Zero new architecture — this phase is purely extending two already-proven
enum-driven patterns (`screen_settings.go` row cycling, `cmd_config.go` key
table) by two entries each.

## Related Code Files

- Modify: `internal/ui/screen_settings.go` (row enum, `Cycle` handler — mirror
  `case rowStrictMode:` block at line ~132)
- Modify: `internal/ui/screen_settings_view.go` (render new rows)
- Modify: `internal/ui/screen_settings_test.go` (TDD: RED first — new row
  cycle tests)
- Modify: `internal/cli/cmd_config.go` (get/set key table — mirror lines ~93,
  ~145 `strict_mode` entries)
- Create: `internal/cli/punctuation_numbers_config_test.go` (mirror
  `strict_mode_config_test.go` structure: `TestPunctuationNumbersSetGet`,
  extend `TestConfigListIncludesStrictMode`-equivalent to assert both new keys
  present in `config list`)

## Implementation Steps

1. **RED**: Write `screen_settings_test.go` cases — cycling `rowPunctuation`/
   `rowNumbers` toggles the corresponding `Settings` field; verify against the
   existing `rowStrictMode` test as the template.
2. **RED**: Write `punctuation_numbers_config_test.go` — copy
   `TestStrictModeSetGet` structure for both `punctuation` and `numbers` keys;
   copy `TestConfigListIncludesStrictMode` asserting both new keys appear.
3. Confirm RED (missing row/key → test failure, not compile error, since
   Settings struct fields already exist from phase 1).
4. **GREEN**: Add `rowPunctuation`/`rowNumbers` to the settings row enum +
   `Cycle` handler in `screen_settings.go`.
5. **GREEN**: Add render lines in `screen_settings_view.go`.
6. **GREEN**: Add `punctuation`/`numbers` entries to `cmd_config.go`'s
   get/set/list key table.
7. Run `go test ./internal/ui/... ./internal/cli/... -race -count=1` —
   confirm GREEN, including any teatest golden files for the Settings screen
   (may need golden regeneration if the screen's rendered row count changed —
   check `make test` output for golden mismatches and regenerate deliberately,
   not blindly).
8. `gofmt -l .` and `go vet ./...` clean.

## Success Criteria

- [x] Settings screen has two new cycleable rows, positioned after Strict Mode
- [x] Toggling either row updates `m.s.Punctuation`/`m.s.Numbers` and persists
      on screen exit (existing auto-save path)
- [x] `config get/set punctuation|numbers` works, rejects invalid values
- [x] `config list` includes both new keys
- [x] No Settings-screen teatest golden files existed for this screen — no
      regeneration needed (confirmed: `find . -iname "*settings*golden*"` empty)
- [x] Full `go test ./... -race -count=1`, `gofmt -l .`, `go vet ./...` green

## Risk Assessment

- **Risk:** Settings screen teatest golden files break due to added rows
  changing the rendered height/layout.
  **Mitigation:** Expected — regenerate goldens deliberately per existing
  project convention (teatest golden update flow), verify diff by eye before
  committing, don't rubber-stamp.
- **Risk:** CLI invalid-value handling diverges from `strict_mode`'s error
  message/behavior, creating inconsistent UX across config keys.
  **Mitigation:** Directly copy `strict_mode`'s validation branch in
  `cmd_config.go`, only swapping the key name and target field.
