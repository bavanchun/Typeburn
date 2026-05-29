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

[2.3.0]: https://github.com/bavanchun/Typeburn/releases/tag/v2.3.0
