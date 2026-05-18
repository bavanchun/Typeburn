---
phase: 2
title: "Theme Packs"
status: pending
priority: P1
effort: "4h"
dependencies: [1]
---

# Phase 2: Theme Packs

## Overview
Add 6 theme packs as pure 16-Role data files; wire into `Load()` +
`Available()`; keep `config.Normalize()` valid-list DRY via Approach A
(documented duplication + sync test). One commit.

## Requirements
- Functional: 8 themes selectable (`default`, `mono`, + 6 packs). Settings
  picker auto-lists them (no UI code change — derives from `theme.Available()`).
- Functional: each pack resolves **all 16 Roles** to non-nil colors.
- Functional: unknown name → `default`; `NO_COLOR` unaffected (Load
  short-circuits before name switch — verify, do not modify).
- Non-functional: every new prod file < 200 LOC; `gofmt`/`vet` clean;
  `theme` package retains 100% coverage.

## Architecture
Theme = `Theme{name, colors map[Role]color.Color, noColor}` (`theme.go:13`).
16 Roles in `roles.go:12-27`. Each pack mirrors `default-theme.go:12-34`
exactly (one constructor func returning the role map). `Load()` switch
(`theme.go:36`) + `Available()` slice (`theme.go:24`) extended.

**Approach A (locked):** `config/settings.go:51` Normalize switch stays
hardcoded (core layering forbids `config`→`theme` import). Add:
(a) a code comment on that switch explaining the intentional duplication and
pointing at the sync test; (b) a sync test in `internal/config` (or
`internal/theme`, wherever it can import both legitimately — the test package
may import both) asserting the Normalize-accepted set == `theme.Available()`.

## Related Code Files
- Create: `internal/theme/solarized-dark-theme.go`,
  `internal/theme/solarized-light-theme.go`,
  `internal/theme/dracula-theme.go`, `internal/theme/nord-theme.go`,
  `internal/theme/gruvbox-dark-theme.go`,
  `internal/theme/gruvbox-light-theme.go`
- Create: `internal/theme/theme_available_sync_test.go` (sync test)
- Modify: `internal/theme/theme.go` (`Load` switch + `Available` slice + drop
  the stale "solarized-dark reserved" comment), `internal/config/settings.go`
  (`Normalize` switch + explanatory comment), `internal/theme/theme_test.go`
  (table-extend: all-16-roles-non-nil per pack, name round-trip),
  `internal/config/settings_test.go` (Normalize accepts new names),
  `docs/design-guidelines.md` §2.1 (add 6 palettes)

## Palette spec (16 Roles each — apply verbatim)

Roles order: Bg, Surface, SurfaceAlt, TextPrimary, TextMuted, TextFaint,
Accent, AccentDim, Error, ErrorBg, Warning, Success, CursorBg, CursorFg,
Border, BorderFocus.

- **Solarized Dark:** `#002b36 #073642 #094d5c #93a1a1 #839496 #586e75
  #268bd2 #1f6a9e #dc322f #4a1715 #b58900 #859900 #268bd2 #002b36 #586e75
  #268bd2`
- **Solarized Light:** `#fdf6e3 #eee8d5 #ded8c5 #586e75 #657b83 #93a1a1
  #268bd2 #3a7ca5 #dc322f #f5d0cd #b58900 #859900 #268bd2 #fdf6e3 #93a1a1
  #268bd2`
- **Dracula:** `#282a36 #343746 #44475a #f8f8f2 #c9c9d1 #6272a4 #bd93f9
  #7d5bbe #ff5555 #5c1a1a #f1fa8c #50fa7b #bd93f9 #282a36 #6272a4 #bd93f9`
- **Nord:** `#2e3440 #3b4252 #434c5e #eceff4 #d8dee9 #4c566a #88c0d0
  #5e81ac #bf616a #4a2326 #ebcb8b #a3be8c #88c0d0 #2e3440 #4c566a #88c0d0`
