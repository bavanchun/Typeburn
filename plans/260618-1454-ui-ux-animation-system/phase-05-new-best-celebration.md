---
phase: 5
title: "New-best celebration"
status: completed
priority: P3
effort: "4h"
dependencies: [2, 4]
---

# Phase 5: New-best celebration

## Overview
When a test beats the per-mode personal best (the ★), fire a short, self-contained
sparkle/confetti burst around the badge. Triggers **only on a new best** (never on
ordinary results), then settles. Overlays existing cells — never shifts layout.

## Requirements
- Functional: ~1s particle burst anchored to the new-best badge; new-best only; one-shot.
- Non-functional: overlay onto blank/padding cells only (no reflow); NO_COLOR → attribute
  sparkle (reverse/bold) with identical cell count; stdlib particle math.

## Architecture
**Trigger:** the root already computes new-best when handling `ResultMsg`
(`model.go:97` → `handleResultMsg`, the only place that detects new-best per CLAUDE.md).
Pass an `isNewBest bool` into `ResultModel` on **every** `ResultMsg` (true OR false — never
leave prior-result residue, or an ordinary result inherits `isNewBest=true` and fires spurious
confetti). When true, set `celebrateStartMs = revealStartMs` (or slightly after the badge
appears) so the burst rides the same frame loop from P4; when false, zero `celebrateStartMs`.

**Particle model (`celebration.go`):** small fixed-capacity set (research: 5–7 cells) in a
bounded region. Each particle: `{dx, dy, bornMs, lifeMs, glyph}`. Deterministic "randomness"
without `math/rand` global: derive jitter from `(index, revealStartMs)` so renders are
reproducible for golden tests. **Glyphs are ASCII width-1 ONLY** (`* + · .`) — no `✧`/non-BMP
runes: the renderer hard-codes `cellW := 1` (`word_stream_renderer.go:124`), and a double-width
or missing-glyph rune would break layout. Assert every glyph has display width 1.

**Lifetime / render (red-team P2 — NOT the toast technique):** at `nowMs`, a particle is visible
while `nowMs-bornMs < lifeMs` (`lifeMs≈250–300`); it twinkles on a short sub-cycle. The
persistence-toast at `model_view.go:74` only rewrites the *single last line*; confetti needs a 2D
region, and surgically replacing a rune at a visual column inside an already-SGR-styled line is
error-prone (byte offset ≠ column offset). **Therefore restrict the burst to full-width blank
MARGIN rows** (unstyled padding lines above/below the result card block), where rune replacement
is trivial and width-safe. Never splice into styled content lines.

**NO_COLOR:** particles render via `Reverse(true)`/`Bold(true)` on the glyph instead of accent color.
Cell count and rune width unchanged → layout-identical (glyph content differs by design).

**HasActiveAnim:** `ResultModel` extends its active window to include
`nowMs < celebrateStartMs + celebrateMs` (`celebrateMs≈1000`).

## Related Code Files
- Create: `internal/ui/celebration.go` — particle set, `renderCelebration(nowMs, region, theme) []overlayCell`.
- Create: `internal/ui/celebration_test.go` — particle lifetime, deterministic jitter, NO_COLOR attr,
  overlay never changes line count / line width.
- Modify: `internal/ui/screen_result.go` — `isNewBest`, `celebrateStartMs`; extend `HasActiveAnim`.
- Modify: `internal/ui/screen_result_view.go` — apply celebration overlay onto the rendered frame.
- Modify: `internal/app/model_result.go` — pass `isNewBest` into ResultModel construction.

## Implementation Steps
1. `celebration.go`: define particles anchored to a region rect; deterministic jitter from index+seed;
   `renderCelebration` returns overlay cell positions+styled glyphs.
2. Plumb `isNewBest` from the root’s new-best detection into `ResultModel`.
3. Set `celebrateStartMs` (only when `isNewBest`); extend `HasActiveAnim` window.
4. In result view, after composing the frame, splat overlay cells onto existing positions
   (reuse the line-splice technique from `model_view.go`), guarding against out-of-bounds.
5. Tests: assert overlay preserves `len(lines)` and each line’s rune width; NO_COLOR attr-only;
   non-new-best result emits zero particles.

## Success Criteria
- [x] New best → visible ~1s sparkle in the card’s blank margin rows; ordinary result → nothing.
- [x] A non-new-best result immediately following a new-best emits ZERO particles (no residue).
- [x] All glyphs are ASCII display-width 1 (asserted); overlay only touches blank margin rows.
- [x] Overlay never changes line count or any line’s rune width (asserted).
- [x] Under `NO_COLOR`, sparkle is attribute-only; layout-identical (line count + rune width).
- [x] Burst is one-shot; `HasActiveAnim` false after `celebrateMs`; loop self-stops.
- [x] `make test-race`, `go vet`, `gofmt -l` clean; files < 200 LOC.

## Risk Assessment
- **Annoyance / overuse:** restricted to new-best only and one-shot — matches research guidance.
- **Overlay clipping at min terminal (60×20):** clamp region to the result frame bounds; if the
  badge area is too tight, degrade to a minimal in-line sparkle rather than risk overrun.
- **Determinism vs `math/rand`:** use index+seed jitter (no global RNG) so golden tests are stable.
