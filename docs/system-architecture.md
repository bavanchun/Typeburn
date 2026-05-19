# System Architecture — Bubble Tea v2 Elm Pattern

---

## High-Level Architecture

Root `app.Model` implements Bubble Tea's `tea.Model` interface (Init/Update/View). Global state: screen enum + five sub-models (Home, Typing, Result, Settings, History) + theme + keymap.

**Message flow:** Top-level messages (StartTestMsg, ResultMsg, AbortMsg, NavHistoryMsg) route via `if _,ok := msg.(Type)` in root Update(); screen-specific messages are delegated to sub-models.

**Rendered output:** Each screen's View() is composed into a single terminal frame. Bubble Tea's Cursed Renderer diffs against the previous frame and outputs only changed cells.

---

## Routing & State Machine

### Active Screen Enum

```go
type Screen int
const (
  ScreenHome      Screen = iota
  ScreenTyping
  ScreenResult
  ScreenSettings
  ScreenHistory
  ScreenCodePaste
)
```

Root Model holds the current `screen` value and delegates to the matching sub-model's Update/View.

### Message Types (in `internal/ui/messages.go`)

| Message | Emitter | Handler | Effect |
|---------|---------|---------|--------|
| `StartTestMsg` | Home screen | root app.Model | New TypingModel; switch to ScreenTyping |
| `ResultMsg` | TypingModel | root app.Model | Persist record, detect new-best, switch to ScreenResult |
| `AbortMsg` | TypingModel | root app.Model | Discard test, return to ScreenHome |
| `NavHistoryMsg` | Result/Settings | root app.Model | Load fresh history from disk, switch to ScreenHistory |
| `NavCodePasteMsg` | Home screen (empty Code row) | root app.Model | Open ScreenCodePaste with a fresh paste sub-model |
| `CodePastedMsg` | CodePasteModel | root app.Model | Set codeText, clear codeHint, apply via HomeModel.WithCodeText, return to ScreenHome |
| `tea.KeyPressMsg` | Bubble Tea | active screen | Typed character or control key |
| `tea.PasteMsg` | Bubble Tea | TypingModel / CodePasteModel | Typing: chars logged; ScreenCodePaste: normalized via codetext.Normalize |
| `tea.WindowSizeMsg` | Bubble Tea | root app.Model | Terminal resized; reflow content |
| `tickMsg` | TypingModel | TypingModel | ~100ms periodic update; recalculate live metrics |

### Sub-Model Tree

```
Model (app)
├─ HomeModel (ui)
├─ TypingModel (ui) → owns typing.Engine + metrics.Result
├─ ResultModel (ui)
├─ SettingsModel (ui)
├─ HistoryModel (ui)
└─ quitPromptModel (app) — overlay on Home
```

Each sub-model has `SetSize(w, h)` to handle WindowSizeMsg; root calls it on resize.

---

## Full Test Flow (Home → Typing → Result → History)

1. **Home screen:** User selects mode (Time/Words/Quote) + length; presses Enter
   - Emits `StartTestMsg{Mode, Length, QuoteLen}`
2. **Root receives StartTestMsg:**
   - Creates new `TypingModel` with mode/length/generated target words
   - Switches `screen = ScreenTyping`
   - TypingModel.Init() returns no cmd (tick is armed on first keystroke)
3. **Typing screen:**
   - User types; each keystroke → `tea.KeyPressMsg`
   - TypingModel.Update() → `typing.Engine.Apply(rune)` logs keystroke
   - On first key, TypingModel returns `tickCmd()` to arm 100ms timer loop
   - Every ~100ms, tickMsg → recalculate live WPM/accuracy/consistency from keystroke log
   - Test completes on: (Time mode: timer expires) OR (Words mode: word count reached) OR (Quote mode: all runes typed)
   - Emits `ResultMsg` with final `metrics.Result`
4. **Root receives ResultMsg:**
   - Calls `storage.AppendHistory()` to persist record (atomic write, cap 200)
   - Calls `storage.IsNewBest()` to detect if this is a new personal best for that mode+length
   - Creates ResultModel with record + new-best flag
   - Switches `screen = ScreenResult`
5. **Result screen:**
   - Displays big-digit WPM, charts, stats
   - User can: Tab/Enter to restart same test, Ctrl+R for new test, 3 to view History, Esc to home
6. **History screen:**
   - Scrollable table of all records; reverse-sorted (newest first)
   - ★ badge on per-mode best; grey out records from other modes/lengths

---

## Pure vs. UI Layers

### Pure Packages (Zero Bubble Tea Imports)

