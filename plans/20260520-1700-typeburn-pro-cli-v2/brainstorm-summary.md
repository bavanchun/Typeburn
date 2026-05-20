# Brainstorm Summary — Typeburn Pro CLI v2

**Date:** 2026-05-20
**Status:** Approved by user (big-bang, single PR)
**Predecessor:** v1.5 (only `--version` + `--text`; no `-h`, no subcommands)

## Problem Statement

Current CLI surface is minimal/under-professional:
- No `--help` / usage (silently swallowed by `ContinueOnError` + `io.Discard`)
- No subcommands; scripting only via flags
- No headless / scriptable result emission
- All gameplay, config, history locked behind interactive TUI

User wants a "professional CLI" that is scriptable, discoverable, and idiomatic — while preserving the deliberate fall-through UX where `typeburn <anything>` still launches the TUI.

## Requirements (locked)

| # | Item | Value |
|---|------|-------|
| 1 | Expected output | Multi-subcommand `typeburn` binary (run / history / version / config / replay) with fang-styled help, --json modes, headless replay + raw-stdin runner |
| 2 | Acceptance | See "Acceptance Criteria" below — 10 concrete items |
| 3 | Scope OUT | Completions in archives, man page in archives, `config edit`, themed pipe output |
| 4 | Constraints | Single PR ("big bang"); preserve `decide()` purity; preserve `typeburn <bare>` → TUI; back-compat `--version` and `--text` aliases; `-v` stays reserved for future `--verbose`; no shortcut for `--version`; `-h` → help (cobra default) |
| 5 | Touchpoints | `main.go`, new `internal/cli/`, new `internal/cli/notui/`, new `internal/cli/output/`, new `internal/runner/`, references to `internal/typing`, `internal/metrics`, `internal/storage`, `internal/config`, `internal/codetext`, `internal/version` |

## Evaluated Approaches

### A. stdlib `flag` + hand-written dispatcher (rejected)
- **Pros:** Honors stated "no new deps" rule. Smallest binary.
- **Cons:** Hand-rolled help is ugly + drifts. No completions. ~150 LOC of plumbing the user has to maintain.

### B. `kong` (rejected)
- **Pros:** Lightweight (~300KB). Struct-tag declarative.
- **Cons:** Doesn't match the charm aesthetic the project leans on. No man page generator.

### C. `charmbracelet/fang` + `cobra` (CHOSEN)
- **Pros:** Styled help/errors match Lip Gloss aesthetic; auto man page + shell completions; same vendor ecosystem (`charm.land/bubbletea` already a dep); industry-standard cobra under the hood.
- **Cons:** ~1.5MB binary growth; cobra is heavier than needed for 5 subcommands; explicitly breaks the project's documented "no new deps for core behavior" rule (user accepted; rule will be revised in CLAUDE.md to allow charm-ecosystem + `golang.org/x/*` deps).

## Final Solution

### Layout

```
main.go                          # thin: fang.Execute(cli.NewRoot())
internal/cli/
  root.go                        # cobra root; --version & --text aliases; routes bare-args → TUI
  cmd_run.go                     # `typeburn run` (flags → settings; TUI or --no-tui)
  cmd_history.go                 # `typeburn history` (table | --json)
  cmd_version.go                 # `typeburn version` (text | --json)
  cmd_config.go                  # `typeburn config get|set|list`
  cmd_replay.go                  # `typeburn replay <log.json>` (deterministic metrics)
  exitcodes.go                   # 0/1/2/3/4
  notui/
    runner.go                    # x/term raw mode + signal restore
    render.go                    # ANSI prompt drawing
  output/
    table.go
    json.go
internal/runner/                 # NEW shared driver
  session.go                     # builds a session from settings; used by TUI screen_typing.go AND notui
```

### CLI Surface

```
typeburn                              # bare → TUI Home (unchanged)
typeburn run [--mode … --duration … --words … --theme … --text … --no-tui --json]
typeburn history [-n N] [--json]
typeburn version [--json]
typeburn config get <key>
typeburn config set <key> <value>
typeburn config list [--json]
typeburn replay <keystroke-log.json> [--json]
typeburn -h | --help                  # fang-styled
typeburn --version                    # back-compat alias for `version`
typeburn --text <file>                # back-compat alias for `run --text <file>`
```

### Exit Codes

| 0 | success |
| 1 | usage error (only after recognized subcommand) |
| 2 | I/O error |
| 3 | user abort (^C mid-test) |
| 4 | internal error |

### Invariants Preserved

