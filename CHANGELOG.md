# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is the authoritative source of GitHub Release notes (the latest
release section is extracted verbatim and passed to GoReleaser via
`--release-notes`).

## [Unreleased]

## [1.1.0] - 2026-05-19

### Added

- **Theme packs:** six new color themes selectable in Settings —
  `solarized-dark`, `solarized-light`, `dracula`, `nord`, `gruvbox-dark`,
  `gruvbox-light` (bringing the total to eight, with `default` and `mono`).
  `solarized-light` and `gruvbox-light` are the first light themes. `NO_COLOR`
  behavior is unchanged.
- **Persistence-failure notice:** if saving a result or settings to disk
  fails, a dismissible one-line notice now appears instead of the failure
  being silent. It never blocks input and clears on the next keypress.

### Changed

- Documentation corrected post-1.0: removed the stale "badges 404 until the
  first tag" note; the dependency-layering rule now describes the real
  invariant (`config`/`theme` intentionally bridge to the TUI); the word
  stream's wrap comment now matches the actual character-cell behavior.

## [1.0.1] - 2026-05-18

### Fixed

- **New-best precision:** the ★ personal-best badge compared rounded integer
  WPM, so a strictly faster run that rounded to the same integer (e.g. 75.4 vs
  75.0) did not earn the badge. New-best detection now compares the full-precision
  net WPM. History records written by v1.0.0 (which lack the new field) fall
  back to their stored rounded WPM, so existing personal bests are preserved.

### Removed

- **`missed` stat:** the result screen showed a `missed` counter that was always
  `0` (the metrics package never received the target text to compute it). The
  unusable field and its display were removed; no real metric is affected.

## [1.0.0] - 2026-05-18

First public release. A distraction-free, keyboard-driven Monkeytype-style
terminal typing test built with Go and Bubble Tea v2.

### Added

- **Three test modes:** Time (15/30/60/120 s), Words (10/25/50/100 words),
  Quote (short/medium/long/epic).
- **Live metrics:** net/raw WPM, accuracy, consistency, and CPS, recomputed
  every keystroke; per-second bucketing with an end-of-test sparkline chart.
- **Five screens:** Home (mode/length picker with logo), Typing (live test),
  Result (big-digit WPM, sparkline, full character breakdown), Settings
  (4 settings, live preview), History (scrollable table, per-mode ★ best).
- **Persistence:** XDG-compliant `settings.json` and `history.json` (newest
  200 records) with atomic writes.
- **Themes:** `default` (dark + green accent) and `mono` (attribute-only),
  plus full `NO_COLOR` support (bold/underline/faint only).
- **Per-screen keymap:** Vim-style navigation across all screens.
- **Resize handling:** graceful degraded notice below 60×20, auto-resume.
- **CI:** GitHub Actions on ubuntu + macOS — build → vet → gofmt → race tests.
- **Release engineering:** `internal/version` build-stamp package with a
  `--version` flag (ldflags-injected, `debug.ReadBuildInfo()` fallback),
  GoReleaser cross-platform binaries (linux/darwin/windows × amd64/arm64),
  and a self-gating tag-triggered release workflow.

### Fixed

- **Timer re-arm on tab-restart (M1):** in Time mode, restarting a test with
  `tab` left the header WPM/elapsed frozen until the next keystroke;
  `restartSame()` now re-arms the timer tick like a fresh test.

### Security

- Release binaries are **not** cryptographically signed. Integrity rests on
  HTTPS transport and the pipeline-generated `checksums.txt`. See
  [SECURITY.md](./SECURITY.md).

[Unreleased]: https://github.com/bavanchun/Typeburn/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.1.0
[1.0.1]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.1
[1.0.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0