- **`internal/typing`:** Engine state machine + keystroke logging. No UI/Bubble Tea.
- **`internal/metrics`:** WPM/accuracy/consistency computation. No UI/Bubble Tea.
- **`internal/words`:** Word generator + quote pack. No UI/Bubble Tea.
- **`internal/config`:** Settings types + keymap. No UI/Bubble Tea.
- **`internal/storage`:** JSON persistence (atomic write, XDG paths). No UI/Bubble Tea.
- **`internal/theme`:** Role-based color mapping. No UI/Bubble Tea.

### UI Package

- **`internal/ui`:** Screen models (Home/Typing/Result/Settings/History) + components (word-stream, stat-card, table, timer).
- **`internal/app`:** Root Elm model + routing logic.

This separation allows metrics to be tested without running the terminal, and reused in future CLI/server projects.

---

## Core Data Structures

### typing.Keystroke Log

```go
type Keystroke struct {
  TimeMs  int64 // relative to test start
  Typed   rune  // what user typed (0 for backspace)
  Target  rune  // expected char at that position
  Correct bool  // typed == target
}
```

All computation (WPM, accuracy, consistency) derives from this immutable log.

### metrics.Result

```go
type Result struct {
  NetWPM       float64
  RawWPM       float64
  Accuracy     float64
  Consistency  float64
  CPS          float64
  TimeMs       int64
  CharCount    int
  ErrorCount   int
  ErrorHistory []int         // per-second errors
  WPMHistory   []float64     // per-second raw WPM
  // ... other fields
}
```

Computed from keystroke log post-hoc (no live state). Passed to ResultMsg.

### storage.Record

```go
type Record struct {
  WPM       int           // rounded for display
  NetWPM    float64       // v2: add this for precise new-best comparison
  RawWPM    float64
  Accuracy  float64
  Consistency float64
  Mode      config.Mode
  Length    int
  Time      time.Time
  // ... other fields
}
```

Persisted to history.json; compared in IsNewBest() logic.

### config.Settings

```go
type Settings struct {
  Theme         string        // "default" | "mono"
  DefaultMode   config.Mode   // "time" | "words" | "quote"
  DefaultLength int           // 15, 30, 60, 120 (time) or 10, 25, 50, 100 (words)
  BlinkCursor   bool
}
```

Loaded from XDG_CONFIG_HOME at startup; auto-persisted on settings change.

---

## Timer & Live Metrics

**100ms tick loop** (not tick-count; wall-clock deltas):

```go
func tickCmd() tea.Cmd {
  return tea.Tick(100*time.Millisecond, func() tea.Msg {
    return tickMsg{time.Now()}
  })
}
```

In TypingModel.Update(tickMsg):
- Calculate elapsed = now − startMs
- Replay keystroke log to final state
- Compute metrics.Compute(log, startMs, elapsed)
- Render live WPM in header
- Re-arm tick (returned from handleTick)

**AFK (Away From Keyboard) trimming:** Time mode only. If last keystroke >7s ago, trim trailing empty seconds from metric buckets.

---

## Theme Role System

**Never hardcode color.** Screen code references semantic roles:

```go
type Role string
const (
  RoleTextPrimary
  RoleTextMuted
  RoleAccent
  RoleError
  RoleCursor
  // ... 12 roles total
)
```

Theme maps each role to a color (or attribute-only under NO_COLOR):

```go
type Theme struct {
  name    string
  colors  map[Role]color.Color
  noColor bool // true if NO_COLOR env set or mono theme chosen
}

func (t Theme) Style(r Role) lipgloss.Style {
  // Returns lipgloss.Style with color + attributes
}
```

### NO_COLOR Behavior

If `NO_COLOR=1` set or theme="mono", the theme switches to attribute-only:
- Accent → bold
- Error → underline
- Muted → normal + slightly dimmer attribute
- Layout unchanged (no color, same cells)

---

## Degraded Mode (Small Terminal)

**Single chokepoint:** `app/model_view.go:20` checks terminal size **before any screen delegation**.

```go
if w < 60 || h < 20 {
  // Render centered notice only; no partial paint
  return centerNotice("Terminal too small...")
}
// Else delegate to active screen's View()
```

Ensures no screen attempts to render below the minimum; user sees a resize prompt and app resumes automatically.

---

## Storage & Persistence

### Atomic File Write Pattern (`internal/storage/atomic_write.go`)

