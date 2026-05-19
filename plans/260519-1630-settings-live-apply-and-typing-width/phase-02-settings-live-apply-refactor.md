---
phase: 2
title: Settings Live-Apply Refactor
status: completed
priority: P1
effort: 4h
dependencies:
  - 1
---

# Phase 2: Settings Live-Apply Refactor

## Overview

Convert Settings from a callback+pointer model (bound to the orphaned `New()`
copy) to the codebase's message pattern so theme, blink-cursor, default
mode/length and the `persistErr` toast apply live in-session.

## Context Links
- [brainstorm-summary.md](./brainstorm-summary.md) — root cause + chosen approach
- `internal/app/model.go:75`, `internal/app/model_settings.go` (callback site)
- `internal/ui/screen_settings.go`, `internal/ui/messages.go`
- `internal/app/smoke_test.go:170-222` (false rationalisation comment + weak test)

## Key Insights
- Root cause: `m.onSettingsChange` (pointer-receiver method value) + `&m.settings`
  captured in `New()`'s local `m`; `return m` copies → callback mutates a
  dead struct. Symptom: in-session no change, but value IS persisted so a
  relaunch shows it. Same path breaks blink/default-mode/toast.
- Codebase already routes screen→root via messages (`AbortMsg`,
  `StartTestMsg`, `NavCodePasteMsg`, `CodePastedMsg`) handled on the live
  `m` in `model.go Update`. This is the target pattern; the callback is the
  anomaly.

## Requirements
- Functional: cycling any Settings row updates the **rendered** model
  immediately (theme recolors all screens; blink toggles on the typing
  cursor; default mode/length re-seed Home; a forced save failure shows the
  `persistErr` toast that turn).
- Functional: `SettingsModel.sel` is preserved after a change (no jump to
  row 0).
- Non-functional: no public/CLI contract change (internal TUI only); files
  stay <200 LOC; no plan-artifact refs in code/test names or comments.

## Architecture

```
SettingsModel.Update (cycle) ──emits──> tea.Cmd { SettingsChangedMsg{Settings} }
                                              │
root Model.Update receives ───────────────────┘
  m.settings = msg.Settings
  storage.SaveSettings → on err set m.persistErr
  m.theme = theme.Load(...)
  rebuild m.home / m.typing / m.result / m.sett  (preserve sett.sel)
  return m, nil          (mutates the LIVE m value Bubble Tea drives)
```

- `SettingsModel` no longer holds `*config.Settings` or `onChange`. It owns a
  `config.Settings` value (copy) + its rows; `cycleSelected` mutates the
  local copy and returns `func() tea.Msg { return SettingsChangedMsg{copy} }`.
- Root reuses the existing `onSettingsChange` body but as an inline handler
  on the live `m` (the method/`func (m *Model)` form is deleted).
- Rebuild of `m.sett` must carry `sel` forward (constructor or a `WithSel`).

## Related Code Files
- Modify: `internal/ui/messages.go` (add `SettingsChangedMsg{Settings config.Settings}`)
- Modify: `internal/ui/screen_settings.go` (drop pointer/callback; emit cmd; keep sel)
- Modify: `internal/app/model.go` (handle `SettingsChangedMsg`; `NewSettings` call site)
- Modify: `internal/app/model_settings.go` (replace `onSettingsChange` method with
  root-side handler; `NewFromDisk` unchanged behaviour)
- Modify: `internal/ui/screen_settings_test.go` (`newTestSettings` signature;
  onChange-based tests → assert emitted `SettingsChangedMsg`)
- Modify: `internal/app/smoke_test.go` (delete false comment; real regression test)

## Implementation Steps (TDD)

### RED
1. In `internal/app/smoke_test.go` add `TestSmoke_Settings_ThemeAppliesLive`:
   drive root (`New(theme.Default(), Defaults(), "", "")`) → size → `'2'`
   (Settings) → `tea.KeyRight` on Theme row → execute returned `tea.Cmd` and
   feed the resulting msg back into root `Update` → assert
   `m.(Model).theme.Name() == "mono"` AND `sm_view(m)` differs from the
   pre-change frame. This FAILS on current code (callback hits orphan).
2. Add `TestSmoke_Settings_BlinkAppliesLive` (analogous: blink row → assert
   `m.(Model).settings.BlinkCursor` flips on the live model).
3. Lock adjacent: keep `TestNewSettingsExactly4Rows`, up/down clamp tests
   (unaffected by the signature change — adjust only constructor call).
4. Run `go test ./internal/app/... ./internal/ui/...` → confirm new tests RED,
   the rest GREEN (or compile-failing only where the signature changes).

### GREEN
5. `messages.go`: add `SettingsChangedMsg{ Settings config.Settings }` (config
   already imported) with a doc comment explaining the invariant (live apply
   on the root-owned model — no callback/pointer).
6. `screen_settings.go`: change `SettingsModel` to hold `s config.Settings`
   (value) + remove `onChange`; `NewSettings(s config.Settings, th, km)`;
   `cycleSelected` mutates local `s`, rebuilds dependent length row, returns
   `(m, func() tea.Msg { return SettingsChangedMsg{m.s} })`. `Update` returns
   that cmd. Keep `sel` field as-is.
7. `model.go`: `NewSettings` call site updated; in `Update` add
   `if sc, ok := msg.(ui.SettingsChangedMsg); ok { … }` placed beside the
   other message handlers; body = current `onSettingsChange` logic applied to
   the receiver `m`, rebuilding `m.sett` with `sel` preserved.
8. `model_settings.go`: delete `func (m *Model) onSettingsChange`; keep
   `NewFromDisk` (now constructs `SettingsModel` via the new signature).
9. Preserve `sel`: rebuild via `ui.NewSettings(...).WithSel(old)` (add tiny
   `WithSel`) OR mutate rows in place — pick the smaller change; do not reset
   to 0.
10. Update `screen_settings_test.go` `newTestSettings` to the new signature;
    rewrite onChange-assertion tests to execute the returned cmd and assert
    `SettingsChangedMsg` payload.
11. Replace `smoke_test.go:189-196` comment with a truthful one; ensure the
    two new live tests pass.
12. `gofmt`, `go vet`, `go test ./... -race` GREEN. Commit.

## Success Criteria
- [ ] New live regression tests (theme + blink) PASS; would FAIL on pre-fix code
- [ ] No `*config.Settings`/`onChange` left in `SettingsModel`; no
      `onSettingsChange` method
- [ ] `sel` preserved after a change
- [ ] Default-mode/length + `persistErr` toast verified live (test or manual note)
- [ ] False `smoke_test.go` comment removed; full `-race` suite GREEN
- [ ] All touched files <200 LOC; no plan-artifact refs in code/tests

## Risk Assessment
- Hidden callers of `onSettingsChange`/old `NewSettings` → grep before/after;
  compiler will catch signature breaks.
- Rebuild dropping `sel` → explicit preserve step + test.
- Over-scoping into Fix #1 → none of these files touch width/typing layout.

## Security Considerations
None — local settings persistence only; no new I/O surface.

## Next Steps
Phase 3 (typing width) once `-race` is fully green and committed.