- **Gruvbox Dark:** `#282828 #3c3836 #504945 #ebdbb2 #a89984 #665c54
  #fabd2f #b57614 #fb4934 #4a1f1c #fe8019 #b8bb26 #fabd2f #282828 #665c54
  #fabd2f`
- **Gruvbox Light:** `#fbf1c7 #ebdbb2 #d5c4a1 #3c3836 #7c6f64 #bdae93
  #b57614 #79740e #9d0006 #f2d0cd #af3a03 #79740e #b57614 #fbf1c7 #bdae93
  #b57614`

Names (for `Available()` / `Normalize`): `solarized-dark`,
`solarized-light`, `dracula`, `nord`, `gruvbox-dark`, `gruvbox-light`.
`Available()` order: `default, mono, solarized-dark, solarized-light,
dracula, nord, gruvbox-dark, gruvbox-light`.

## Implementation Steps
1. Create 6 `*-theme.go` files, each `func <Name>() Theme` returning the
   16-Role map (mirror `default-theme.go`). Use `lipgloss.Color("#…")`.
2. `theme.go`: add 6 `case` arms to `Load()`; extend `Available()`; remove
   the obsolete "solarized-dark reserved but not implemented" comment.
3. `settings.go` `Normalize()`: add 6 names to the switch; add comment
   "Intentional duplication of theme.Available() — core layering forbids
   importing theme here; theme_available_sync_test.go guards drift."
4. Sync test (`package config_test` or `theme_test` — external test pkg may
   import both without violating layering; no new exported API on `config`):
   for each `name` in `theme.Available()`, construct
   `s := config.Settings{Theme: name}`, call `s.Normalize()`, assert
   `s.Theme == name` (every Available theme is accepted by Normalize); then
   `s := config.Settings{Theme: "definitely-bogus"}; s.Normalize();` assert
   `s.Theme == "default"` (unknown resets). This proves the two lists stay in
   lockstep WITHOUT config exposing its internal set.
5. Extend `theme_test.go`: per pack, assert `Load(name,false).Name()==name`
   and every Role in `roles.go` resolves non-nil; assert
   `Load("bogus",false)` == default; assert `Load(name,true)` == no-color.
6. Extend `settings_test.go`: Normalize keeps each new valid name; unknown
   → `default`.
7. **Light-theme contrast review:** run `make run`, switch to
   `solarized-light` and `gruvbox-light`, eyeball all 5 screens (Home,
   Typing, Result, History, Settings). Check Bg/Surface/SurfaceAlt vs
   TextPrimary/Muted/Faint legibility, CursorFg over CursorBg, ErrorBg
   marker. Adjust hex only if a pair fails contrast; record any change in
   `design-guidelines.md`.
8. Add the 6 palettes to `docs/design-guidelines.md` §2.1.
9. `make fmt && make lint && make test-race`. Commit:
   `feat(theme): add Solarized/Dracula/Nord/Gruvbox theme packs`.

## Success Criteria
- [ ] 8 themes selectable; picker lists all (manual `make run`).
- [ ] Each pack: 16/16 Roles non-nil; name round-trips; unknown→default;
  NO_COLOR unaffected — all asserted by tests.
- [ ] Sync test fails if Normalize list and `Available()` diverge (verify by
  temporarily removing one entry, see red, restore).
- [ ] Light themes legible on all 5 screens.
- [ ] `gofmt`/`vet`/`test -race` green; new files < 200 LOC; coverage ≥ prior.
- [ ] One commit on the feature branch.

## Risk Assessment
- **Light themes designed for dark bg:** mitigated by step 7 explicit review.
  Optional cheap guard: a test asserting relative luminance of `RoleBg` vs
  `RoleTextPrimary` is "light bg / dark text" for `*-light` and inverse for
  dark themes — catches an accidentally inverted palette without subjective
  judgement. Add only if trivial; visual review remains the real gate.
- **Palette drift from official sources:** hex values pinned in this spec;
  cite sources in `design-guidelines.md`.
- **Sync-test placement:** must live where importing both `config` and
  `theme` is legal (a `_test.go` in either package's external test package
  `package theme_test` / `package config_test` can import both).
