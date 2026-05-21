# Scout Report — Internal Layer Boundaries for Pro CLI v2

**Date:** 2026-05-20
**Purpose:** Map exact extraction surface needed for new `internal/runner` (shared driver used by both TUI and `--no-tui`/`replay`).

## Per-package summary (pure logic, UI-free, reusable as-is)

| Package | Public surface used by future runner | UI deps? |
|---------|--------------------------------------|---------|
| `internal/typing` | `New(target, mode, wordTarget) *Engine`, `Engine.Apply(rune, nowMs)`, `Engine.Backspace(nowMs)`, `Engine.Log()`, `Engine.Progress()`, `Engine.Complete(nowMs)` (in completion.go), `Keystroke` struct | none |
| `internal/metrics` | `Compute(log, mode, endMs) Result`, `Result` struct | none |
| `internal/words` | `NewGenerator(seed)`, `ForMode(g, mode, length, ql) string`, `QuoteLen` | none |
| `internal/codetext` | `Load(path) (string, error)`, `Normalize(string) (string, error)`, `ErrEmpty`/`ErrTooLarge`/`ErrBinary` | none |
| `internal/storage` | `LoadHistory() []Record`, `AppendHistory(Record) ([]Record, error)`, `LoadSettings() Settings`, `SaveSettings(Settings) error`, `Record`, `SettingsPath()`, `HistoryPath()` | none |
| `internal/config` | `Settings`, `Mode`, `LengthsFor(Mode)`, `Defaults()`, `Settings.Normalize()`, `DataDir()`, `ConfigDir()` | binds Bubble Tea key types (config/keymap.go) — irrelevant to CLI |
| `internal/version` | `Resolve() Info`, `Info{Version,Commit,Date}`, `String()` | none |

## Extraction targets for `internal/runner`

**Current duplication risk:** `internal/ui/screen_typing.go:59-82` (`newTypingWithSeed`) is the canonical "build a session from settings" code. It does:
1. `g := words.NewGenerator(seed)`
2. `target := words.ForMode(g, mode, length, ql)`
3. `wordTarget := length; if Time → length * 1000`
4. `eng := typing.New(target, mode, wordTarget)`

Steps 1-4 are pure logic but currently live in the UI package. They must move to `internal/runner` so both the TUI and `--no-tui`/`replay` can build identical sessions.

### Proposed `internal/runner` surface

```go
package runner

type Session struct {
    Engine     *typing.Engine
    Target     string
    Mode       config.Mode
    Length     int
    QuoteLen   words.QuoteLen
    CodeText   string  // empty unless ModeCode
}

// NewSession builds a fresh typing session. Pure: no I/O, no signals.
// seed==0 → random time-based seed; non-zero → deterministic (testing).
func NewSession(mode config.Mode, length int, ql words.QuoteLen, seed int64) Session

// NewCodeSession builds a ModeCode session from a pre-normalized snippet.
func NewCodeSession(snippet string) Session

// Tick returns true if Time mode has elapsed past its limit (for non-TUI loop).
func (s Session) ExpiredAt(startMs, nowMs int64) bool

// Complete checks completion for non-Time modes after each keystroke.
// (Already exists on typing.Engine — kept here as forwarding helper if needed.)
```

Tests for `internal/runner` are deterministic (table-driven, fixed seeds) — same pattern as existing `metrics_test.go`.

## TUI integration (must not regress)

After extraction, `internal/ui/screen_typing.go:NewTyping(...)` becomes a thin shim that calls `runner.NewSession(...)` and wraps the result with UI state (`w`, `h`, `th`, `keys`, `blink`, `headerWPM`, etc.). Same for `NewTypingCode`. Existing `teatest` golden files must continue to pass.

## `internal/app/model.go` Elm flow — unchanged

- `Update()` already routes domain messages (`StartTestMsg`, `ResultMsg`, `AbortMsg`, `NavHistoryMsg`, `NavCodePasteMsg`, `CodePastedMsg`).
- Only `app.New(...)` is touched: `main.go` no longer calls it directly when a subcommand path bypasses the TUI. `app.NewFromDisk` stays untouched.

## What `cli/cmd_run.go` (TUI path) needs

```
flags → config.Settings overrides → app.New(theme, settings, codeText, codeHint) → tea.NewProgram(...).Run()
```

Flag mapping:
- `--mode <time|words|quote|code>` → temporary Settings.DefaultMode override (without persisting)
- `--duration <N>` → length when mode=time
- `--words <N>` → length when mode=words
- `--theme <name>` → temporary theme override; must intersect with `theme.Available()` (currently UI-bound; expose as small theme.Names() function so CLI can validate without importing lipgloss)
- `--text <file>` → reuses existing `codetext.Load`

## What `cli/cmd_history.go` needs

- `storage.LoadHistory() []Record` is enough. Reverse + truncate for `--limit N`.
- JSON output = `json.Marshal(records)` (Record already has json tags).
- Table output = new `internal/cli/output/table.go` writer; columns: when / mode / len / WPM / acc / cons.

## What `cli/cmd_config.go` needs

