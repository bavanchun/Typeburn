# monkeytype-tui

A Monkeytype-style terminal typing test built with Go and Bubble Tea v2.
Distraction-free, keyboard-driven, and works on any ANSI terminal.

## Features

- **Three test modes**: Time (15/30/60/120 s), Words (10/25/50/100 words), Quote (short/medium/long/epic)
- **Live stats**: WPM, raw WPM, accuracy, and consistency updated every keystroke
- **Result screen**: big-digit WPM, sparkline chart, full char breakdown
- **History**: scrollable table of all past tests with per-mode best marker (★)
- **Themes**: `default` (dark, green accent) and `mono` (attribute-only, no color codes)
- **NO_COLOR support**: set `NO_COLOR=1` for a fully attribute-only render (bold/underline/faint only)
- **Minimum terminal**: 60 columns × 20 rows; graceful degraded notice below that
- **XDG-compliant paths**: settings and history go to `$XDG_CONFIG_HOME` / `$XDG_DATA_HOME`

## Install / Build

Requires **Go 1.26+**.

```sh
# Build binary into ./bin/
make build

# Or directly with go
go build -o monkeytype-tui .

# Run without installing
make run
# or
go run .
```

## Usage

```sh
./bin/monkeytype-tui
```

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
| `tab` / `shift+tab` | Cycle mode forward / backward (Time → Words → Quote) |
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
| Settings | `~/.config/monkeytype-tui/settings.json` |
| History | `~/.local/share/monkeytype-tui/history.json` |

Override with `$XDG_CONFIG_HOME` and `$XDG_DATA_HOME` respectively.

## Development

```sh
make test        # go test ./...
make test-race   # go test ./... -race -count=1
make lint        # gofmt -l check + go vet
make fmt         # gofmt -w .
make build       # ./bin/monkeytype-tui
make clean       # remove ./bin/
```

CI runs on **ubuntu-latest** and **macos-latest** via GitHub Actions (`.github/workflows/ci.yml`).
Steps: build → vet → gofmt check → test with race detector.

## License

MIT — see LICENSE (placeholder; add your LICENSE file).