1. Create temp file in same directory
2. Write JSON + fsync
3. Close file
4. Atomic rename temp → target
5. (Known limitation: parent directory not fsync'd; acceptable for local user data)

### XDG Paths

```go
// Settings
~/.config/typeburn/settings.json  [or $XDG_CONFIG_HOME]

// History
~/.local/share/typeburn/history.json  [or $XDG_DATA_HOME]
```

### Error Handling

- Missing files → safe defaults (empty history, default settings)
- Corrupt JSON → returns nil slice + logs nothing (never panic)
- Failed writes → in-memory state unaffected, caller informed via error

---

## Package Dependency Graph

```
main.go
  └─ app.Model
      ├─ ui (all screens)
      ├─ theme
      ├─ config (keymap, settings)
      └─ storage (load/save)

ui (screens)
  ├─ typing (Engine)
  ├─ metrics (Compute)
  ├─ words (generator)
  ├─ config
  ├─ theme
  └─ storage (history table)

metrics
  └─ config (Mode)

typing
  └─ config (Mode)

words
  └─ config (Mode, QuoteLen)

storage
  ├─ config (Settings, XDG paths)
  └─ metrics (Result for serialization)
```

No circular imports; pure packages have zero UI dependencies.

---

## Lifecycle & Testing

### Init

1. `main.go` calls `app.NewFromDisk()`
2. Loads settings from XDG_CONFIG (or defaults if missing)
3. Loads theme based on settings.Theme
4. Creates root Model with Home screen active
5. `tea.NewProgram(m)` starts event loop

### Shutdown

- Ctrl+C hard-quits everywhere
- Esc on Home shows quit prompt ("quit? y/n", default "n")
- Settings auto-persisted on change (no manual save)
- History auto-persisted on test completion

### Testing

- Pure functions (metrics, word gen) tested via `go test` with table-driven tests
- UI screens tested via `teatest.TestModel()` (drives keyboard input, captures rendered frames)
- Golden files in `internal/ui/*_test.go` verify rendered output matches baseline
- `-race` flag verifies no goroutine leaks or data races

---

## Release Engineering & Versioning

### Version Injection

The `internal/version` package supports two injection paths:

1. **GoReleaser (release builds):** Makefile + `.goreleaser.yaml` inject Version/Commit/Date via ldflags (`-X` flags)
   - Version: `v{MAJOR}.{MINOR}.{PATCH}` (v-prefixed to match `go install` semantics)
   - Commit: short SHA
   - Date: UTC RFC3339 timestamp

2. **Fallback (bare `go install`):** When ldflags are not set, `version.Resolve()` reads `debug.ReadBuildInfo()`:
   - Version: from `go.mod` version (ignores synthetic "(devel)")
   - Commit/Date: from vcs.revision and vcs.time build settings
   - Ultimate fallback: version = "dev"

**Entry point:** `--version` flag (parsed in `main.go` via pure `decide()` function) prints a one-line banner:
```
typeburn v1.0.0 (61a4afd, 2026-05-18T21:10:00Z, go1.26.2 darwin/arm64)
```

### Build Pipeline

**Local builds:**
```bash
make build         # ldflags-stamped binary with git metadata
make test-race     # verify no races before release
make snapshot      # GoReleaser dry-run (builds + archives, no publish)
```

**Release (CI-only):**
- Triggered by git tag matching `v*` (e.g., `v1.0.0`)
- `.github/workflows/release.yml`: 
  - Separate least-privilege `test` job (contents:read) runs before publish
  - `publish` job (contents:write) runs GoReleaser v2.15.4 (SHA-pinned) with `--release-notes=.github/release-notes.md`
  - Post-publish: asserts exactly 7 assets (2 tar.gz + 2 zip + 2 tar.gz arm64 + checksums.txt)

### Artifact Distribution

**Cross-platform binaries:** 6 combinations
- Linux (amd64, arm64): `.tar.gz` archives include README, LICENSE, CHANGELOG
- macOS (amd64, arm64): `.tar.gz` archives
- Windows (amd64, arm64): `.zip` archives

**Integrity:** SHA-256 checksums in `checksums.txt` (GoReleaser native); verify with `sha256sum -c checksums.txt`

**Installation paths:**
- `go install github.com/bavanchun/Typeburn@v1.0.0` (from GitHub via Go module)
- Download pre-built binary from [Releases](https://github.com/bavanchun/Typeburn/releases)

### Release Process (Fix-Forward Policy)

Tags are immutable once pushed; releases are never re-tagged or reverted. **Fix-forward policy:**
1. Identify the bug in v1.0.0
2. Fix on main branch
3. Tag as v1.0.1 (next patch version)
4. Push tag → CI automatically publishes new release with updated CHANGELOG

**Before real tag:** Dry-run with `make snapshot` to verify build + archive paths locally (no publish/auth).
