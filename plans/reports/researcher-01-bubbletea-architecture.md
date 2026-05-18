# Bubble Tea Architecture Research Report
**Date:** 2026-05-18 | **Target:** High-performance, flicker-free typing app in Go/Charm

---

## 1. Version Stack (Recommended)

**Target v2.0.0+ (stable since Feb 2026)**
- `bubbletea` v2.0.3+ (import: `charm.land/bubbletea/v2`)
- `lipgloss` v2.0.0+ (pure rendering, Bubble Tea manages I/O)
- `bubbles` v2.0.0+ (all components updated for v2 API)

**Migration burden:** HIGH. v1→v2 requires new View() struct (returns tea.View, not string), redesigned KeyMsg/MouseMsg/PasteMsg, cursor positioning via tea.Cursor struct. Official upgrade guide exists; ~2-3 days per medium app.

**Why v2:** Cursed Renderer (ncurses-based) = orders-of-magnitude faster; SSH bandwidth reduced; declarative terminal features eliminate state conflicts; built-for-production (powers Charm's Crush agent).

---

## 2. Architecture: State Machine + Delegation Pattern (RECOMMENDED)

**Pattern:** Root model routes msgs to sub-models (component tree, not enum-based state routing).

**Rationale:** Avoids massive Update() switch; each component owns its logic/state. Idiomatic for Bubble Tea v2.

**Structure:**
```
MainModel {
  state: activeScreen (enum: WelcomeScreen, TypingScreen, ResultsScreen)
  welcome, typing, results: Model (embedded sub-models)
}

Update(msg): routes msg to active component's Update(), batches Cmds
View(): composes View structs from active component's View()
```

Not nested tree delegation (too complex for typing app). Simple enough: 3–5 top-level screens, each wraps bubbles components (input, button, viewport).

---

## 3. Flicker-Free Rendering

**Built-in (v2 default):**
- Cursed Renderer (ncurses algorithm) handles diff rendering automatically
- Mode 2026: synchronized output (atomic writes, no tearing)
- Mode 2027: wide Unicode handling (emojis safe)
- Debounce on resize to prevent flicker cascade

**What NOT needed:** Alternate screen buffer toggling (Cursed Renderer manages it). View() returns struct with `.AltScreen` field if needed; don't issue EnterAltScreen cmd.

**Partial rendering:** Not exposed; Cursed Renderer diff-computes diffs cell-by-cell. No manual cell caching required.

**Cursor:** Use `tea.Cursor{X: x, Y: y, Style: tea.CursorBlock}` in View().Content. Don't style terminal cursor—let framework manage it.

---

## 4. Keyboard Input (v2)

**KeyMsg split:**
- `tea.KeyPressMsg` / `tea.KeyReleaseMsg` (not single KeyMsg)
- Fields: `Code` (rune), `Text` (string), `Mod` (modifiers struct: Ctrl, Shift, Alt, Super)

**Rune handling:** Iterate runes, not bytes. UTF-8 safe—framework decodes multi-byte (Chinese IME multi-rune input supported).

**Special keys:** Backspace, arrows, Enter via `key.Code` constants (e.g., `key.CodeBackspace`).

**Paste:** Dedicated `tea.PasteMsg` with `Data` field. No more `msg.Paste` flag on KeyMsg.

**Ctrl combos:** Check `msg.Mod.Ctrl && msg.Code == key.CodeC`. Space bar returns `key.CodeSpace` (not `" "`).

---

## 5. Timing Pattern for WPM (~10x/sec, 100ms ticks)

**Idiomatic v2 approach:**
```go
func tick() tea.Msg { return TickMsg{Time: time.Now()} }
return tea.Tick(100 * time.Millisecond, tick)
```

**In Update():** Use wall-clock delta (`msg.Time - m.startTime`) NOT tick counts. Avoids drift over long typing sessions.

**Tea.Every():** Alternative for repeated ticks (same sig as Tick). Use Tick for init, Every for repeat in Update.

**Avoid:** time.Ticker goroutines (framework manages concurrency; Tick/Every integrate cleanly).

---

## 6. Terminal Resize Handling

**Pattern:** Catch `tea.WindowSizeMsg` in Update(). Reflow wrapped text using `ansi.Reflow(text, newWidth)` (lipgloss).

```go
case tea.WindowSizeMsg:
  m.width, m.height = msg.Width, msg.Height
  m.content = reflowText(m.content, m.width)
  // Re-init child components with new size
```

**Windows:** No SIGWINCH support; manual polling not required (framework handles). Resizing may lag slightly on Windows.

---

## 7. Project Layout (non-trivial apps)

**Recommended structure:**
```
monkeytype/
├── main.go (entry point, tea.Program)
├── internal/
│   ├── app/
│   │   └── model.go (MainModel, Init/Update/View)
│   ├── ui/
│   │   ├── screen_welcome.go
│   │   ├── screen_typing.go
│   │   ├── screen_results.go
│   │   └── components.go (reusable input, button wrappers)
│   ├── theme/
│   │   └── theme.go (lipgloss styles)
│   ├── metrics/
│   │   └── wpm.go (calculation logic, decoupled from UI)
│   └── config/
│       └── config.go (settings, key bindings)
├── go.mod
└── README.md
```

**Reference:** Charm's examples/ dir (particularly `examples/pager`, `examples/timer`). Rekall CLI structure is also solid (cmd/ + internal/tui/).

---

## 8. Testing

**Teatest library** (charmbracelet/x/exp):
- `teatest.TestModel()` spins a real tea.Program, drives input, captures frames.
- Golden file testing (compare rendered output to baseline).

**Unit testing Update():**
```go
model, cmd := model.Update(tea.KeyPressMsg{Code: key.CodeBackspace})
assert(model.content == expectedContent)
// Check cmd type: tea.NoneCmd or custom Cmd
```

**Pure logic testing:** Decouple WPM calc, validation, state transitions into pure functions (no UI deps). Test those separately.

---

## Dependency Constraints

```
require (
  charm.land/bubbletea/v2 v2.0.3
  charm.land/lipgloss/v2 v2.0.0
  charm.land/bubbles/v2 v2.0.0
)
```

Optional: `charm.land/x` for `teatest` (experimental but stable).

---

## Top 5 Architecture Recommendations

1. **Start with v2.0.3+.** Breaking changes are worth it; Cursed Renderer alone justifies migration. v1 is obsolete for new projects.
2. **Root model → screen enum + component delegation.** Avoids massive Update switch; simpler than nested tree routing.
3. **Use tea.Cursor struct for input focus.** Don't fake terminal cursor positioning with styled text.
4. **Wall-clock deltas for timers.** `time.Now()` in TickMsg, compute elapsed in Update(). Tick counts drift.
5. **Decouple metrics (WPM, accuracy) from rendering.** Pure functions ease testing and later refactoring to backend sync.

---

## Unresolved Questions

- Does monkeytype require multiplayer/spectate mode? (Affects session model lifecycle, state persistence strategy.)
- Exact FPS target? (Cursed Renderer debounces; 60 FPS is safe default. 100 FPS may add no benefit on 100 Hz terminals.)
- IME input testing coverage? (Framework handles Unicode; confirm with manual CJK input on target terminals.)

---

## Sources

- [Bubble Tea v2.0.0 Release](https://github.com/charmbracelet/bubbletea/releases/tag/v2.0.0)
- [Charm v2 Blog](https://charm.land/blog/v2/)
- [Bubble Tea v2 Discussion](https://github.com/charmbracelet/bubbletea/discussions/1374)
- [Managing Nested Models](https://donderom.com/posts/managing-nested-models-with-bubble-tea/)
- [State Machine Pattern](https://zackproser.com/blog/bubbletea-state-machine)
- [Bubble Tea Testing](https://charm.land/blog/teatest/)
- [Tips for Building Bubble Tea](https://leg100.github.io/en/posts/building-bubbletea-programs/)
