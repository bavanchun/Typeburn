# CLI Reference

Typeburn v2 adds a cobra/fang command surface while preserving the v1 shortcuts:
bare `typeburn` opens the TUI, `typeburn --version` prints the banner, and
`typeburn --text <file>` starts Code mode from a file.

## Commands

| Command | Purpose |
|---|---|
| `typeburn` | Open the interactive TUI Home screen |
| `typeburn run` | Start a test directly, optionally in raw `--no-tui` mode |
| `typeburn history` | Print saved history as a table or JSON |
| `typeburn config` | Read or update persisted settings |
| `typeburn version` | Print version info as text or JSON |
| `typeburn replay` | Compute metrics from a schema-versioned keystroke log |

## Exit Codes

| Code | Meaning |
|---|---|
| 0 | success |
| 1 | usage or validation error |
| 2 | I/O or replay-file error |
| 3 | user abort in raw `--no-tui` mode |
| 4 | reserved for internal errors |

## Run

```sh
typeburn run --mode time --duration 30 --theme nord
typeburn run --mode words --words 25
typeburn run --mode quote --quote-len short
typeburn run --mode code --text snippet.go
typeburn run --no-tui --mode words --words 10
```

Flags:

| Flag | Modes | Notes |
|---|---|---|
| `--mode time|words|quote|code` | all | Defaults to persisted default mode |
| `--duration N` | time | Positive seconds |
| `--words N` | words | Positive word count |
| `--quote-len short|medium|long` | quote | Defaults to medium |
| `--theme NAME` | TUI | Initial-only override; Settings changes inside the TUI win |
| `--text PATH|-` | code | Required for CLI Code mode |
| `--no-tui` | all | Raw stdin/stdout runner; stdin must be a terminal |
| `--json` | `--no-tui` only | Emits final metrics JSON |

Code mode from the CLI never opens the in-app paste screen. Use bare `typeburn`
and choose Code to paste interactively.

Raw mode limitation: if a `--no-tui` process is killed by SIGKILL or the parent
terminal disappears, Typeburn cannot restore the terminal. Run `reset` or
`stty sane` to recover.

## History

```sh
typeburn history
typeburn history -n 5 --json
```

Table mode prints newest records first. Empty history prints `no history yet`;
JSON mode prints `[]`.

## Config

```sh
typeburn config list
typeburn config list --json
typeburn config get theme
typeburn config set theme nord
typeburn config set update_check on
```

Keys: `theme`, `default_mode`, `default_length`, `blink_cursor`, `update_check`.

| Key | Values | Default | Notes |
|---|---|---|---|
| `theme` | name string | `default` | Built-in theme name |
| `default_mode` | `time\|words\|quote\|code` | `time` | |
| `default_length` | positive int | `30` | Seconds or word count |
| `blink_cursor` | `true\|false` | `false` | |
| `update_check` | `on\|off` (also `true\|false\|yes\|no\|1\|0`) | `off` | Opt-in opportunistic check |

`config set` is strict: invalid values exit 1 before writing `settings.json`.

### update_check

When `update_check` is `on`, each TUI launch fires an opportunistic GitHub
release check with an 800 ms timeout. On success the result is cached for 24 h
at `$XDG_STATE_HOME/typeburn/update-check.json` (default
`~/.local/state/typeburn/update-check.json`). If a newer stable release exists,
the Result screen shows a muted footer hint.

The check is **always opt-in**. It is never triggered by `--no-tui` runs.

## Version

```sh
typeburn version
typeburn version --json
typeburn version --check-update
typeburn version --check-update --json
```

`--check-update` always hits the network (ignores cache). Output:

Human-readable (no upgrade):
```
typeburn v2.0.0 (abc1234, 2026-05-20)  ✓ up to date
```

Human-readable (upgrade available):
```
typeburn v2.0.0 (abc1234, 2026-05-20)

A newer version is available: v2.1.0

Upgrade:
  brew upgrade typeburn
  curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh
  go install github.com/bavanchun/Typeburn@latest
```

JSON schema (`--check-update --json`):
```json
{
  "version": {"version": "v2.0.0", "commit": "abc1234", "date": "2026-05-20T00:00:00Z"},
  "update_check": {
    "schema_version": 1,
    "current": "v2.0.0",
    "latest": "v2.1.0",
    "upgrade_available": true,
    "release_url": "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
    "checked_at": "2026-05-21T10:00:00Z"
  }
}
```

## Replay

```sh
typeburn replay testdata/sample-keystroke-log.json
typeburn replay testdata/sample-keystroke-log.json --json
```

Input schema:

```json
{
  "schema_version": 1,
  "mode": "words",
  "end_ms": 5000,
  "log": [
    {"time_ms": 0, "typed": 104, "target": 104, "correct": true}
  ]
}
```

Unsupported schema versions, malformed JSON, missing files, empty logs, and
invalid modes exit 2.
