---
phase: 4
title: "Docs Sync + Full Verify"
status: completed
priority: P2
effort: "0.5d"
dependencies: [1, 2, 3]
---

# Phase 4: Docs Sync + Full Verify

## Overview

Update evergreen docs to the new Result layout and run the full CI-equivalent
gate. Prepare the PR (branch off `main`, conventional commits, no AI refs).
No code changes unless the gate surfaces a regression.

## Requirements

- Functional: `docs/wireframe/mockups.md` §3 Result mockup redrawn to the new
  layout; `docs/project-roadmap.md` gains a release/feature entry; README
  "Result screen" bullet updated if wording drifts.
- Non-functional: docs.maxLoc respected (≤800); links/claims verified against
  source; full gate green.

## Architecture

Docs-only. The Result-screen behavior authority is the code (Phases 1-3); docs
reflect it. Roadmap entry references the new graph + 2-col grid; wireframe §3
mockup updated with the dual-axis graph + hero wpm/acc + stats grid.

## Related Code Files

- Modify: `docs/wireframe/mockups.md` (§3 Result Summary)
- Modify: `docs/project-roadmap.md` (add Result redesign entry to release history / shipped features)
- Maybe modify: `README.md` (Result-screen bullet, line 16: "big-digit WPM, sparkline chart, full char breakdown" → reflect new graph + stats grid)
- Maybe modify: `docs/codebase-summary.md` (if it describes the Result screen rendering)

## Implementation Steps

1. Read current `docs/wireframe/mockups.md` §3; redraw to new layout (hero wpm+acc,
   dual-axis graph, 2-col stats, heatmap).
2. Add roadmap entry (Result-screen redesign); update README Result bullet.
3. Update `codebase-summary.md` Result section if it names the old sparkline.
4. Run full gate: `make test-race`, `make lint` (gofmt + vet + no-TUI guard),
   `make size-check`.
5. Branch `feat/result-ui-redesign` off `main`; conventional commits; push; open PR.
6. Verify CI green before handoff.

## Success Criteria

- [ ] Wireframe §3 matches shipped Result layout.
- [ ] Roadmap + README reflect the new Result screen.
- [ ] `make test-race`, `make lint`, `make size-check` all green.
- [ ] PR opened against `main`; CI green; no AI references in commits.

## Risk Assessment

- **Doc drift:** only update surfaces that actually describe Result rendering;
  avoid churn elsewhere (documentation-management rule: smallest owning surface).
- **Size cap:** new graph file could nudge binary size — `make size-check`
  catches it; mitigate by keeping braille tables minimal.
