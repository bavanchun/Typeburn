---
phase: 9
title: Polish resize & NO_COLOR
status: completed
priority: P2
effort: ~4h
dependencies:
  - 7
  - 8
---

# Phase 9: Polish resize & NO_COLOR

## Overview

Robustness pass across ALL screens: degraded small-terminal mode, full NO_COLOR audit, narrow-terminal footer/logo collapse, keybinding consistency audit vs design §8, flicker/perf sanity, AFK-trim verification, Unicode/paste edge cases, and esc-on-Home quit prompt. No new features — harden what exists.

Refs: design §4.3 (degraded), §7 (accessibility/NO_COLOR), §5.4 (footer collapse), §8 (keybind source of truth); mockups "Responsive / Degraded".

## Requirements

### Functional
- Degraded mode: `termW<60 OR termH<20` on EVERY screen → single centered notice (`warning`) per §4.3, live re-render on `WindowSizeMsg`; never crash/partial-paint. Gate render in root `View` before delegating.
- NO_COLOR: full pass — every role maps to attribute-only (reverse/underline/bold/faint), layout identical, errors still underlined, cursor reverse-video, no raw hex leaks.
- Narrow (60–72 cols): footer drops action words → glyphs only (`↹ · ⌃r · esc`); word-stream re-wraps `termW-4`; logo `<64` plain-text fallback (verify all screens).
- Keybinding audit: every key in design §8 (global + per screen) implemented & only those; centralized in `internal/config`.
- `esc` on Home → quit confirmation prompt (per §8.1 / design §10 spec'd as quit-prompt), `ctrl+c` still hard-quits anywhere.
- AFK trim verified end-to-end in Time mode only (idle >7s at end trimmed; Words/Quote unaffected).

### Non-functional
- No flicker on resize/restart (trust v2 renderer; debounce not manual). Unicode/multi-byte + paste safe. Files <200 lines.

## Architecture

Single chokepoint: root `app.View()` checks size → degraded notice OR delegate. Footer/logo collapse driven by width tier helper. NO_COLOR theme already a swap (Phase 1) — this phase audits coverage, not re-architects.

```go
// internal/app
func (m Model) View() tea.View {
    if m.w < 60 || m.h < 20 { return degradedView(m.w, m.h, m.theme) } // ui.DegradedNotice
    /* delegate to active screen */
}
// internal/ui
func DegradedNotice(w, h int, th theme.Theme) string     // degraded-notice.go
func WidthTier(w int) Tier                                // wide|mid|narrow|degraded
func RenderFooter(... , tier Tier, ...)                   // collapse glyphs when narrow
// internal/app
type quitPromptModel struct{ /* y/n overlay on Home */ }  // esc-on-home
```

## Related Code Files

Create:
- `internal/ui/degraded-notice.go`
- `internal/ui/width-tier.go`
- `internal/app/quit-prompt.go`

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (size gate in View; esc-on-Home → quit prompt)
- `internal/ui/footer.go` (tier-aware glyph collapse)
- `internal/ui/ascii-logo.go` (confirm `<64` fallback on all screens)
- `internal/ui/word-stream-renderer.go` (narrow re-wrap `termW-4`)
- `internal/theme/theme.go` (NO_COLOR coverage fixes if audit finds gaps)
- per-screen files (apply WidthTier where footers/logos render)

Delete: none.

## Implementation Steps

1. Add `DegradedNotice` + size gate in root `View` (before screen delegate) — test by resizing below 60×20 on each screen.
2. `width-tier.go`: classify `w` → tier; thread tier into footer + logo + word-stream wrap.
3. Footer collapse: narrow tier → glyph-only hints per §5.4; verify each screen's footer.
4. NO_COLOR audit: run `NO_COLOR=1` through every screen + state (errors, cursor, selected rows, sparkline, badges); fix any role still emitting color; assert layout unchanged.
5. Keybinding audit: cross-check implemented bindings against design §8 table per screen; remove extras, add missing; ensure all sourced from `config.Keymap`.
6. esc-on-Home quit prompt: overlay y/n; `ctrl+c` unaffected.
7. Edge cases: paste (`tea.PasteMsg` → treat as sequential runes or ignore — decide & document; recommend: feed runes), multi-byte/CJK input into word-stream, very long quote wrap, rapid resize.
8. AFK verification: scripted Time-mode idle >7s tail → trimmed; Words/Quote idle → not trimmed.
9. Flicker/perf sanity on resize+restart (visual + no error). Build/vet/gofmt; full manual loop.

## Success Criteria

- [ ] Every screen shows degraded notice <60×20, recovers live on resize, never partial-paints.
- [ ] `NO_COLOR=1` full pass: no color anywhere, layout identical, errors underlined, cursor reverse-video.
- [ ] Narrow (60–72): footers glyph-only, word-stream re-wraps, logo fallback — all screens.
- [ ] Keybindings exactly match design §8 (no extras/missing), all from `config.Keymap`.
- [ ] esc-on-Home shows quit prompt; ctrl+c hard-quits everywhere.
- [ ] AFK trim Time-only verified; Unicode/paste edge cases handled without crash.
- [ ] No flicker on resize/restart; build/vet/gofmt clean.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| A screen bypasses size gate → partial paint | M×H | Single gate in root View before any delegation |
| NO_COLOR gap in a late-added component (sparkline/badge) | M×M | Explicit per-component NO_COLOR checklist in step 4 |
| Paste semantics unclear | M×L | Decide+document (feed runes sequentially); covered by edge test |
| Keybind drift from design | M×M | Audit table cross-check; centralized keymap is single source |
