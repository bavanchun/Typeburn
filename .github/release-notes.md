## [2.0.0] - 2026-05-20

### Added

- Professional CLI surface via cobra/fang: `run`, `history`, `version`,
  `config`, and `replay` subcommands, with styled `-h` / `--help`.
- Scriptable JSON output for `history`, `version`, `config list`, `replay`,
  and raw `run --no-tui --json` results.
- Raw terminal runner: `typeburn run --no-tui --mode words --words 10`.
  It restores terminal state on normal completion, panic, and handled aborts.
- Schema-versioned keystroke replay fixture and parser
  (`schema_version: 1`).

### Changed

- `main.go` is now a thin fang/cobra entrypoint. The v1 aliases
  `--version` and `--text <file>` still work, and root-level unknown args still
  fall through to the TUI.
- Session construction moved into `internal/runner` so the TUI, replay, and
  raw runner share target/engine setup.
- Dependency policy now explicitly permits cobra, fang, and `golang.org/x/*`
  for the CLI surface.

[2.0.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.0.0
