# Brainstorm Summary — v1.1.0: Theme Packs + Hygiene

Date: 2026-05-18 · Status: agreed, ready for `/ck:plan`

## Problem statement

Typeburn is publicly released (v1.0.0/v1.0.1), production-ready. A project
audit surfaced cheap correctness/doc-hygiene gaps plus a product-upgrade
backlog. Decision: bundle hygiene fixes with ONE product feature into the next
milestone. Feature chosen: **theme packs**. Semver: new feature ⇒ **v1.1.0**
(minor), NOT a v1.0.2 patch.

## Audit validation (verified against live code, not accepted blindly)

| Claim | Verdict | Evidence |
|---|---|---|
| Stale README badge note | TRUE, real | `README.md:9` says badges "404 until first tag … pre-v1.0.0"; tags are live |
| `config`/`theme` import bubbletea/lipgloss | TRUE, doc-vs-reality drift not a bug | `config/keymap.go:3`, `theme/mono-theme.go:6`; no 2nd consumer (YAGNI) |
| Persistence errors silently swallowed | TRUE, real UX gap | `model_history.go:40` `_,_=AppendHistory`; `model_settings.go:24` `_=SaveSettings` |
| Word-wrap not word-aware; CJK `cellW:=1` | PARTLY | `word_stream_renderer.go:84` comment claims scan-back; impl hard-wraps. CJK = roadmap-accepted deferral (m5) |
| Atomic write skips parent-dir fsync | TRUE, already accepted | `atomic_write.go`; roadmap-accepted, no new data ⇒ do NOT reverse |

## Scope (agreed)

### Feature: 6 theme packs
Solarized Dark, Solarized Light, Dracula, Nord, Gruvbox Dark, Gruvbox Light.
Total 8 selectable (+ `default` + `mono`).

Architecture (verified): theme = `Theme{name, colors map[Role]color.Color,
noColor}` (`theme.go:13`), 16 Roles (`roles.go:12-27`). `Load()` is a
`switch name` (`theme.go:36`); **NO_COLOR short-circuits before the switch**
⇒ packs auto-inert under NO_COLOR, zero extra work. Settings picker
auto-derives from `theme.Available()` (`settings_rows.go:30`) ⇒ **no UI code
touched**.

Per pack: one new `internal/theme/<name>-theme.go` with `func <Name>() Theme`
returning the 16-Role map (mirror `default-theme.go:12-34`). Extend `Load()`
switch + `Available()` slice. Add palettes to `design-guidelines.md §2.1`
(documented color source of truth).

### DRY decision (the one real architectural choice) — Approach A
Keep `config/settings.go:51` hardcoded valid-theme switch (core layering
forbids `config`→`theme` import). Add a sync **test** asserting the Normalize
switch and `theme.Available()` stay in lockstep + a code comment explaining
the intentional duplication. KISS, zero layering risk, drift-proof.

### Hygiene bundled in
- IN: README/docs staleness (`README.md:9` + roadmap badge notes);
  silent-persistence-failure non-blocking UI notice
  (`model_history.go:40`, `model_settings.go:24`); fix the misleading
  word-wrap comment (`word_stream_renderer.go:84`); update the core-import
  doc rule (CLAUDE.md / architecture docs) to match reality.
- DEFER: real word-aware wrap implementation (bigger than hygiene; only bites
  Quote mode on words longer than line width); parent-dir fsync
  (roadmap-accepted, no new data → not reversed).

## Risks

- **Light themes (Solarized Light, Gruvbox Light):** the 16 Roles were
  palette-tuned for dark backgrounds. Each light pack needs explicit per-Role
  contrast review — esp. `RoleBg`, `RoleSurface`, `RoleSurfaceAlt`,
  `RoleTextPrimary/Muted/Faint`, `RoleCursorFg`, `RoleErrorBg`. Mitigation:
  dedicate a phase step to eyeball each light theme across all 5 screens.
- **Picker UX with 8 themes:** cycle-through row may feel long. Acceptable for
  v1.1; revisit only if user feedback.
- **Persistence-notice surface:** must be non-blocking and not break NO_COLOR
  layout (attribute-only). Reuse the existing degraded-notice pattern
  (`internal/ui/degraded-notice.go`).

## Success criteria

- 8 themes selectable; each resolves all 16 Roles non-nil; unknown→default;
  NO_COLOR unaffected.
- Sync test fails if `Available()` and `Normalize()` lists diverge.
- Disk-write failure shows a visible, non-blocking notice; no crash.
- README/roadmap/CLAUDE.md no longer carry stale/false statements.
- `gofmt`, `go vet`, `go test ./... -race -count=1` green; prod files <200 LOC.
- Ships via protected-main flow: branch → per-phase commits → PR →
  squash-merge → tag `v1.1.0` on merged SHA (CONTRIBUTING release runbook).

## Out of scope (v1.1.0)

Code/custom-text mode, custom wordlists, history import/export, real
word-aware wrap, CJK width, parent-dir fsync, online sync. (Roadmap backlog.)

## Open questions

None — both architectural decisions locked with the user.
