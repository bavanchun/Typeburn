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
[1.0.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0
