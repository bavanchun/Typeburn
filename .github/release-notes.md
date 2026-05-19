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

[1.5.0]: https://github.com/bavanchun/Typeburn/releases/tag/v1.5.0
