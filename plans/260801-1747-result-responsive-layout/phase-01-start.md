---
phase: 1
title: "Width Contract And Baseline Snapshots"
status: completed
priority: P1
effort: "2h"
dependencies: []
---

# Phase 1: Width Contract And Baseline Snapshots

## Overview

Establish one shared answer to "how wide is the content allowed to be", and
capture what the screen renders today at each width so every later change is
measured against a recorded baseline rather than a memory of it.

## Requirements

- Functional: a single exported-within-package width helper the panel, hero,
  graph, and stats grid all consult. No component computes its own width policy.
- Non-functional: no visible output change in this phase. The baseline snapshots
  must match `main` exactly.

## Architecture

Today four places make independent, conflicting width decisions:

| Place | Current policy |
|---|---|
| `screen_result_view.go:87` | `panelW := m.w - 8` — grows without bound |
| `screen_result_hero.go:22` | `_ = innerW` — ignores width entirely |
| `result_graph.go:57` | `cols := len(perSec)` — width comes from the data |
| `screen_result_view.go:160` | `colW := innerW / 2` — gutter grows without bound |

That is the actual defect. A 200-column terminal produces a 192-wide panel
holding ~35 columns of content, with the right stats column flung 94 characters
away from the left one.

Introduce one policy:

```go
// resultMaxContentW caps the panel's inner content. Past roughly this width
// a fixed-position two-column layout stops reading as one object: the eye has
// to travel too far between related values. The panel is centred in whatever
// space remains rather than stretched to fill it.
const resultMaxContentW = 88

// resultLayout resolves the panel geometry for a terminal width. Single source
// of truth: the panel, hero, graph and stats grid all size themselves from it,
// so they can never disagree about how much room they have.
type resultLayout struct {
    PanelW  int // bordered panel width
    InnerW  int // content width inside border + padding
    LeftPad int // columns of margin that centre the panel
}

func layoutFor(termW int) resultLayout
```

`88` is a starting value, not a law — Phase 3 checks it against real renders and
the plan records the final number.

## Related Code Files

- Create: `internal/ui/result_layout.go`, `internal/ui/result_layout_test.go`
- Create: `internal/ui/testdata/result_baseline_*.txt` (snapshots)
- Modify: `internal/ui/screen_result_view.go` (derive `panelW`/`innerW` from `layoutFor`, behavior unchanged this phase)

## Implementation Steps

1. Add `result_layout.go` with `resultLayout`, `layoutFor`, and the cap constant.
   In this phase `layoutFor` reproduces today's arithmetic exactly
   (`PanelW = termW - 8`, floor 40, `LeftPad = 0`) so nothing moves yet.
2. Route `renderPanel` through `layoutFor`; confirm output is byte-identical.
3. Add a settled-frame snapshot helper and record baselines at **60, 80, 120,
   200** columns, with and without `NO_COLOR`. 60 is the degraded-gate boundary;
   200 is the reported terminal.
4. Table-driven test asserting `layoutFor` is monotonic and never returns
   `InnerW < 1` for any width from 1 to 400 — the floor must hold for absurd
   inputs, not just plausible ones.

## Success Criteria

- [x] `layoutFor` is the only place a width policy is decided
- [x] Baselines recorded for 4 widths × 2 color modes
- [x] Rendered output byte-identical to `main` at every recorded width
- [x] `go test ./internal/ui/ -race -count=1` green

**Discovered here:** `innerW` was `panelW - 4`, but border (2) + padding (4) is
6. Latent because no section had ever used its full width; phase 2 exposed it
the moment the chart tried to.

## Risk Assessment

**Risk:** snapshots bake in current defects, so later phases "pass" by matching
something wrong.
**Mitigation:** the baselines exist to make the *diff* reviewable, not to assert
correctness. Phase 3 replaces them and the diff is read by a human.
