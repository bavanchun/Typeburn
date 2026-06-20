---
phase: 1
title: Sync Animation Plan Artifacts
status: completed
priority: P1
effort: 1h
dependencies: []
---

# Phase 1: Sync Animation Plan Artifacts

## Overview

Reconcile `plans/260618-1454-ui-ux-animation-system/` with shipped reality. The
top-level phase table says done, but frontmatter and phase checklists still say
pending/TODO.

## Requirements

- Functional: old animation plan reads as completed and traceable to PRs #40-#47.
- Non-functional: preserve historical context; do not rewrite implementation
  decisions or invent new scope.

## Architecture

Use the existing plan directory as the source-of-truth artifact for the shipped
animation work. Keep the new cleanup plan separate so future readers can see why
metadata changed after the implementation PRs.

Prefer `ck plan check` for CLI-managed phase status if it works cleanly in the
old plan directory. Manually patch only fields/checklists that the CLI cannot
sync.

## Related Code Files

- Modify: `plans/260618-1454-ui-ux-animation-system/plan.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-01-anim-core-package.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-02-frame-driver.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-03-caret-animation.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-04-result-reveal.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-05-new-best-celebration.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-06-screen-transitions.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/phase-07-hardening-and-docs.md`
- Modify: `plans/260618-1454-ui-ux-animation-system/handover-phases-04-07-cook-continuation.md`

## Implementation Steps

1. Update old plan frontmatter `status: completed`.
2. Confirm phase table already lists all phases done with PR references.
3. Update each phase file frontmatter `status: completed`.
4. Change success criteria checkboxes from `[ ]` to `[x]` only where verified by
   merged code/tests/docs. If any item is ambiguous, leave unchecked and note why.
5. Update handover file:
   - Mark it as superseded by PRs #44-#47; or
   - Rewrite status table to shipped, preserving useful deviations/gotchas as
     historical notes.
6. Remove stale "remain", "TODO", or "needs user confirmation" text only when
   the current code/docs prove it is resolved.
7. Run `rg -n "status: pending|\\[ \\]|TODO|remain"` inside the animation plan
   directory and review remaining hits.

## Success Criteria

- [x] `plan.md` frontmatter says `status: completed`.
- [x] Phase 1-7 files say `status: completed`.
- [x] No stale unchecked success criteria remain for completed shipped work.
- [x] Handover file cannot be misread as current TODO work.
- [x] Remaining search hits are intentional and documented.

## Risk Assessment

- Risk: marking an unverified item complete because implementation exists nearby.
  Mitigation: cite merged PRs/tests or leave item unchecked with note.
- Risk: losing useful historical context in the handover. Mitigation: mark
  superseded instead of deleting unless content is misleading/noisy.
