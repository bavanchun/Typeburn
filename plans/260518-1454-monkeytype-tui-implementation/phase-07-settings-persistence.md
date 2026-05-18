---
phase: 7
title: "Settings & persistence"
status: pending
priority: P1
effort: ~5h
dependencies: [6]
---

# Phase 7: Settings & persistence

## Overview

Settings screen + atomic JSON persistence. v1 rows (ALL functional, nothing else): Theme, Default test mode, Default length, Blink cursor. Auto-persist on change; corrupt/missing file → sane defaults, never crash. Apply effects live: blink → typing cursor, default mode/length → Home init, theme → live swap.

Refs: design §5.5 (selectable row), §8.5 (keys), §2 (theme contract); mockups §4 (EXCLUDE error-mode/smooth-cursor/restart-flash/sound — do NOT render them).

## Requirements

### Functional
- `internal/storage`: load/save `config.Settings` JSON at `$XDG_CONFIG_HOME/monkeytype-tui/settings.json` (fallback `~/.config/monkeytype-tui/`). Atomic write (temp file + rename). Corrupt/unparseable/missing → `config.Defaults()`, no crash.
- Settings screen rows (exactly 4): Theme (cycles `theme.Available()` = {default, mono}; "solarized-dark" reserved, NOT selectable), Default mode (time/words/quote), Default length (mode-appropriate set), Blink cursor (on/off).
- Keys (§8.5): `↑↓`/`j k` move; `←→`/`h l`/`enter` cycle value; `esc`/`1` save & back (auto-persist already done on each change — esc just returns).
- Auto-persist: every value change → debounced/immediate atomic save.
- Apply: blink → `TypingModel.blink`; default mode/length → Home seed; theme change → rebuild theme map live across app.

### Non-functional
- Atomic, no partial files. Defaults on any read failure. Files <200 lines. Unit-tested (incl. corrupt + XDG fallback).

## Architecture

Data flow: load at startup (`app` Init) → `Settings` held in root model → Settings screen mutates copy → on change `storage.SaveSettings` (atomic) + propagate (theme rebuild, Home re-seed). Theme cycle limited to `theme.Available()`.

```go
// internal/storage
func SettingsPath() (string, error)            // config.ConfigDir()+"/settings.json"
func LoadSettings() config.Settings            // missing/corrupt → config.Defaults(), never error-out
func SaveSettings(s config.Settings) error     // write temp → fsync → os.Rename (atomic)
func atomicWrite(path string, data []byte) error

// internal/ui
type SettingsModel struct {
    rows   []settingRow            // 4 rows
    sel    int
    s      *config.Settings        // pointer to root settings
    th     theme.Theme
    km     config.Keymap
    w, h   int
    onChange func(config.Settings) // triggers save + propagate
}
type settingRow struct { label string; values []string; idx int; help string }
func NewSettings(s *config.Settings, th theme.Theme, km config.Keymap, onChange func(config.Settings)) SettingsModel
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd)
func (m SettingsModel) View() string

// internal/theme
func Available() []string  // v1: ["default","mono"]; "solarized-dark" reserved, NOT returned
```

`onChange` callback in `app` performs `storage.SaveSettings` + theme rebuild + propagate to Home/Typing builders.

## Related Code Files

Create:
- `internal/storage/settings-store.go`
- `internal/storage/atomic-write.go`
- `internal/storage/settings-store_test.go` (load/save/corrupt/XDG fallback)
- `internal/ui/screen-settings.go`

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (load settings at Init; real Settings screen; onChange wiring; theme rebuild; Home/Typing consume settings)
- `internal/ui/screen-typing.go` (consume `blink`: 530ms toggle cursor bg per §5.1)
- `internal/ui/screen-home.go` (seed from settings defaults)
- `internal/config/settings.go` (ensure JSON tags)

Delete: none.

## Implementation Steps

1. `atomic-write.go`: write to `path+".tmp"`, `f.Sync()`, `os.Rename` over target; 0600 perms; mkdir -p config dir.
2. `settings-store.go`: `LoadSettings` → read file; on `os.IsNotExist` OR `json.Unmarshal` error OR invalid enum → return `config.Defaults()` (never propagate error to UI). `SaveSettings` via atomicWrite.
3. `screen-settings.go`: 4 rows only; selectable-list (Phase 5 component) styling per §5.5; help line under separator = selected row help; on value cycle call `onChange`. `esc/1` return.
4. `app`: at Init `s := storage.LoadSettings()`; build theme from `s.Theme`; pass `&s` + onChange (save + rebuild theme + re-seed Home + update Typing blink) into SettingsModel.
5. Typing: implement blink (530ms tick toggling cursor bg between `cursor-bg`/`surface-alt`) gated on `s.BlinkCursor`; steady when off.
6. Tests: save→load round-trip; corrupt JSON → defaults; missing file → defaults; XDG_CONFIG_HOME set vs unset (HOME fallback); atomic (no `.tmp` left, no partial on simulated error).
7. Build, run: change each setting, restart app → persisted; theme/blink/default apply live. vet/gofmt.

## Success Criteria

- [ ] Exactly 4 settings rows; excluded options absent from UI entirely.
- [ ] Each change auto-persists atomically (no `.tmp` residue, no partial file).
- [ ] Corrupt/missing settings.json → defaults, app starts normally.
- [ ] XDG_CONFIG_HOME honored; HOME `~/.config` fallback works (tested).
- [ ] Blink toggles cursor (530ms) when on, steady when off; default mode/length seed Home; theme swap live.
- [ ] Theme cycles {default, mono} via `theme.Available()`; switching applies live; "solarized-dark" not selectable.
- [ ] `go test ./internal/storage/... -race` passes; build/vet/gofmt clean.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Non-atomic write corrupts settings on crash | L×H | temp+fsync+rename; test asserts no partial |
| Theme rebuild not propagated to live screens | M×M | Central onChange rebuilds theme + re-injects into all sub-models |
| Default length invalid for chosen default mode | M×M | Length value set keyed to mode; clamp on mode change |
| XDG fallback wrong (macOS no XDG vars) | L×M | Explicit HOME fallback; both paths unit-tested |
