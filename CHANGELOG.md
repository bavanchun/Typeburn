# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

This file is the authoritative source of GitHub Release notes (the latest
release section is extracted verbatim and passed to GoReleaser via
`--release-notes`).

## [Unreleased]

## [2.3.0] - 2026-05-30

### Added

- `typeburn update` self-update command. Downloads the matching release archive,
  verifies it against the published SHA-256 `checksums.txt` over HTTPS, extracts
  the binary, and atomically replaces the running executable in place (Linux,
  macOS, Windows). `--check` reports availability without installing; `--yes`
  skips the confirmation prompt. Package-manager builds (Homebrew, `go install`)
  are refused with the matching upgrade command and a distinct exit code;
  non-interactive streams refuse unless `--yes` is given. Integrity is
  checksum-only (unsigned binaries) — the same trust model as the install
  script.

## [2.2.0] - 2026-05-29

### Added

- Per-key error heatmap. The Result screen now shows a "most missed:" line
  listing the keys fumbled most during the test (top 8, case-folded, with
  corrected mistakes counted). The CLI surfaces the same data: `typeburn run`
  and `typeburn replay` with `--json` gain a `key_misses` array, and the
  metrics table gains `most_missed_*` rows. Computed post-hoc from the
  keystroke log — no persistence and no history schema change.

## [2.1.3] - 2026-05-29

### Fixed

- `typeburn run --no-tui` in Time mode now ends automatically at the time limit.
  Previously the raw runner only checked completion after a keystroke, so a Time
  test would hang until the next keypress (indefinitely in piped/non-interactive
  runs); a keystroke arriving after the limit could also inflate the final
  metrics. The runner now completes on the clock with zero trailing input.
- `typeburn version --check-update` now prints a release URL only when it points
  at the official repository, matching the validation already applied to cached
  results.

## [2.1.2] - 2026-05-25

### Changed

- Internal tooling and documentation housekeeping; no user-visible changes.

## [2.1.1] - 2026-05-22

### Fixed

- `--no-tui` live WPM: O(n²) recompute replaced with O(1) incremental
  `metrics.LiveWPM` — eliminates lag on long Time-mode tests.
- `typeburn version --json --check-update`: double-emitted error on network
  failure now silenced; JSON output is always valid.
- `stripANSI`: non-SGR CSI sequences (e.g. cursor-movement) no longer corrupt
  the Result panel border-title width calculation.
- Update cache: pre-release builds no longer exhaust the 800 ms timeout budget
  on every TUI launch.
- `--no-tui` ESC parser: split-read escape sequences no longer leave a stray
  `[` character in the typed output.
- `internal/update` package globals: synchronized via mutex accessors,
  eliminating a latent data race.

## [2.1.0] - 2026-05-21

### Added

- **Update check** (`internal/update`): pure-stdlib package that fetches the
  latest GitHub release, caches the result for 24 h under
  `$XDG_STATE_HOME/typeburn/update-check.json`, and compares semver.
- `typeburn version --check-update` explicit flag: always hits the network,
  prints a human-readable or `--json` result regardless of config.
- `update_check` config key (boolean, default `false`, opt-in):
  `typeburn config set update_check on`. When enabled, every TUI launch
  performs an opportunistic background check (800 ms timeout); if a newer
  stable release is found, the Result screen shows a muted footer hint.
- Result-screen update-available footer: `↑ v2.1.0 available — run "typeburn version --check-update"`.
  Belt-and-suspenders semver injection guard at both cache-load and render time.

### Changed

- Size cap raised from 8 MiB → 10 MiB (net/http adds ~260 KB).
- `typeburn config list` and `typeburn config set` now include `update_check`.
- `parseBool` now accepts `on`/`off`/`yes`/`no` in addition to `true`/`false`/`1`/`0`.

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

## [1.5.0] - 2026-05-20

### Added

- **One-line installer for Linux and macOS.** No Go toolchain required:

  ```sh
  curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
  ```

  It detects your OS/arch, downloads the matching release archive, **verifies
  its sha256 against `checksums.txt`**, and installs `typeburn` into
  `~/.local/bin` (no sudo). `BIN_DIR=` and `VERSION=` overrides are supported,
  and the README documents a non-piped audit path plus the honest trust
  boundary (the checksum defends the download, not a compromised release).
- **Homebrew cask.** Install or upgrade via Homebrew on macOS/Linux:

  ```sh
  brew install bavanchun/tap-typeburn/typeburn
  ```

  The cask wraps the prebuilt release archive — no Go/Xcode toolchain needed.

### Changed

- **`go install` now requires Go 1.25+** (was 1.26+). The effective floor is
  set by direct dependencies; 1.25 is the lowest the module graph allows.
  Pre-built binaries, the installer, and the Homebrew cask need no Go at all.

## [1.4.0] - 2026-05-19

### Fixed

- **Theme and other Settings changes now apply immediately.** Changing the
  theme (or blink cursor, or default mode / length) on the Settings screen
  previously had no visible effect until you quit and relaunched — the new
  value was written to disk but never applied to the running session.
  Settings changes now take effect live across every screen, and the
  Settings row you just changed stays selected.

### Changed

- **The typing text uses more of a wide terminal and is vertically
  centered.** On wide terminals the typing line was capped at 80 columns
  and anchored to the top with the footer pinned to the very bottom,
  leaving the text small and lost in empty space. It now scales to about
  82% of the terminal width (never narrower than before, and it grows with
  the screen) and the whole block is centered, so the text fills the
  screen more comfortably. Narrow and mid-size terminals are unchanged.
  (On-screen character size itself is set by your terminal's font, not the
  app.)

## [1.3.0] - 2026-05-19

### Added

- **In-app paste for Code mode:** you no longer need `typeburn --text <file>`
  to practice on your own snippet. On the Home screen, tab to **Code** and
  press enter on the empty row to open a paste screen; bracket-paste your
  snippet and it is validated and loaded in place — press enter again to
  start the test. `--text` is still supported and takes precedence (when a
  snippet is supplied that way, Code is enabled and enter starts immediately;
  the paste screen is not used that run).
- Pasted snippets go through the **same** normalization and caps as `--text`
  (CRLF→LF, one trailing newline trimmed, UTF-8 BOM stripped; empty,
  non-text, or oversized &gt;10k runes / &gt;500 lines input is rejected with
  a clear reason and you can paste again). `esc` returns to Home unchanged.
  Code runs still save to History but never count toward the ★ personal
  best. `NO_COLOR` behavior is unchanged.

## [1.2.0] - 2026-05-19

### Added

- **Code mode:** practice typing on your own text or code. Supply it with
  `typeburn --text <file>` or pipe via `typeburn --text -`. The snippet is
  rendered with full-literal layout — real line breaks, tabs shown as two
  columns, and you type every space and indentation exactly; the test
  completes on an exact full-text match. Long snippets scroll with a
  caret-following viewport. Code mode is always listed in the mode tabs;
  without `--text` it shows a hint and is disabled (in-app paste is planned).
  `NO_COLOR` behavior is unchanged.
- Code runs are saved to History but never count toward the ★ personal
  best (custom text is not comparable run-to-run). Oversized
  (&gt;10k runes / &gt;500 lines), empty, or non-text input is rejected with
  a clear reason instead of starting.

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

[Unreleased]: https://github.com/bavanchun/Typeburn/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.0.0
[1.5.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.5.0
[1.4.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.4.0
[1.3.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.3.0
[1.2.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.2.0
[1.1.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.1.0
[1.0.1]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.1
[1.0.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0
