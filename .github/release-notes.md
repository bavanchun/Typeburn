## [2.7.0] - 2026-08-01 — Animated Update CLI

### Added

- **`typeburn update` now animates.** On a terminal at least 56 columns wide,
  the update renders a bordered checklist — checksums, downloading, verifying,
  installing — with a spring-smoothed gradient progress bar driven by the real
  byte count of the release archive. Built on `charm.land/bubbles/v2`.
- The checksums fetch, which the updater always performed, is now reported as
  its own step instead of happening invisibly.
- Interrupting with `ctrl+c` during the download stops the update and waits for
  it to unwind, so the update lock and the partial download are removed rather
  than left behind. The interrupt is deliberately refused once installing
  begins, because the remaining work is the atomic binary swap. Only the
  download is cancellable, so an interrupt arriving during verification or the
  swap reports `stopped too late — <from> → <to> was already installed` rather
  than falsely claiming nothing changed.

### Changed

- Plain `typeburn update` output — used for pipes, redirects, CI, and terminals
  too narrow for the frame — is unchanged apart from the new `checksums...`
  line, and is now pinned by an exact-output test.

No config, storage, or release-archive contract changes. The plain
`typeburn update` output used by pipes, redirects, and CI is unchanged apart
from the new `checksums...` line. Existing history and settings files are fully
compatible.
