## [2.1.0] - 2026-05-21

### Added

- **Opt-in update check.** Enable with `typeburn config set update_check on`.
  Each TUI launch fires a background check (800 ms timeout); if a newer stable
  release is found the Result screen shows a muted footer hint:
  `↑ v2.1.0 available — run "typeburn version --check-update"`.
- `typeburn version --check-update [--json]` — explicit network check that
  always bypasses the 24 h cache.
- `internal/update` package: pure stdlib (no bubbletea/lipgloss), 24 h
  XDG-state cache with semver injection guard, GitHub API client with
  redirect-block and 1.5 s total timeout.

### Changed

- `typeburn config set update_check on|off` (also accepts `true/false/yes/no/1/0`).
- `typeburn config list` now includes `update_check` (default `false`).
- Binary size cap raised 8 MiB → 10 MiB (net/http adds ~260 KB).

[2.1.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0