- Keys = struct field names of `config.Settings`, lower-cased: `theme`, `default_mode`, `default_length`, `blink_cursor`.
- `get <key>` → load, reflect into known list, print value.
- `set <key> <value>` → load, parse value (with type guard per key), call `Settings.Normalize()`, `SaveSettings()`.
- `list` → marshal whole struct (JSON or table).
- Validation reuses `Settings.Normalize()` (already idempotent).

## What `cli/cmd_replay.go` needs

- Input: a JSON file containing `[]typing.Keystroke` + mode + endMs.
- Decode → `metrics.Compute(log, mode, endMs)` → print Result (JSON or table).
- `typing.Keystroke` does NOT have json tags currently. **Action:** add `json:"..."` tags to `internal/typing/engine.go:Keystroke` (back-compatible, no schema change).
- Define a wire-format wrapper in `internal/cli/cmd_replay.go`:

```go
type ReplayInput struct {
    SchemaVersion int               `json:"schema_version"` // 1
    Mode          string            `json:"mode"`           // "time"/"words"/"quote"/"code"
    EndMs         int64             `json:"end_ms"`
    Log           []typing.Keystroke `json:"log"`
}
```

## What `cli/cmd_version.go` needs

- `version.Resolve() Info` already returns the triple.
- Add `Info.MarshalJSON` OR just inline a JSON struct in cmd_version.go — simpler. Include `go_version`, `os`, `arch` to mirror banner.

## What `internal/cli/notui` needs (researcher-02 will fill in)

- `runner.NewSession(...)` → loop: read keystroke → eng.Apply → check Complete → on done call metrics.Compute → print result.
- Raw mode + signal handling: see researcher-02 report.
- Render loop: minimal — clear line + reprint prompt with cursor. No themes initially (keep monochrome for v2; theme rendering is a v2.1 polish).

## Touchpoint summary (files modified by the plan)

**Modify (existing):**
- `main.go` — thin shim to `cli.NewRoot()` + fang.Execute
- `internal/ui/screen_typing.go` — `NewTyping`/`NewTypingCode` delegate to `runner`
- `internal/typing/engine.go` — add json tags to `Keystroke` (back-compat)
- `internal/theme/theme.go` — add `Names() []string` if not present, for CLI validation
- `CLAUDE.md` — revise "no new deps" rule to permit charm-ecosystem + `golang.org/x/*`
- `README.md` — new CLI section
- `Makefile` — add `make size-check`
- `CONTRIBUTING.md` — dep policy revision

**Create (new):**
- `internal/runner/session.go` + `_test.go`
- `internal/cli/{root,cmd_run,cmd_history,cmd_version,cmd_config,cmd_replay,exitcodes}.go` + tests
- `internal/cli/notui/{runner,render}.go` + tests
- `internal/cli/output/{table,json}.go` + tests
- `testdata/sample-keystroke-log.json`
- `go.sum` entries for cobra, fang, x/term

**Delete:** none. (decide() moves into cli/root.go but the test stays as-is — it tested a pure function, the function moved location.)

## Risks the plan must address

1. **`decide()` purity contract** — cobra's default `Execute()` calls `os.Exit`. Pre-build a wrapper that returns errors as values so the `_test.go` for the CLI root can still test parsing without subprocesses.
2. **Bare `typeburn` fall-through** — cobra root cmd needs `Args: cobra.ArbitraryArgs` + `RunE` that launches TUI when called with no recognized subcommand. The exact pattern depends on cobra's argument parsing semantics — researcher-01 will confirm.
3. **`--text` flag ambiguity** — `--text` lives at root (back-compat) AND on `run` subcommand. Best to define it only on the root persistent flags + on `run`'s own flag set, with root's pre-run hook redirecting bare `--text` to `run --text`. Risk of double-binding; needs an integration test.
4. **Theme/storage UI dep leak** — `theme.Available()` currently lives in `internal/theme/theme.go` but the package imports lipgloss. CLI's `cmd_config.go` validation needs the names but must NOT import lipgloss into a CLI handler unless we accept that. Two options: (a) add a tiny `theme/names.go` with just the `[]string` constant (UI-free), or (b) duplicate the name list in CLI (current pattern in `config.Settings.Normalize()` already duplicates it intentionally — accepted with a sync test). **Recommendation:** follow option (b) — add a `theme_available_sync_test.go`-style guard between CLI and theme.

## Unresolved questions for plan phase

- Q1: Do we promote `Keystroke` to have JSON tags now, or wrap it with a CLI-side mirror struct? (Recommend: add tags to source; minor and useful.)
- Q2: Should `run --json` print a result envelope or just the bare `metrics.Result`? (Recommend: envelope with `mode`, `length`, `wpm`, `result_at`, and embedded `result`.)
- Q3: Where does `--no-tui` get its target text? Same `runner.NewSession` path? (Yes — by design.)

---

**Status:** DONE
**Summary:** Mapped exact extraction surface. `internal/runner` is the new shared driver, ~50 LOC of moved logic. All pure-logic packages already reusable without modification (except adding JSON tags to `Keystroke`). UI invariants preserved. CLI surface depends on small additive helpers (`theme.Names()` or sync test).
