---
phase: 3
title: "run subcommand TUI mode"
status: completed
priority: P1
effort: "4h"
dependencies: [1, 2]
---

# Phase 3: `run` subcommand (TUI mode)

## Overview

Add `typeburn run [flags]` that launches the TUI directly into the Typing
screen with a pre-configured session (mode/length/theme/text) — skipping
Home. The `--no-tui` flag is wired in Phase 5; this phase ships only the
TUI path.

## Requirements

- Functional:
  - `typeburn run` → TUI starts in Typing screen with current persisted defaults
  - `typeburn run --mode time --duration 30` → 30s time mode
  - `typeburn run --mode words --words 25` → 25-word mode
  - `typeburn run --mode quote` → quote mode (medium bucket default)
  - `typeburn run --mode code --text snippet.go` → code mode from file
  - `typeburn run --theme nord` → temporary theme override (NOT persisted)
- Functional: invalid combinations (e.g. `--mode time --words 25`) exit 1 with a clear message
- Non-functional: persisted settings unchanged by `run` flags (override is per-invocation)

## Architecture

```
internal/cli/
  cmd_run.go         # `run` cobra cmd + flag parsing + validation + TUI launch
  cmd_run_test.go    # flag parsing + validation table tests
```

Flag → session mapping:

| Flag | Type | Validation | Maps to |
|------|------|-----------|---------|
| `--mode` | string | one of {time,words,quote,code} | session mode |
| `--duration` | int | time mode only; positive | session length (seconds) |
| `--words` | int | words mode only; positive | session length |
| `--quote-len` | string | quote mode only; one of {short,medium,long} | words.QuoteLen |
| `--theme` | string | one of `theme.Names()` (added in this phase) | temporary theme |
| `--text` | string | code mode only; file path or "-" | reuses `codetext.Load` (SAME helper that root-level `--text` alias uses — single code path, no duplicate impls per F4) |
| `--no-tui` | bool | wired Phase 5; here just registered | -- |
| `--json` | bool | with `--no-tui` only; wired Phase 5 | -- |

Build flow:

```
parse flags → validate combo → settings := persistedSettings.Override(flags)
            → theme := theme.Load(name)
            → codeText := codetext.Load(--text) (if mode=code)
            → model := app.New(theme, settings, codeText, codeHint)
            → initCmd := func() tea.Msg { return ui.StartTestMsg{Mode, Length, QuoteLen, CodeText} }
            → tea.NewProgram(model, tea.WithInitialCommand(initCmd)).Run()
```

**Skip-Home approach (F20 resolution):** Reuse existing `app.New(...)` constructor; inject a
`StartTestMsg` as the program's initial command. The root model's existing Update routes it
into Typing screen (`internal/app/model.go:86-98`). No new constructor needed; smaller surface
than adding `app.NewFromCLI`.

**Code mode validation (validate-7):** `--mode code` REQUIRES `--text <file>` — never falls
back to the in-app paste screen from CLI. Empty `--text` with `--mode code` exits 1 with a
clear message pointing the user to `--text snippet.go` or the bare-TUI flow.

**Theme override semantics (F7):** `--theme` only seeds the initial theme passed to `app.New`.
If the user opens the Settings screen mid-test and changes the theme, the persisted Settings
change wins (existing Elm `SettingsChangedMsg` flow). Document in `docs/cli-reference.md`:
"`--theme` is initial-only; in-TUI changes override it." No additional code.

## Related Code Files

- Create: `internal/cli/cmd_run.go`, `internal/cli/cmd_run_test.go`
- Modify: `internal/cli/root.go` (export the shared `--text` loader so both root alias and `run --text` use it)
- Modify: `internal/theme/theme.go` (add `Names() []string` if not present)
- NO change to `internal/app/model.go` — reuse `app.New` + initial `StartTestMsg` instead of a new constructor (F20).

## Implementation Steps

1. Add `theme.Names() []string` (UI-free; just the canonical list — duplicates `Settings.Normalize`'s list, with a sync test in Phase 7).
2. Implement `internal/cli/cmd_run.go`:
   - Define flags
   - `RunE`: parse → `validateRunFlags()` pure fn → build session → launch via `tea.NewProgram(model, WithInitialCommand(...)).Run()`
3. `validateRunFlags()` MUST reject:
   - `--mode code` without `--text` (validate-7)
   - `--mode time --words N` (mutex)
   - `--mode words --duration N` (mutex)
   - `--theme X` where X ∉ `theme.Names()`
4. Tests in `cmd_run_test.go`:
   - Validate good combos (table, ≥10 cases)
   - Validate bad combos: time+words, words+duration, code without --text, unknown theme
   - File-load error path → message returned (does not panic)
   - Skip-Home injection: assert `tea.NewProgram` constructed with an initial command that produces `ui.StartTestMsg` matching the parsed flags
5. Manual smoke: `typeburn run --mode time --duration 15 --theme nord`

## Success Criteria

- [ ] All 5 flag combos in Requirements work end-to-end
- [ ] Invalid combos exit 1 with clear stderr message
- [ ] Persisted settings unchanged after `run --theme x`
- [ ] All existing tests pass
- [ ] `cmd_run_test.go` covers ≥10 table cases

## Risk Assessment

- **Risk:** Skip-Home initial-command path desyncs with normal Home → Typing transition.
  **Mitigation:** Reuse the existing `ui.StartTestMsg` flow (`internal/app/model.go:86-98`) via `tea.WithInitialCommand`. No new constructor (F20 resolution). Cover with a teatest that boots into Typing screen and asserts identical state vs Home-then-Enter.
- **Risk:** `--theme` override leaks into persisted settings.
  **Mitigation:** Pass an in-memory `Settings` copy; never call `SaveSettings` from `run`.
- **Risk:** `--text -` (stdin) collides with code-paste TUI screen.
  **Mitigation:** `--text -` loads via `codetext.Load("-")` then enters Typing directly; never opens code-paste screen.
