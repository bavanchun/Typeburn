---
phase: 1
title: Scaffold theme & app skeleton
status: completed
priority: P1
effort: ~4h
dependencies: []
---

# Phase 1: Scaffold theme & app skeleton

## Overview

Bootstrap the Go module, pin Charm v2 deps, and stand up the Elm root model with screen-enum routing (placeholder views). Theme (roles + default dark + NO_COLOR swap) and config (keybindings + defaults + XDG paths) land here so all later screens build on them. End state: binary launches, shows placeholder Home, `ctrl+c` quits.

Refs: researcher-01 §1,2,7,9; design-guidelines §2.1, §8, §9.

## Requirements

### Functional
- App launches into Home screen (placeholder text).
- Screen-enum routing: Home/Typing/Result/Settings/Quit reachable structurally (stub views).
- Global keys wired: `ctrl+c` quit, `1` Home, `2` Settings, `3` History, `esc` back.
- Theme exposes role lookup; `NO_COLOR` env → attributes-only theme.

### Non-functional
- Exact Charm v2 module path pinned in `go.mod`.
- Every file <200 lines, kebab-case.
- Cross-platform (Linux/macOS) — no OS-specific syscalls.

## Architecture

Data flow: `main.go` builds root `app.Model` → `tea.NewProgram(m).Run()`. `Update` switches on `m.screen` enum, delegates `msg` to active sub-model's `Update`, batches `tea.Cmd`. `View` returns active sub-model `View()` placed via `lipgloss.Place`.

```go
// internal/app
type Screen int
const ( ScreenHome Screen = iota; ScreenTyping; ScreenResult; ScreenSettings; ScreenHistory )

type Model struct {
    screen Screen
    w, h   int
    theme  theme.Theme
    keys   config.Keymap
    // sub-models added in later phases (placeholders now)
}
func (m Model) Init() tea.Cmd
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) // WindowSizeMsg, KeyPressMsg(global), route
func (m Model) View() tea.View

// internal/theme
type Role int
const ( RoleBg Role = iota; RoleSurface; RoleSurfaceAlt; RoleTextPrimary; RoleTextMuted;
        RoleTextFaint; RoleAccent; RoleAccentDim; RoleError; RoleErrorBg; RoleWarning;
        RoleSuccess; RoleCursorBg; RoleCursorFg; RoleBorder; RoleBorderFocus )
type Theme struct { name string; colors map[Role]lipgloss.Color; noColor bool }
func Default() Theme           // dark, truecolor hex per design §2.1
func Mono() Theme              // greyscale: accent→bold, error→underline, single-hue (design §2.2 "mono")
func NoColorTheme() Theme       // attributes-only sentinel (NO_COLOR env)
func Available() []string      // v1: ["default","mono"]  (solarized-dark reserved, not returned)
func Load(name string, noColor bool) Theme  // unknown name → Default()
func (t Theme) Style(r Role) lipgloss.Style

// internal/config
type Settings struct { Theme string; DefaultMode string; DefaultLength int; BlinkCursor bool }
func Defaults() Settings        // theme=default, mode=time, length=30, blink=false
type Keymap struct { /* key.Binding per design §8 global+per-screen */ }
func DefaultKeymap() Keymap
func ConfigDir() (string, error) // $XDG_CONFIG_HOME/monkeytype-tui  | ~/.config/...
func DataDir() (string, error)   // $XDG_DATA_HOME/monkeytype-tui    | ~/.local/share/...
```

## Related Code Files

Create:
- `main.go`
- `internal/app/model.go`, `internal/app/routing.go`
- `internal/theme/theme.go`, `internal/theme/roles.go`, `internal/theme/default-theme.go`, `internal/theme/mono-theme.go`
- `internal/config/settings.go`, `internal/config/keymap.go`, `internal/config/xdg-paths.go`
- `go.mod`, `go.sum`

Modify: none. Delete: none.

## Implementation Steps

1. `go mod init monkeytype-tui` (Go 1.26 in `go.mod`).
2. **Verify Charm v2 path**: try `go get github.com/charmbracelet/bubbletea/v2@latest` AND `go get charm.land/bubbletea/v2@latest`; pick whichever resolves; repeat for `lipgloss/v2`, `bubbles/v2`. Default v2. If neither v2 resolves → STOP, escalate to user (never silently fall back to v1). Pin resolved versions in `go.mod`.
3. `internal/theme`: define `Role` enum, `Theme` struct, `Default()` (truecolor hex per §2.1), `Mono()` (greyscale per §2.2 — accent=bold, error=underline), `NoColorTheme()`, `Available()` (["default","mono"]), `Load` (unknown→Default), `Style(role)` (NO_COLOR → attribute-only style: reverse/underline/bold/faint per §2.1 fallback col).
4. `internal/config`: `Settings`+`Defaults`; `Keymap` with all bindings from design §8 (global + per-screen) via Bubbles `key.Binding`; `xdg-paths.go` ConfigDir/DataDir with XDG env then HOME fallback.
5. `internal/app`: `Screen` enum, `Model` (screen/w/h/theme/keys), `Init` (returns nil cmd), `Update` (handle `tea.WindowSizeMsg` store w/h; global keys: ctrl+c→`tea.Quit`, 1/2/3 switch screen, esc back), `routing.go` placeholder `View` per screen ("[Home]", etc.) centered via `lipgloss.Place`.
6. `main.go`: detect `os.Getenv("NO_COLOR")`, build theme, `tea.NewProgram(app.New(...))`, run, exit non-zero on err.
7. `go build ./...` then `go vet ./...`; run binary, confirm Home placeholder + ctrl+c quits.

## Success Criteria

- [ ] Charm v2 module path resolved & pinned in go.mod (documented which path won).
- [ ] `go build ./...` succeeds; `go vet ./...` clean; `gofmt -l` empty.
- [ ] Binary runs: shows centered Home placeholder.
- [ ] `ctrl+c` quits; `1/2/3` switch placeholder screens; `esc` returns to Home.
- [ ] `NO_COLOR=1 ./monkeytype-tui` runs without color (no crash).
- [ ] All files <200 lines, kebab-case.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Charm v2 path differs from research (github vs charm.land) | M×H | Step 2 dual-probe; escalate if v2 unresolvable — do not assume |
| v2 API (tea.View struct, KeyPressMsg) differs from v1 habits | M×M | Follow researcher-01 §3,4; consult context7 docs-seeker on signature errors |
| XDG fallback wrong on macOS (no XDG vars) | L×M | Explicit HOME-based fallback; unit-tested in Phase 7 |
