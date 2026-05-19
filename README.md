# Typeburn

[![CI](https://github.com/bavanchun/Typeburn/actions/workflows/ci.yml/badge.svg)](https://github.com/bavanchun/Typeburn/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/bavanchun/Typeburn?sort=semver)](https://github.com/bavanchun/Typeburn/releases/latest)
[![Go](https://img.shields.io/github/go-mod/go-version/bavanchun/Typeburn)](https://go.dev/)
[![Go Reference](https://pkg.go.dev/badge/github.com/bavanchun/Typeburn.svg)](https://pkg.go.dev/github.com/bavanchun/Typeburn)
[![License](https://img.shields.io/github/license/bavanchun/Typeburn)](./LICENSE)

A Monkeytype-style terminal typing test built with Go and Bubble Tea v2.
Distraction-free, keyboard-driven, and works on any ANSI terminal.

## Features

- **Four test modes**: Time (15/30/60/120 s), Words (10/25/50/100 words), Quote (short/medium/long/epic), Code (your own text via `--text`)
- **Live stats**: WPM, raw WPM, accuracy, and consistency updated every keystroke
- **Result screen**: big-digit WPM, sparkline chart, full char breakdown
- **History**: scrollable table of all past tests with per-mode best marker (★)
- **Themes**: `default` (dark, green accent) and `mono` (attribute-only, no color codes)
- **NO_COLOR support**: set `NO_COLOR=1` for a fully attribute-only render (bold/underline/faint only)
- **Minimum terminal**: 60 columns × 20 rows; graceful degraded notice below that
- **XDG-compliant paths**: settings and history go to `$XDG_CONFIG_HOME` / `$XDG_DATA_HOME`

## Installation

Requires **Go 1.26+**.

**1. `go install` (latest tagged release):**

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

**2. Download a pre-built binary:**

Grab the archive for your OS/arch from the
[latest release](https://github.com/bavanchun/Typeburn/releases/latest)
(linux/darwin/windows × amd64/arm64), verify it against `checksums.txt`,
extract, and run the `Typeburn` binary.

**3. Build from source:**

```sh
make build            # → ./bin/typeburn (lowercase, local convention)
# or
go build -o typeburn .
make run               # run without installing (or: go run .)
```

**4. Homebrew:** planned, not yet available (see [CONTRIBUTING.md](./CONTRIBUTING.md)).

## Usage

```sh
./bin/typeburn            # from `make build`
Typeburn                  # from `go install` / release archive
Typeburn --version        # print version, commit, build date, toolchain; then exit
Typeburn --text snippet.go # Code mode: type your own file
cat snippet.go | Typeburn --text -   # Code mode: read the snippet from stdin
```

In **Code mode** you type the supplied text exactly — every space, tab, and
line break — and the test finishes on an exact match. Without `--text`, the
Code tab is shown but disabled (in-app paste is planned). Code runs appear in
History but never set a ★ personal best.

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
