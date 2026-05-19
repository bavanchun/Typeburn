---
phase: 4
title: "Code Renderer + Viewport"
status: pending
priority: P1
effort: "5h"
dependencies: [1]
---

# Phase 4: Code Renderer + Viewport (TDD)

## Overview
New isolated `internal/ui/code_stream_renderer.go`: render the rune buffer
with **literal** line breaks (`\n`) and 2-col tab (`\t`) visuals, per-char
`theme.Role` state styling, plus a vertical viewport that scrolls to keep the
caret line visible. `word_stream_renderer.go` and its golden tests are NOT
touched. One commit.

## Requirements
- Functional: `RenderCodeStream(target, typed []rune, states
  []typing.CharState, th theme.Theme, width, height int) string`. Splits on
  literal `\n` into rows; `\t` rendered as 2 visual columns (the target rune
  stays a single `\t`; caret/state map 1:1 to runes); reuses the same
  per-char state→Role styling as the word stream (correct/incorrect/cursor/
  untyped). Long rows: hard-wrap at `width` (continuation rows) so no
  horizontal scroll. **Viewport:** show a height-bounded window of rows that
  always contains the caret row (scroll-follow); caret near top → window from
  0; near bottom → clamps so last rows show; file shorter than height → no
  scroll, top-aligned.
- Non-functional: NO_COLOR-safe (attribute-only path identical layout);
  pure function (no Bubble Tea state); file <200 LOC (split helpers into
  `code_viewport.go` if needed, each <200).

## Architecture
Approach A — isolation. Rune index ↔ (row,col) mapping computed from `\n`
positions; tab expands visual width only (display), never the rune/state
index. Viewport = compute caretRow, then `start = clamp(caretRow - margin, 0,
maxStart)`, render rows `[start, start+height)`. Reuse the existing
state→`theme.Style(Role)` helper from the word renderer if exported/shared;
if private, extract a tiny shared `char_style.go` (no behaviour change to
word stream — covered by its untouched goldens).

## Related Code Files
- Create: `internal/ui/code_stream_renderer.go`,
  `internal/ui/code_stream_renderer_test.go` (+ `code_viewport.go` /
  `_test.go` if split for <200 LOC)
- Modify: possibly extract a shared `internal/ui/char_style.go` ONLY if the
  per-char styler is currently private to word_stream (behaviour-neutral;
  word-stream goldens must stay green to prove it)
- Delete: none

## Implementation Steps (tests-first)
1. **RED:** `code_stream_renderer_test.go` cases (string-asserted, ANSI
   stripped or NO_COLOR theme for determinism):
   - `"a\nb"` → two rows; tab `"\tx"` → 2 spaces then `x`.
   - state styling: an incorrect rune carries the error treatment; cursor on
     the right rune; untyped faint — same expectations as word-stream tests.
   - viewport: target of N rows with height H<N — caret on row 0 → window
     `[0,H)`; caret on last row → window ends at N (last rows visible); caret
     mid → caret row within window; N≤H → all rows, no scroll.
   - long single line > width → wraps into continuation rows; caret tracking
     stays correct across the wrap.
   - NO_COLOR theme → identical row/line structure (only attrs differ).
   Run → red.
2. **GREEN:** implement renderer + viewport minimally to pass.
3. Refactor; split for <200 LOC; ensure word_stream_renderer.go + its
   goldens are byte-unchanged (`git diff` shows no edits there, or only the
   behaviour-neutral shared-styler extraction with goldens still green).
4. `make fmt && make lint && make test-race`. Commit:
   `feat(ui): literal multi-line code renderer with scroll-follow viewport`.

## Success Criteria
- [ ] Literal `\n` → real rows; `\t` → 2-col visual; per-char states correct.
- [ ] Viewport keeps caret visible for top/mid/bottom/short/long; clamps at
  ends; no horizontal scroll (long lines wrap).
- [ ] word_stream_renderer.go untouched (or only neutral styler extraction)
  — its golden tests still green.
- [ ] NO_COLOR layout identical; gofmt/vet/`-race` green; files <200 LOC.
- [ ] One commit.

## Risk Assessment
- Viewport off-by-one at clamps — explicit top/bottom boundary tests.
- Tab-as-2-cols vs caret index drift — assert caret column after a `\t`
  equals prior+2 visually while rune index advances by 1.
- Shared-styler extraction risk — gate on word-stream goldens staying green;
  if private and risky to extract, duplicate the tiny styler instead (KISS >
  forced DRY, same call as the v1.1.0 renderer decision).
- Degraded (terminal <60×20) gate still wins upstream in `model_view.go`
  (unchanged) — renderer assumes it's only called above the safe minimum.
