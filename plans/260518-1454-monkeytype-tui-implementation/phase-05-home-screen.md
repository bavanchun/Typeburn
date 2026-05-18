---
phase: 5
title: Home screen
status: completed
priority: P1
effort: ~4h
dependencies:
  - 4
---

# Phase 5: Home screen

## Overview

Real Home screen: ASCII logo (with narrow fallback), mode selector tabs (Time/Words/Quote), per-mode length options, start action, and navigation to Settings/History. Replaces the Phase 1 placeholder and the Phase 4 placeholder start trigger.

Refs: design §8.2 (keys), §5.5 (selectable rows/tabs), §6 (tab states); mockups §1.

## Requirements

### Functional
- ASCII block-art `MONKEYTYPE` logo in `accent`, trailing `type` in `text-muted`. If `termW < 64` → plain `Bold accent "monkeytype"`.
- Mode selector: Time / Words / Quote tabs. Active tab `accent + Bold + ▎`; inactive `text-muted`.
- Length options per mode: Time {15,30,60,120}; Words {10,25,50,100}; Quote {short,medium,long}. Chosen value `accent Bold [ ]`; siblings `text-muted`.
- Keys (§8.2): `tab`/`shift+tab` cycle mode; `←→`/`h l` change length option; `enter`/`space` start → launch TypingModel with chosen mode+length; `2` Settings, `3` History.
- Initial mode/length seeded from settings defaults (settings real in Phase 7; until then use `config.Defaults()`).

### Non-functional
- Centered single block via `lipgloss.Place`. Rune-safe. Files <200 lines.

## Architecture

Data flow: Home holds `selMode`, `selLen` (index per mode). `enter` → `app` swaps to `NewTyping(selMode, selLen, quoteLen, theme, keys)`. Quote length maps to `words.QuoteLen`.

```go
// internal/ui
type HomeModel struct {
    mode    typing.Mode
    lenIdx  map[typing.Mode]int   // selected option index per mode
    w, h    int
    th      theme.Theme
    km      config.Keymap
    defaults config.Settings
}
func NewHome(s config.Settings, th theme.Theme, km config.Keymap) HomeModel
func (m HomeModel) Update(msg tea.Msg) (HomeModel, tea.Cmd)
func (m HomeModel) View() string
type StartTestMsg struct{ Mode typing.Mode; Length int; QuoteLen words.QuoteLen }

// reusable
func RenderLogo(width int, th theme.Theme) string        // ascii-logo.go (+narrow fallback)
func RenderTabs(opts []string, active int, th theme.Theme) string // selectable-list.go (tab mode)
func RenderOptions(opts []string, chosen int, th theme.Theme) string
```

Length option lists are static per mode; `lenIdx` persists per mode so switching tabs keeps each mode's chosen length.

## Related Code Files

Create:
- `internal/ui/screen-home.go`
- `internal/ui/ascii-logo.go`
- `internal/ui/selectable-list.go` (shared by Home tabs + Settings/History later)

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (use real HomeModel; `StartTestMsg` → TypingModel; `2`/`3` nav)

Delete: none (Home placeholder removed by replacement).

## Implementation Steps

1. `ascii-logo.go`: embed/const the block-art lines; `RenderLogo(width,…)` returns art or narrow fallback `<64`.
2. `selectable-list.go`: generic styled row/tab renderer per §5.5/§6 (used here for tabs+options; reused Phase 7/8).
3. `screen-home.go`: state init from `config.Settings`; `Update` per §8.2 keys (cycle mode, change option, start, 2/3 nav); `View` = JoinVertical(logo, blank, tabs, blank, options, blank, "press enter to start", spacer, footer) centered.
4. Emit `StartTestMsg` on enter/space; `app` builds TypingModel; map Quote option → `words.QuoteLen`.
5. Wire `2`→Settings placeholder, `3`→History placeholder (real in 7/8).
6. Build, run: cycle modes, change lengths, start each mode, verify routing back from Phase 4 result → Home. vet/gofmt.

## Success Criteria

- [ ] Logo renders; narrow (`<64`) fallback shows plain text, no overflow.
- [ ] Mode tabs cycle with tab/shift+tab; active styling per §6.
- [ ] Length options change with ←→/h l; per-mode selection persists across tab switches.
- [ ] enter/space starts the correct mode+length test (Time/Words/Quote).
- [ ] `2`/`3` navigate; full loop Home→Typing→(result)→Home works.
- [ ] Build/vet/gofmt clean.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Logo overflow on narrow terminals | M×M | `<64` fallback; centered Place clips gracefully |
| Quote length → QuoteLen mapping mismatch | L×M | Single mapping fn, unit-checked |
| Per-mode length state lost on tab switch | M×L | `lenIdx` map keyed by mode, not single int |