- `decide()` purity contract maintained (custom cobra error handler; no `os.Exit` from parse path)
- TUI Elm message flow unchanged (`internal/app`)
- Pure logic packages (`typing`, `metrics`, `words`, `codetext`, `storage`, `version`) stay UI-free
- `typing.Engine` + `metrics.Compute` reused by TUI runner, `--no-tui` runner, and `replay` — via new `internal/runner` shared driver
- Storage/settings JSON schema unchanged
- Bare `typeburn <unknown-arg>` still launches TUI (root-level fall-through)
- `-v` remains reserved for future `--verbose`

### Acceptance Criteria

1. `typeburn -h` → fang-styled help, exit 0
2. `typeburn run --mode time --duration 30 --theme nord` → TUI launches directly into 30s time test on nord
3. `typeburn history --json | jq '.[0].wpm'` → number
4. `typeburn config set theme nord` persists to XDG; `typeburn config get theme` prints `nord`
5. `typeburn replay testdata/sample-log.json --json` → deterministic metrics JSON, no TTY required
6. `typeburn run --no-tui --mode words --words 10` → raw-stdin typing test; ^C restores terminal and exits 3
7. `typeburn --version` still works (back-compat alias)
8. `typeburn --text snippet.go` still works (back-compat alias)
9. `typeburn anything-unknown` still launches TUI (root fall-through); `typeburn run --bogus` exits 1
10. `go test ./... -race`, `go vet ./...`, empty `gofmt -l .` — all green; binary growth ≤ 2.5MB

### Dependencies Added (explicit policy break, user-approved)

- `github.com/spf13/cobra`
- `github.com/charmbracelet/fang`
- `golang.org/x/term`

**Policy change required:** update `CLAUDE.md` "Conventions & Constraints" section to permit charm-ecosystem + `golang.org/x/*` deps. To be done in the plan as a documented decision, not silently.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Raw-terminal `--no-tui` leaves TTY broken on panic | `defer term.Restore()` + SIGINT/SIGTERM/SIGQUIT handlers + integration test on linux/macOS |
| cobra/fang call `os.Exit` from parse — breaks `decide()` purity contract | Custom `SilenceErrors`+`SilenceUsage`; explicit error handler; pure decide-replacement still unit-tested |
| `--no-tui` paste detection unreliable across terminals | Document as known limitation; bracket-paste opt-in only via explicit flag |
| Binary bloat regression | Add `make size-check` asserting ≤ size threshold; CI gate |
| Back-compat alias ambiguity (`--text` at root vs `run --text`) | Root `--text` internally invokes `run --text`; integration test covers both |
| Big-bang PR review burden | Split commits per phase even though one PR; reviewer can walk commits |

## Implementation Considerations

- New `internal/runner` package is the critical abstraction; must be designed before any cmd_*.go file. Extracting it from current `internal/app`/`internal/ui` is the first phase.
- `notui/` rendering should reuse `internal/theme` color roles where terminal supports color, and degrade gracefully under `NO_COLOR`.
- `replay` keystroke-log format must match what TUI could optionally dump in a future flag — keep the on-disk schema forward-compatible from day one (`schema_version: 1` field).
- Phases (suggested for `/ck:plan`):
  1. Extract `internal/runner` from TUI; tests pass unchanged
  2. Add cobra+fang skeleton with root, version, --version alias, --help; ensure bare-launch + back-compat
  3. Implement `run` subcommand (TUI path), `history`, `config`
  4. Implement `notui/` raw-terminal runner; `run --no-tui`
  5. Implement `replay` subcommand; lock keystroke-log schema
  6. Docs: README, CONTRIBUTING dep-policy revision, CLAUDE.md update, exit code table
  7. CI: `make size-check`; integration test matrix linux+macOS

## Success Metrics

- `typeburn -h` exists and is useful (manual review)
- Binary size ≤ +2.5MB vs main HEAD
- CI race + vet + gofmt + integration tests green
- All 10 acceptance criteria pass
- No regression on existing v1.5 surface (back-compat smoke test)

## Next Steps

Hand off to `/ck:plan` with this summary as context. Recommend default `/ck:plan` (not `--tdd`) — most code is new (cli/, runner/, notui/, output/) rather than refactor of behavior. TDD shines for the `internal/runner` extraction phase specifically, but doesn't need to dominate the entire plan.

## Unresolved Questions

- Should `replay` accept the keystroke-log emitted by a future TUI-dump flag (`typeburn run --dump-log out.json`)? Schema is forward-compatible per design but the dump-flag itself is OUT of scope this round — defer to a v2.1 round.
- Should `config set` validate value against a schema (e.g. theme must be a known theme name)? Yes recommended; defer specifics to plan phase 3.
- macOS code-signing for the now-larger binary — orthogonal, defer.
