---
phase: 2
title: Sync Evergreen Documentation
status: completed
priority: P1
effort: 1h
dependencies:
  - 1
---

# Phase 2: Sync Evergreen Documentation

## Overview

Fix project docs that still describe pre-animation behavior or incomplete PR
ranges. Keep docs concise and consistent with the shipped implementation.

## Requirements

- Functional: docs describe current motion behavior accurately.
- Non-functional: no code comments or docs should refer to volatile phase/finding
  labels as the reason for behavior; explain stable behavior/invariants instead.

## Architecture

Evergreen docs are the source of truth for future contributors. Update the small
set of files that directly mention motion, NO_COLOR behavior, or animation PR
status. Avoid broad documentation rewrites.

## Related Code Files

- Modify: `docs/design-guidelines.md`
- Modify: `README.md`
- Modify: `docs/project-roadmap.md`
- Inspect/modify if needed: `docs/system-architecture.md`
- Inspect/modify if needed: `docs/codebase-summary.md`

## Implementation Steps

1. Update `docs/design-guidelines.md` motion policy:
   - Replace "screen transitions: hard cut" with shipped policy.
   - Mention Typing->Result gets a short transition.
   - Mention NO_COLOR uses row wipe / attribute-only behavior.
   - Remove stale "already static" reduced-motion wording if it contradicts
     always-on animation.
2. Update `README.md` feature bullet if needed so it includes the Typing->Result
   transition, not only caret/reveal/celebration.
3. Update `docs/project-roadmap.md` PR range from `#40-#46` to `#40-#47`.
4. Inspect `docs/system-architecture.md` and `docs/codebase-summary.md`; patch
   only if they conflict with current code or this cleanup.
5. Search docs for stale motion statements:
   `rg -n "hard cut|no motion|near-static|#40.*#46|TODO|remain" docs README.md`.

## Success Criteria

- [x] `docs/design-guidelines.md` matches shipped animation behavior.
- [x] README feature list mentions transition or deliberately omits it with no
      contradiction.
- [x] Roadmap status references PRs #40-#47.
- [x] `docs/system-architecture.md` and `docs/codebase-summary.md` remain
      consistent after inspection.
- [x] No stale motion-policy contradictions remain in docs.

## Risk Assessment

- Risk: over-editing design docs beyond the requested drift. Mitigation: patch
  only paragraphs that contradict shipped behavior.
- Risk: docs claim more than code supports. Mitigation: verify against
  `internal/app/transition.go`, `internal/ui/frame_tick.go`, and existing tests.
