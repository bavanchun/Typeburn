---
phase: 2
title: "Result Screen Surfacing"
status: completed
priority: P2
effort: "1.5h"
dependencies: [1]
---

# Phase 2: Result Screen Surfacing

## Overview

Render the top missed keys on the Result screen, below the correct/incorrect/
extra char-stats line. One compact line; clean runs show a faint "no missed keys".

## Requirements

- **Functional:**
  - New `renderKeyHeatmap()` produces a line like:
    `most missed:  e ×4   t ×3   a ×2   ␣ ×2   r ×1` (top 8, fit to width).
  - Clean run (`len(KeyMisses)==0`) → faint `no missed keys`.
  - Inserted into the result panel after `renderCharStats()` (before `renderMeta`).
- **Non-functional:**
  - Theme roles only — no hex. Mono/NO_COLOR safe (layout identical, attrs only).
  - `screen_result_view.go` stays < 200 LOC → put the new helper in the existing
    `internal/ui/result_render_helpers.go`.
  - Width-aware: truncate the key list to `innerW`; never overflow the panel.

## Architecture

`renderPanel()` (`screen_result_view.go:80-108`) currently assembles:
`hero → sparkline → charStats → meta`. Insert heatmap between charStats and meta:

```go
inner.WriteString(m.renderCharStats())
inner.WriteString("\n")
inner.WriteString(m.renderKeyHeatmap(innerW))   // NEW
inner.WriteString("\n")
inner.WriteString(m.renderMeta())
```

`renderKeyHeatmap(innerW int) string` lives in `result_render_helpers.go`
(method on `ResultModel`, same receiver as the existing helpers there). Reads
`m.res.KeyMisses`.

Styling (theme roles, consistent with `renderCharStats`):
- label `most missed:` → `RoleTextMuted`
- key glyph → `RoleTextPrimary`
- `×N` count → `RoleError`
- clean-run text → `RoleTextFaint`

Width handling: build entries left-to-right, stop appending once the visible
(ANSI-stripped) width would exceed `innerW`. Reuse the same width discipline the
file already uses; if a `lipgloss.Width` measure is needed, follow the existing
pattern in this package (do not hand-count runes against ANSI).

## Related Code Files

- Modify: `internal/ui/result_render_helpers.go` (add `renderKeyHeatmap`)
- Modify: `internal/ui/screen_result_view.go` (wire 2 lines into `renderPanel`)
- Modify: `internal/ui/result_render_helpers_test.go` and/or
  `internal/ui/screen_result_test.go` (substring assertions — NOT golden files;
  Result view tests use `strings.Contains`, confirmed)

## Implementation Steps

1. Add `renderKeyHeatmap(innerW int) string` to `result_render_helpers.go`:
   - If `len(m.res.KeyMisses) == 0` → return faint `no missed keys`.
   - Else build `most missed:  <k ×n>...` up to 8 entries, width-capped to `innerW`.
2. Wire it into `renderPanel()` in `screen_result_view.go` (2 inserted lines).
3. Tests:
   - With misses: view contains `most missed` and at least the top key + `×`.
   - Clean run: view contains `no missed keys`.
   - Mono theme + `NO_COLOR`: assert the heatmap text still present and that
     overall line count / layout is unchanged vs the colored render (attribute-
     only invariant — mirror how existing theme tests assert this).
4. Run `go test ./internal/ui/ -race -count=1`.

## Success Criteria

- [ ] Result screen shows top missed keys (or "no missed keys" on clean runs).
- [ ] Mono / NO_COLOR render identical layout (attributes only).
- [ ] No panel overflow at min terminal width (60 cols).
- [ ] `screen_result_view.go` still < 200 LOC.
- [ ] `go test ./internal/ui/ -race -count=1` GREEN; `gofmt`/`vet` clean.

## Risk Assessment

- **Low-medium.** Main risk is width math causing overflow at narrow widths —
  mitigated by truncation capped at `innerW` and a 60-col test.
- No golden-file regeneration needed (substring-based tests confirmed), which
  removes the usual teatest churn risk for this screen.
- Vertical space: +1 line; footer still pins to bottom via existing `spacer` math.
