# Typeburn

[![CI](https://github.com/bavanchun/Typeburn/actions/workflows/ci.yml/badge.svg)](https://github.com/bavanchun/Typeburn/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/bavanchun/Typeburn?sort=semver)](https://github.com/bavanchun/Typeburn/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/bavanchun/Typeburn)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/bavanchun/Typeburn.svg)](https://pkg.go.dev/github.com/bavanchun/Typeburn)
[![License](https://img.shields.io/github/license/bavanchun/Typeburn)](./LICENSE)

A Monkeytype-style terminal typing test built with Go and Bubble Tea v2.
Distraction-free, keyboard-driven, and works on any ANSI terminal.

## Features

- **Four test modes**: Time (15/30/60/120 s), Words (10/25/50/100 words), Quote (short/medium/long/epic), Code (your own text via `--text` or in-app paste)
- **Live stats**: WPM, raw WPM, accuracy, and consistency updated every keystroke
- **Result screen**: big-digit WPM, sparkline chart, full char breakdown
- **History**: scrollable table of all past tests with per-mode best marker (★)
- **Themes**: `default` (dark, green accent) and `mono` (attribute-only, no color codes)
- **NO_COLOR support**: set `NO_COLOR=1` for a fully attribute-only render (bold/underline/faint only)
- **Minimum terminal**: 60 columns × 20 rows; graceful degraded notice below that
- **XDG-compliant paths**: settings and history go to `$XDG_CONFIG_HOME` / `$XDG_DATA_HOME`

## Installation

**1. Quick install (Linux/macOS, no Go toolchain):**

```sh
curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
```

Detects your OS/arch, downloads the matching release archive, **verifies its
sha256 against `checksums.txt`**, and installs `typeburn` into `~/.local/bin`
(no sudo). Override the target with `BIN_DIR=…` or pin a tag with `VERSION=vX.Y.Z`.

> **Trust boundary — read before piping any script to a shell.** The sha256
> check defends against a corrupted or man-in-the-middled *download*. It does
> **not** make `curl … | sh` inherently safe: the script, the archive, and
> `checksums.txt` all come from the same GitHub release (and `checksums.txt`
> is unsigned), so a compromised release would be self-consistent. If that
> boundary matters to you, use the non-piped audit path instead:
>
> ```sh
> curl -fsSL -o install.sh https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh
> less install.sh        # read it
> sh install.sh          # then run it
> ```
>
> Windows is not covered by `install.sh` (POSIX `sh` only) — use the manual
> archive below.

**2. `go install` (latest tagged release):**

```sh
go install github.com/bavanchun/Typeburn@latest
```

> The module path is **case-sensitive — the capital `T` in `Typeburn` is
> required**. `go install` installs an executable named `Typeburn` into
> `$(go env GOPATH)/bin`.
>
> A freshly published release may lag the Go module proxy by up to ~1 hour;
> until the proxy ingests the tag, `go install ...@vX.Y.Z` can 404. Downloading
> the release binary (below) is immediate and unaffected.

Requires **Go 1.25+**.

**3. Download a pre-built binary:**

Grab the archive for your OS/arch from the
[latest release](https://github.com/bavanchun/Typeburn/releases/latest)
(linux/darwin/windows × amd64/arm64), verify it against `checksums.txt`,
extract, and run the `typeburn` binary (the release archives ship a lowercase
`typeburn`; only the `go install` path above produces `Typeburn`).

**4. Build from source:**

```sh
make build            # → ./bin/typeburn (lowercase, local convention)
# or
go build -o typeburn .
make run               # run without installing (or: go run .)
```

**5. Homebrew (macOS/Linux):**

```sh
brew install bavanchun/tap-typeburn/typeburn
```

A cask wrapping the prebuilt release archive (no Go/Xcode toolchain needed).

## Usage

```sh
typeburn                         # open the TUI Home screen
typeburn -h                      # styled help with subcommands
typeburn run --mode time --duration 30 --theme nord
typeburn history --json
typeburn config set theme nord
typeburn replay testdata/sample-keystroke-log.json --json
typeburn --version               # v1-compatible alias
typeburn --text snippet.go       # v1-compatible Code mode alias
```

In **Code mode** you type the supplied text exactly — every space, tab, and
line break — and the test finishes on an exact match. Without `--text`, tab to
the Code row and press enter to open the in-app paste screen, bracket-paste a
snippet, then press enter to start. Code runs appear in History but never set
a ★ personal best.

See [docs/cli-reference.md](./docs/cli-reference.md) for the full subcommand
surface, JSON shapes, exit codes, and raw `--no-tui` limitations.

### Check for updates

```sh
typeburn version --check-update          # always hits network, human-readable
typeburn version --check-update --json   # machine-readable JSON
```

Opt-in automatic check on TUI launch (result cached 24 h):

```sh
typeburn config set update_check on
```

When enabled, every TUI launch runs a background check with an 800 ms timeout.
If a newer stable release is found, the Result screen shows a muted footer hint:
`↑ v2.1.0 available — run "typeburn version --check-update"`.

### Self-update

```sh
typeburn update            # confirm, then download + install the latest release
typeburn update --check    # report availability only, never installs
typeburn update --yes      # skip the confirmation prompt (needed for non-interactive use)
```

`update` downloads the matching release archive, verifies it against the
published SHA-256 `checksums.txt` over HTTPS, then atomically replaces the
running binary in place. Integrity rests on TLS + checksums — the **same trust
model as `curl install.sh | sh`**. Release binaries are unsigned, so this
detects a corrupted or truncated download, **not** a compromised release host
(see [SECURITY.md](./SECURITY.md)).

Builds installed by a package manager are **not** self-updated: a Homebrew or
`go install` binary prints the matching upgrade command (`brew upgrade typeburn`
/ `go install github.com/bavanchun/Typeburn@latest`) and exits without touching
anything. On a non-interactive stream (pipe/redirect) `update` refuses unless
`--yes` is passed, rather than blocking on a prompt.

The minimum usable terminal size is **60 columns × 20 rows**. If the terminal is
too small the app shows a resize prompt and resumes automatically once you resize.

## Keybindings

### Global (every screen)

| Key | Action |
|---|---|
| `ctrl+c` | Quit immediately |
| `esc` | Back / cancel (on Home: shows quit prompt) |
| `ctrl+r` | Restart with fresh test |
| `1` | Go to Home |
| `2` | Go to Settings |
| `3` | Go to History |

### Home / Welcome

| Key | Action |
|---|---|
| `tab` / `shift+tab` | Cycle mode forward / backward (Time → Words → Quote → Code) |
| `←` `→` / `h` `l` | Change length option |
| `enter` / `space` | Start test |

### Typing Test

| Key | Action |
|---|---|
| (any printable) | Type that character |
| `backspace` | Delete last character |
| `tab` | Restart same test |
| `ctrl+r` | New test (re-pick words) |
| `esc` | Abort → Home |

### Result Summary

| Key | Action |
|---|---|
| `tab` / `enter` | Restart same mode and length |
| `ctrl+r` | New test |
| `esc` / `1` | Back to Home |
| `3` | View History |

### Settings

| Key | Action |
|---|---|
| `↑` `↓` / `k` `j` | Move selection |
| `←` `→` / `h` `l` / `enter` | Cycle / toggle selected value |
| `esc` / `1` | Save and back to Home (auto-persists) |

### History

| Key | Action |
|---|---|
| `↑` `↓` / `k` `j` | Scroll rows |
| `g` | Jump to top |
| `G` (shift+g) | Jump to bottom |
| `esc` / `1` | Back to Home |

## Configuration & Data Paths

Settings and history follow the [XDG Base Directory Specification](https://specifications.freedesktop.org/basedir-spec/latest/).

| File | Default path (macOS / Linux) |
|---|---|
| Settings | `~/.config/typeburn/settings.json` |
| History | `~/.local/share/typeburn/history.json` |

Override with `$XDG_CONFIG_HOME` and `$XDG_DATA_HOME` respectively.

## Development

```sh
make test        # go test ./...
make test-race   # go test ./... -race -count=1
make lint        # gofmt -l check + go vet
make fmt         # gofmt -w .
make build       # ./bin/typeburn
make clean       # remove ./bin/
```

CI runs on **ubuntu-latest** and **macos-latest** via GitHub Actions (`.github/workflows/ci.yml`).
Steps: build → vet → gofmt check → test with race detector.

## License

MIT — see [LICENSE](./LICENSE).
