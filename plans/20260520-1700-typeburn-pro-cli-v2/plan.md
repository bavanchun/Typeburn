---
title: "Typeburn Pro CLI v2 — fang+cobra subcommands, headless modes, scriptable JSON"
description: >-
  Transform Typeburn's CLI from `--version`/`--text` only to a fang+cobra
  multi-subcommand surface (run/history/version/config/replay) with --help,
  --json, exit codes, back-compat aliases, and a raw-terminal --no-tui mode.
  Preserve `decide()` purity, the bare-`typeburn` TUI fall-through, and the
  pure-logic layering. Big-bang single PR. TDD where it matters (Phase 1
  refactor + Phase 5 raw-mode lifecycle).
status: completed
priority: P1
branch: "feat/pro-cli-v2"
tags: [feature, cli, refactor, release, tdd]
blockedBy: []
blocks: []
created: "2026-05-20T06:50:49.659Z"
createdBy: "ck:plan"
source: skill
---

# Typeburn Pro CLI v2 — fang+cobra subcommands, headless modes, scriptable JSON

## Overview

Predecessor: v1.5 (`typeburn`, `typeburn --version`, `typeburn --text <file>` only;
`-h` silently swallowed; everything else interactive in TUI).

This plan adds:
- 5 subcommands: `run`, `history`, `version`, `config`, `replay`
- fang-styled `--help` (cobra under the hood), `-h` works
- Top-level back-compat aliases: `--version`, `--text <file>`
- Per-subcommand `--json` for scripting
- `run --no-tui` raw-terminal mode (no Bubble Tea)
- `replay <log.json>` deterministic post-hoc metrics

Hard preserved invariants:
- `decide()`-style argument-handling purity (no `os.Exit` from parse path)
- Bare `typeburn` → TUI Home (and `typeburn <unknown-arg>` still launches TUI)
- Pure-logic packages (`typing`/`metrics`/`words`/`codetext`/`storage`/`version`)
  remain UI-free
- `-v` stays reserved (for future `--verbose`); no shortcut for version

Explicit user-approved policy break: add `github.com/spf13/cobra`,
`github.com/charmbracelet/fang`, `golang.org/x/term` despite the existing
"no new deps for core behavior" rule. CLAUDE.md is updated in Phase 7
to permit the charm ecosystem + `golang.org/x/*`.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Extract internal/runner shared session driver](./phase-01-extract-internal-runner-shared-session-driver.md) | Complete |
| 2 | [cobra+fang skeleton with bare-TUI fall-through and version subcommand](./phase-02-cobra-fang-skeleton-with-bare-tui-fall-through-and-version-s.md) | Complete |
| 3 | [run subcommand TUI mode](./phase-03-run-subcommand-tui-mode.md) | Complete |
| 4 | [history and config subcommands plus output renderers](./phase-04-history-and-config-subcommands-plus-output-renderers.md) | Complete |
| 5 | [raw-terminal --no-tui runner](./phase-05-raw-terminal-no-tui-runner.md) | Complete |
| 6 | [replay subcommand and keystroke-log JSON schema](./phase-06-replay-subcommand-and-keystroke-log-json-schema.md) | Complete |
| 7 | [docs sync and CI gates](./phase-07-docs-sync-and-ci-gates.md) | Complete |

## Completion

Implemented in working tree on 2026-05-20. Final validation passed:
`make test`, `go test ./... -race -count=1`, `make lint`, `make size-check`,
and command-level smoke checks for help, replay JSON, config set/get,
invalid config, and non-TTY `run --no-tui`.

## Dependencies

None — single greenfield feature plan on `main`. Builds on completed plans:
- `20260519-distribution-v1.5-onramp` (release infra)
- `260519-1630-settings-live-apply-and-typing-width` (settings flow)

## Acceptance Criteria (13)

1. `typeburn -h` shows fang-styled help with subcommands + examples; exit 0
2. `typeburn run --mode time --duration 30 --theme nord` launches TUI directly into a 30s time test on nord
3. `typeburn history --json | jq '.[0].wpm'` returns a number
4. `typeburn config set theme nord` persists to XDG; `typeburn config get theme` prints `nord`
5. `typeburn replay testdata/sample-keystroke-log.json --json` outputs deterministic metrics JSON without TTY
6. `typeburn run --no-tui --mode words --words 10` runs a typing test in raw stdin/stdout, restores terminal on ^C, exits 3 on abort
7. `typeburn --version` still works (back-compat alias)
8. `typeburn --text snippet.go` still works (back-compat alias)
9. `typeburn anything-unknown` still launches TUI (root fall-through); `typeburn run --bogus` exits 1
10. CI green: `go test ./... -race`, `go vet ./...`, empty `gofmt -l .`; binary growth ≤ 2.5MB
11. `typeburn config set theme zzz` exits 1, stderr lists valid options, `settings.json` unchanged
12. `typeburn run --mode code` (no `--text`) exits 1 with clear error
13. `typeburn run --no-tui --mode code` (no `--text`) exits 1 with clear error

## Release version

v2.0.0 — major bump. New top-level CLI surface; back-compat preserved (no breaking changes for v1.5 users).

## References

- `./brainstorm-summary.md` — original user-approved design
- `./research/researcher-01-cobra-fang-integration.md` — fang+cobra integration patterns, DisableFlagParsing trick
- `./research/researcher-02-x-term-raw-mode.md` — raw-mode lifecycle, signal handling, terminal restore
- `./research/scout-internal-layers.md` — extraction surface map for `internal/runner`
