# Code Standards & Conventions

---

## File Organization

### Size & Modularization

- **Target:** <200 lines per file (enforced during code review)
- **Rationale:** Easier navigation, reduced context load, better testability
- **Exception:** Generated data (word lists, quote packs) may exceed slightly
- **Practice:** Large files split into logical modules: screen.go (struct + Init/Update), screen_view.go (View rendering), screen_actions.go (handlers), screen_test.go (tests)

### Naming Convention

**Go file names:** Snake_case with hyphens for semantic/output modules (NOT kebab-case for Go source, but hyphens used for utility/non-core files).

| Pattern | Example | Purpose |
|---------|---------|---------|
| Core logic | `engine.go`, `typing.go` | Main types/functions |
| Screen struct | `screen_home.go` | Model definition |
| View rendering | `screen_home_view.go` | View() implementation |
| Handlers/actions | `screen_home_actions.go`, `screen_typing_actions.go` | Helper functions |
| Tests | `*_test.go` | Unit/integration tests |
| Utility/output | `ascii-logo.go`, `degraded-notice.go` | Non-core, reusable |
| Config/standards | `xdg-paths.go`, `default-theme.go` | Semantic names |

**Rationale:** Go ecosystem is flexible; files <200 LOC are the key constraint. Snake_case for implementation, hyphens for utilities (matches examples/style guides seen in Charm projects).

### Package Structure

```
monkeytype/
├── cmd/typeburn/main.go       # entrypoint
├── go.mod / go.sum           # dependencies
├── internal/
│   ├── app/                  # root Elm model + routing
│   ├── ui/                   # screen sub-models + components
│   ├── mode/                 # shared mode enum + length policy
│   ├── typing/               # pure keystroke engine
│   ├── metrics/              # pure metric formulas
│   ├── words/                # word/quote generator
│   ├── config/               # settings + centralized keybindings
│   ├── storage/              # persistence + new-best detection
│   └── theme/                # color/attribute roles
└── docs/                     # documentation
```

---

## Code Quality Standards

### Compilation & Tooling

- **go build:** Must compile without errors (`go build -o bin/typeburn ./cmd/typeburn`)
- **go vet:** No warnings (`go vet ./...`)
- **gofmt:** Automatic formatting enforced (`gofmt -w .` before commit; CI checks with `gofmt -l`)
- **go test:** All tests PASS (`go test ./... -race -count=1`)

### Doc Comments

- **Exported types/functions:** Must have doc comment (enforced by golint implicit standards)
- **Format:** `// TypeName brief description.` or `// FunctionName brief description.`
- **Example:**
  ```go
  // Engine maintains the mutable typing state: target buffer, typed buffer,
  // keystroke log, and mode metadata.
  type Engine struct { ... }
  
  // Apply records a printable rune keystroke at the given timestamp.
  func (e *Engine) Apply(r rune, nowMs int64) { ... }
  ```
- **Unexported helpers:** Brief inline comments for non-obvious logic

### Error Handling

- **Try-Catch principle:** No panics in production code (only in unit tests for assertions)
- **Pattern:** Explicit error returns + handle gracefully:
  ```go
  // Good: load file, fallback to defaults on error
  func LoadSettings() Settings {
    data, err := os.ReadFile(path)
    if err != nil {
      return Defaults()  // never panic, never return nil
    }
    var s Settings
    if err := json.Unmarshal(data, &s); err != nil {
      return Defaults()
    }
    return s
  }
  
  // Bad: panic on missing file
  func LoadSettings() Settings {
    data := os.ReadFile(path)  // will crash
    ...
  }
  ```
- **Persistence special case:** Corrupt JSON/missing files → empty slice + defaults, **never panic**

### Type Design

- **Exports:** Capitalize (Engine, Result, Settings)
- **Unexported:** lowercase (startMs, typed, log)
- **Interface compliance:** Types implicitly satisfy interfaces (no `var _ Interface = (*Type)(nil)` needed unless clarifying)
- **Pointer receivers:** Use for methods that mutate state (Engine.Apply, Engine.Backspace)
- **Value receivers:** Use for immutable queries or small data (Theme.Style, Record fields)

### Testing

