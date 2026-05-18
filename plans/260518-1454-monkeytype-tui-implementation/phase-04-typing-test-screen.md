---
phase: 4
title: Typing test screen
status: completed
priority: P1
effort: ~7h
dependencies:
  - 2
  - 3
---

# Phase 4: Typing test screen

## Overview

The core interactive screen: render the word-stream with per-char theme styling + block cursor, a mode-aware minimal header, and a footer; wire keystrokes into the Phase 2 engine and a `tea.Tick` wall-clock timer; on completion emit a result message. This is the first screen a user actually types in.

Refs: researcher-01 §3,4,5,6; design §4 (layout), §5.1/§5.2 (cursor/char states), §5.4 (footer), §8.3 (keys); mockups §2.

## Requirements

### Functional
- Word-stream: per-char styled via theme roles (untyped/correct/incorrect/incorrect-space/extra/current), hard-wrap at content width (`min(termW-8,80)`, narrower tiers per §4.1).
- Block cursor on current char (`cursor-bg`/`cursor-fg`); steady (blink wired in Phase 7).
- Header (left, `text-muted`, no border): Time → `WPM   elapsed / total`; Words → `WPM   done / total`; Quote → `WPM   pct% ▰▰▰▱▱`.
- Footer (§5.4): `tab restart · ctrl+r new · esc menu`.
- Timer via `tea.Tick`: ~100ms metric sample, header WPM repaints ~250ms; wall-clock delta `now - startMs`, never tick-count. Clock starts on first keystroke.
- Completion (mode-aware: Time timeout / Words count / Quote full) → emit `ResultMsg{Result}` (consumed by Phase 6; until then route to placeholder Result/Home).
- Keys: any printable → `engine.Apply`; `backspace` → `engine.Backspace`; `tab` restart same; `ctrl+r` new (re-pick words/quote); `esc` → Home.

### Non-functional
- Rune-safe rendering (build styled cells per rune). No flicker (trust v2 Cursed Renderer; no manual alt-screen).
- Re-layout on `tea.WindowSizeMsg`. Files <200 lines.

## Architecture

Data flow: `KeyPressMsg` → `engine.Apply/Backspace` → state recompute. `TickMsg` → sample per-second (engine already logs), recompute header WPM from `metrics` partial. `WindowSizeMsg` → recompute content width, re-wrap. Completion → `func() tea.Msg { return ResultMsg{metrics.Compute(...)} }`.

```go
// internal/ui
type TypingModel struct {
    eng        *typing.Engine
    mode       typing.Mode
    length     int
    target     string
    w, h       int
    startMs    int64
    nowMs      int64
    lastPaint  int64   // header throttle
    headerWPM  float64
    blink      bool    // from settings (Phase 7); steady in v1 default
}
func NewTyping(mode typing.Mode, length int, ql words.QuoteLen, th theme.Theme, km config.Keymap) TypingModel
func (m TypingModel) Update(msg tea.Msg) (TypingModel, tea.Cmd)
func (m TypingModel) View() string

type ResultMsg struct{ Result metrics.Result; Mode typing.Mode; Length int }
type tickMsg struct{ t time.Time }
func tickCmd() tea.Cmd // tea.Tick(100ms, …)

// reusable components
func RenderWordStream(states []typing.CharState, target []rune, typed []rune,
                      width int, th theme.Theme) string  // word-stream-renderer.go
func RenderFooter(hints []Hint, width int, th theme.Theme) string // footer.go
func ModeHeader(mode typing.Mode, wpm float64, done,total int, pct float64, th theme.Theme) string
```

Layout: `lipgloss.JoinVertical(Left, header, "", stream, spacer, footer)` then `lipgloss.Place(w,h,Center,Center,…)`. Degraded (<60×20) deferred to Phase 9 but add a guard hook now (render notice stub) so it never partial-paints.

## Related Code Files

Create:
- `internal/ui/screen-typing.go`
- `internal/ui/word-stream-renderer.go`
- `internal/ui/footer.go`
- `internal/ui/mode-header.go`
- `internal/ui/timer.go` (tickCmd, wall-clock helpers)
- `internal/ui/screen-typing_test.go` (basic Update smoke; full teatest in Phase 10)

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (add TypingModel, route start→typing, handle ResultMsg → placeholder)

Delete: none.

## Implementation Steps

1. `word-stream-renderer.go`: map each `CharState` → `theme.Style(role)` per §5.2 table; build per-rune string; wrap at content width via lipgloss; `extra` runes appended after target; Current cell = cursor style. Rune iteration only.
2. `mode-header.go`: format per mode (elapsed mm:ss for Time, done/total for Words, pct + `▰/▱` bar for Quote).
3. `footer.go`: render `key action · …` per §5.4 (full form; narrow collapse in Phase 9).
4. `timer.go`: `tickCmd()` = `tea.Tick(100*ms, …)`; helper `elapsedMs(now, start)`.
5. `screen-typing.go`: `Update` handles KeyPress (printable/backspace/tab/ctrl+r/esc per §8.3), tickMsg (re-arm tick; if `now-lastPaint≥250ms` recompute headerWPM via `metrics.Compute` on current log; check completion), WindowSizeMsg (store w/h). On completion return `ResultMsg` cmd. `View` composes layout, centers.
6. Wire into `internal/app`: Home "start" (placeholder trigger ok until Phase 5) → `NewTyping`; on `ResultMsg` route to placeholder Result then Home (real Result = Phase 6).
7. `go build ./...`, run: type a Words-10 test end-to-end, confirm states/cursor/header/footer, completion transitions. vet/gofmt.

## Success Criteria

- [ ] Word-stream renders all 6 char-states correctly per §5.2; wraps at content width.
- [ ] Block cursor visible on current char; advances per keystroke; no flicker.
- [ ] Header correct & mode-specific; WPM updates ~250ms via wall-clock (not tick count).
- [ ] Time/Words/Quote completion each emits `ResultMsg` and transitions.
- [ ] `tab`/`ctrl+r`/`esc` behave per §8.3.
- [ ] Resize re-wraps live; no partial paint.
- [ ] Build/vet/gofmt clean; smoke test passes.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Per-char styling slow / flickers on big buffer | M×M | Trust v2 Cursed Renderer; precompute style objects once; cap visible window |
| Tick drift over 120s test | M×H | Wall-clock delta `now-startMs` (researcher-01 §5), never count ticks |
| Wrapping breaks on multi-byte / cursor misplace | L×H | Rune-based width calc; reuse lipgloss wrap; Unicode test in Phase 9/10 |
| Completion edge: Time mode exact-boundary | M×M | Completion checked in tickMsg vs limit; AFK trim handled in metrics |
