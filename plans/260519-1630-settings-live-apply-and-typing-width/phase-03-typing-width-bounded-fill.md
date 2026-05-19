---
phase: 3
title: Typing Width Bounded Fill
status: completed
priority: P2
effort: 3h
dependencies:
  - 2
---

# Phase 3: Typing Width Bounded Fill

## Overview

Make the typing stream use ~82% of wide terminals (never narrower than the
old 80) and sit vertically centered, so text no longer feels lost in
whitespace on large screens.

## Context Links
- [brainstorm-summary.md](./brainstorm-summary.md)
- `internal/ui/width_tier.go:41-59` (`ContentWidth`, TierWide branch)
- `internal/ui/screen_typing_view.go:66-85` (spacer fill + JoinVertical)
- `internal/app/model_view.go:50-53` (root `lipgloss.Place(Center,Center)` for Typing)
- `internal/ui/phase09_polish_test.go:40-55` (locks `ContentWidth(100,TierWide)==80`)
- `internal/ui/screen_typing_test.go` (typing view assertions)

## Key Insights
- TierWide is hard `return 80`. `WidthTier` classification is correct and
  must NOT change; only the wide **content width value** changes.
- `0.82*88 ≈ 72 < 80`: a naive % would shrink the stream at the wide
  boundary. Contract MUST floor at 80 (monotonic non-decreasing, never
  regress) and cap at `termW-8` for breathing room + centering.
- Root already wraps Typing in `lipgloss.Place(m.w,m.h,Center,Center)`, but
  `screen_typing_view.go` pads the block to full `m.h` (spacer = `m.h-used`)
  → Place's vertical centering is a no-op. Removing the height-fill makes a
  compact block that Place genuinely centers. Footer stops being
  terminal-bottom-pinned (intended UX change).
- Width=80 in many existing tests = TierMid (`72≤w<88`) → ContentWidth
  `80-8=72`, UNAFFECTED. Only TierWide (`w≥88`) value changes.

## Requirements
- Functional: `ContentWidth(termW, TierWide) = clamp(round(termW*0.82), 80, termW-8)`.
  Examples: `(88)→80`, `(100)→82`, `(120)→98`, `(160)→131`, `(200)→164`.
- Functional: typing screen renders a compact block centered vertically &
  horizontally by the root Place; no full-height spacer.
- Non-functional: Narrow/Mid/Degraded tiers, `WidthTier`, and all non-typing
  screens byte-unchanged. Files <200 LOC. No plan-artifact refs.

## Architecture

`ContentWidth` TierWide branch:
```go
case TierWide:
    w := int(math.Round(float64(termW) * 0.82))
    if w < 80 { w = 80 }            // never regress below the old cap
    if w > termW-8 { w = termW-8 }  // breathing room; keeps centering sane
    return w
```
`screen_typing_view.go`: drop the `spacerLines = m.h - used` fill; join
`header, "", stream, "", footer` (small fixed gaps) and let root
`model_view.go` Place center it. Keep `fixedOverhead` semantics for the Code
stream height calc consistent (Code path uses `m.h - fixedOverhead`; verify
it still bounds correctly without the trailing spacer).

## Related Code Files
- Modify: `internal/ui/width_tier.go` (TierWide branch; add `math` import)
- Modify: `internal/ui/screen_typing_view.go` (remove height-fill spacer)
- Modify: `internal/ui/phase09_polish_test.go` (rewrite TierWide ContentWidth
  assertions to the new contract — **intentional**, documented)
- Modify: `internal/ui/screen_typing_test.go` (update any layout/line-position
  or footer-on-last-line assertions to the centered-compact contract)
- Modify (if asserting old layout): `internal/app/phase09_polish_test.go`,
  `internal/app/smoke_test.go` typing-frame checks

## Implementation Steps (TDD)

### RED
1. Rewrite `phase09_polish_test.go` `TestContentWidth_PerTier` TierWide cases
   to the new contract with explicit cases: `88→80, 100→82, 120→98,
   200→164`; keep Mid/Narrow/Degraded cases unchanged. Add a monotonic /
   never-below-80 assertion. These FAIL on current `return 80`.
2. Add a typing-centering test: at a large size (e.g. `SetSize(160,50)`)
   assert the rendered frame has roughly balanced top/bottom blank padding
   (stream block not top-anchored) and the stream line width tracks the new
   ContentWidth. Assert non-typing screens unchanged at same size.
3. Run UI/app tests → new assertions RED; confirm Mid-width (80) typing
   tests still GREEN (proves no collateral on the common path).

### GREEN
4. Implement the `ContentWidth` TierWide clamp (add `math` import).
5. Remove the full-height spacer in `screen_typing_view.go`; emit compact
   block; verify Code-mode stream-height calc still correct.
6. Run failing layout/footer assertions in `screen_typing_test.go` /
   `phase09_polish_test.go` (app); update each to the centered contract —
   each change annotated with the behavioural reason (not "phase 3"/plan
   ref). This is the deliberate regression-lock rewrite called out in plan.md.
7. `gofmt`, `go vet`, `go test ./... -race -count=1` GREEN. Commit.

## Success Criteria
- [ ] `ContentWidth` TierWide matches the clamp contract for all example sizes
- [ ] TierWide never returns < 80; Mid/Narrow/Degraded + `WidthTier` unchanged
- [ ] Typing frame vertically centered at large sizes; non-typing screens unchanged
- [ ] Rewritten regression assertions GREEN and reflect the NEW contract (no bypass)
- [ ] Full `-race` suite GREEN; files <200 LOC; no plan-artifact refs

## Risk Assessment
- Unknown tests asserting exact typing line counts → TDD surfaces them; update
  intentionally with behavioural rationale, never skip/xfail.
- Code-mode stream height regression from spacer removal → explicit verify
  step + a Code-mode render test at large size.
- Rounding off-by-one disputes → contract pinned by explicit example cases.

## Security Considerations
None — pure presentational layout.

## Next Steps
Phase 4 (integration verify) with both fixes on the branch.