- **Table-driven tests:** Standard for unit tests
  ```go
  var tests = []struct {
    name   string
    input  string
    want   int
  }{
    {"empty", "", 0},
    {"one word", "hello", 1},
  }
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      if got := Count(tt.input); got != tt.want {
        t.Errorf("Count(%q) = %d, want %d", tt.input, got, tt.want)
      }
    })
  }
  ```
- **No mocks:** Use real instances (no mock Engines, mock storage). Tests drive actual code paths.
- **Golden files:** teatest captures rendered output; baseline stored in `_test.go` or separate file
- **Race detection:** All tests must pass `-race` flag (`-count=1` to avoid flakiness)

### Constants & Magic Numbers

- **Named constants:** For configuration, thresholds
  ```go
  const (
    historyCapMax    = 200
    afkThresholdMs   = 7000  // 7 seconds
    tickIntervalMs   = 100
  )
  ```
- **Magic number in metrics:** Document with comment
  ```go
  // kogasa = 100 * tanh(1 - cv); clamped [0, 100]
  consistency := 100 * math.Tanh(1 - cv)
  ```

### Security & Correctness

- **File permissions:** 0600 (user-readable/writeable only) for settings/history JSON
- **Directory permissions:** 0700 for XDG config/data dirs
- **Atomic writes:** Always use temp+fsync+rename for persistence (no partial writes)
- **Rune safety:** Iterate over `[]rune(string)` NOT `string` or `[]byte` (multi-byte Unicode safe)
- **No hardcoded colors:** Always use theme.Style(role) — never inline hex/ANSI codes

---

## Bubble Tea & Lip Gloss Idioms

### State Management (Elm Pattern)

```go
// Model: immutable update return value
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
  switch msg := msg.(type) {
  case tea.KeyPressMsg:
    m.field = newValue  // mutate copy
    return m, someCmd()  // return updated copy + cmd
  }
  return m, nil
}

// View: pure function
func (m Model) View() tea.View {
  return tea.View{
    Content: m.renderContent(),
  }
}
```

### Messages

- **Global messages:** Emitted by sub-models, handled by root (StartTestMsg, ResultMsg, AbortMsg, NavHistoryMsg)
- **Screen-internal messages:** Handled within the sub-model's Update() (e.g., selection changes in Settings)
- **Custom message types:** Define in `internal/ui/messages.go` or within the screen package

### Theme Integration

```go
// Always use theme.Style(role); never hardcode
button := lipgloss.NewStyle().
  Foreground(t.Color(RoleAccent)).  // may be nil under NO_COLOR
  SetString("Press Enter")

// Safe rendering under NO_COLOR
if t.Color(RoleAccent) == nil {
  // NO_COLOR active; use attributes only
  button = button.Bold(true)
}
```

### Render Pattern

```go
func (m HomeModel) View() tea.View {
  if m.w < 60 || m.h < 20 {
    // Degraded mode; return early
    return tea.View{Content: degradedNotice(m.w, m.h)}
  }
  
  // Build layout
  content := lipgloss.JoinVertical(...)
  
  // Center if room
  if m.w > 80 {
    content = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, content)
  }
  
  return tea.View{Content: content}
}
```

---

## Metrics & Typing Logic

### Pure Computation Guarantee

- **No side effects:** metrics.Compute, typing.Engine.Replay, consistency calculations
- **Layering:** typing/metrics/words/runner use `internal/mode`, not `internal/config`
- **Testable standalone:** No dependency on Bubble Tea, storage, or UI
- **Reusable:** Metrics package suitable for future CLI/server projects

### Keystroke Log Format

```go
// TimeMs: relative to test start (0-based)
// Typed: the rune user typed (0 for backspace marker)
// Target: expected rune at that position
// Correct: typed == target
type Keystroke struct {
  TimeMs  int64
  Typed   rune
  Target  rune
  Correct bool
}
```

### Metric Formulas (Verified)

| Metric | Formula | Notes |
|--------|---------|-------|
| Net WPM | (correct chars / 5) * 60 / seconds | Penalizes uncorrected errors |
| Raw WPM | (all keystrokes / 5) * 60 / seconds | Ignores accuracy |
| Accuracy | 100 * correct / (correct + incorrect) | Final state only; corrections don't penalize |
| Consistency | 100 * tanh(1 - CV) | CV = stdDev / mean of per-second raw WPM |
| CPS | total chars / seconds | Equivalent to raw WPM * 5 / 60 |

### AFK Trimming (Time Mode Only)

