---
phase: 2
title: "Hero: Two Big Numbers"
status: completed
priority: P2
effort: "0.5d"
dependencies: []
---

# Phase 2: Hero: Two Big Numbers

<!-- Updated: Validation Session 1 - BigDigitsFixed location corrected to screen_result_reveal.go -->

**Validation lock (S1):** `BigDigits` = `ascii_big_digits.go:101`;
`BigDigitsFixed` = `screen_result_reveal.go:89` (corrected — not in
`ascii_big_digits.go`).

## Overview

Rework `renderHero` so the hero shows **two** big-number blocks side-by-side —
`wpm <N>` (ASCII big-digits) and `acc <N>%` (large styled digits) — matching the
target image. `raw` + `consistency` remain as smaller secondary cards. Reveal
count-up + `★ new best` preserved.

## Requirements

- Functional: hero renders `wpm` big-digits (left) + `acc` big number (right),
  vertically aligned; `raw` and `consistency` as secondary cards (smaller).
  Strict mode uses `KeystrokeAccuracy` for acc (unchanged rule).
- Non-functional: <200 LOC in `screen_result_hero.go`; reveal byte-stable;
  NO_COLOR/mono layout-identical; min-width graceful (60 cols).

## Architecture

Modify `internal/ui/screen_result_hero.go`:
- Keep WPM big-digit block (`BigDigits`/`BigDigitsFixed` + count-up reveal).
- Add an ACC big block: large `fmt.Sprintf("%.0f%%", accVal)` styled
  `RoleAccent`/`accColorRole(accVal)` at a size comparable to WPM digits (use a
  bold large style, not ASCII art — ASCII art for `%` is heavy; a bold enlarged
  number matches Monkeytype's right block).
- Place WPM and ACC blocks side-by-side via `lipgloss.JoinHorizontal`.
- `raw` + `consistency` become a secondary row beneath (smaller stat cards),
  reusing `StatCard` + `revealLine` stagger.
- `★ new best` stays appended to the WPM label.

## Related Code Files

- Modify: `internal/ui/screen_result_hero.go` (64 LOC → ~90-110)
- Modify: `internal/ui/screen_result_test.go` (re-golden hero assertions)
- Read-only refs: `internal/ui/ascii_big_digits.go`, `internal/ui/stat_card.go`, `internal/ui/result_render_helpers.go` (reveal helpers), `internal/theme/*.go`

## Implementation Steps

1. Update hero tests for the new 2-big-number layout (TDD red).
2. Build ACC big block (styled large number + `acc` label).
3. `JoinHorizontal` WPM + ACC; place `raw`/`consistency` secondary row.
4. Preserve count-up reveal for WPM; stagger-fade ACC + secondary cards.
5. Re-golden affected hero snapshots; run `go test ./internal/ui/ -run TestResult -race`.

## Success Criteria

- [ ] Hero shows `wpm` + `acc` as two prominent numbers side-by-side.
- [ ] `raw` + `consistency` preserved as secondary cards.
- [ ] Count-up reveal + `★ new best` intact; settled byte-stable.
- [ ] NO_COLOR/mono layout-identical; `screen_result_hero.go` <200 LOC.
- [ ] Re-goldened hero tests green under `-race`.

## Risk Assessment

- **ACC visual weight vs WPM ASCII art:** WPM ASCII digits are tall; ACC must
  align without dominating. Mitigate by vertical-centering ACC block against the
  WPM digit block with `lipgloss.JoinHorizontal(..., Middle)`.
- **Narrow terminals:** at 60 cols two big blocks may crowd — floor ACC to a
  compact bold number (no ASCII art) so it always fits.