```go
// Remove trailing seconds with no keystrokes (>7s gap)
// Quote/Words modes: never trim (explicit test boundary)
if mode == ModeTime && lastKeystrokeGap > 7000 {
  // Trim trailing empty buckets
  trim AFK seconds from history
}
```

---

## Persistence

### Settings JSON

```json
{
  "theme": "default",
  "default_mode": "time",
  "default_length": 30,
  "blink_cursor": false
}
```

- **Saved to:** XDG_CONFIG_HOME/typeburn/settings.json
- **On load:** Normalize() repairs out-of-range values
- **Auto-persist:** SettingsModel emits SettingsChangedMsg on each row change; root model's applySettings() persists + rebuilds theme + re-injects into sub-models

### History JSON

```json
[
  {
    "wpm": 75,
    "raw_wpm": 78.5,
    "accuracy": 95.2,
    "consistency": 78.0,
    "mode": "time",
    "length": 30,
    "time": "2026-05-18T14:35:22Z",
    "char_count": 250,
    "error_count": 12
  }
]
```

- **Saved to:** XDG_DATA_HOME/typeburn/history.json
- **Capped:** Keep newest 200 records (rotate oldest)
- **Append-only:** LoadHistory → append → sort → cap → atomic write
- **Error handling:** Missing/corrupt → empty slice; never panic

---

## Testing Patterns

### Unit Test Template

```go
func TestFunctionName(t *testing.T) {
  tests := []struct {
    name    string
    input   string
    want    int
    wantErr bool
  }{
    {"case 1", "input1", 10, false},
    {"case 2", "input2", 0, true},
  }
  for _, tt := range tests {
    t.Run(tt.name, func(t *testing.T) {
      got, err := FunctionName(tt.input)
      if (err != nil) != tt.wantErr {
        t.Errorf("FunctionName(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
      }
      if got != tt.want {
        t.Errorf("FunctionName(%q) = %d, want %d", tt.input, got, tt.want)
      }
    })
  }
}
```

### Integration Test (teatest)

```go
func TestScreenHome_SelectMode(t *testing.T) {
  m := NewHome(defaultSettings, defaultTheme, defaultKeymap)
  m = m.SetSize(80, 24)
  
  // Simulate keystroke
  m, _ = m.Update(tea.KeyPressMsg{Code: key.CodeTab})
  
  // Verify state
  if m.selectedMode != ModeWords {
    t.Errorf("SelectMode: expected Words, got %v", m.selectedMode)
  }
}
```

---

## Commit & CI Standards

### Commit Messages

- **Format:** Conventional commits (feat:, fix:, docs:, test:, refactor:, chore:)
- **Example:** `feat: add sparkline chart to result screen`
- **No AI references:** Don't mention "Claude," "AI," or plan numbers
- **Focused:** One logical change per commit

### CI/CD Gates

- **Build:** `go build ./...` — must succeed
- **Vet:** `go vet ./...` — must be clean
- **Gofmt:** Check for unformatted files — must all be formatted
- **Tests:** `go test ./... -race -count=1` — must PASS
- **Platforms:** ubuntu-latest + macos-latest

---

## Known Deviations & Future Improvements

1. **File naming:** Plan originally specified "kebab-case" but Go ecosystem uses snake_case. Actual codebase mixes appropriately (core logic = snake_case, utilities = hyphens). Document this.
2. **M1 (timer re-arm):** restartSame() should return tickCmd() on restart in Time mode. One-line fix.
3. **M2 (new-best precision):** Add NetWPM float64 to storage.Record; compare that instead of rounded int WPM. Fast-follow.
4. **m5 (CJK width):** Current code assumes all runes width=1. If CJK quotes added, use lipgloss.Width or go-runewidth.

---

## Summary

- **Compilation first:** All code must compile, vet clean, gofmt pass, tests green (-race)
- **File size:** <200 lines per file; split logically
- **No panics:** Graceful error handling, safe defaults on I/O failures
- **Rune safety:** Always iterate over []rune for Unicode-correct handling
- **Pure logic:** metrics, typing, words packages have zero UI imports
- **Theme system:** All color references via role-based theme.Style()
- **Test coverage:** Table-driven unit tests + teatest golden files + race detection
- **Persistence:** Atomic writes (temp+fsync+rename) + XDG paths + corrupt-file safety
