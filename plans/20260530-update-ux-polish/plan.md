---
title: Update-UX Polish (post-v2.3.0)
description: ''
status: completed
priority: P2
branch: feat/update-ux-polish
tags: []
blockedBy: []
blocks: []
created: '2026-05-29T19:45:40.961Z'
createdBy: 'ck:plan'
source: skill
---

# Update-UX Polish (post-v2.3.0)

## Overview

Three UX fixes for the `typeburn update` self-updater shipped in v2.3.0:

1. **Defect** — the in-app result-screen hint points at the check-only
   `typeburn version --check-update`, a dead end that never upgrades. Point it at
   `typeburn update`. (This stale string is why the feature went undiscovered.)
2. **Polish** — `typeburn update` blocks silently during the ~9 MB
   download/verify/swap. Emit `downloading… / verifying… / installing…` step
   lines via a nil-safe reporter threaded through `update.Apply`.
3. **Polish** — surface `Release notes: <url>` in `update` output, mirroring the
   wording `version --check-update` already uses (`Result.ReleaseURL` exists and
   is repo-guarded — no new fetching).

Design source: [brainstorm-summary.md](./brainstorm-summary.md).

**TDD:** each phase writes/updates tests first (red), then implements to green.
Existing update tests (httptest fixtures, string assertions) lock current
behavior before the changes land.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Fix in-app update hint](./phase-01-fix-in-app-update-hint.md) | Completed |
| 2 | [Update progress feedback](./phase-02-update-progress-feedback.md) | Completed |
| 3 | [Surface release URL + docs](./phase-03-surface-release-url-docs.md) | Completed |

## Key Constraints (from CLAUDE.md)

- **Layering:** no UI deps in `internal/update`; reporter is a plain stdlib
  `func(Stage)`. CLI step-printing stays in `internal/cli`.
- **File size:** every Go file <200 LOC; `cmd_update.go` is at 161 — watch the cap
  in Phase 2/3 (split a helper out if it would cross 200).
- **Themes:** NO_COLOR + mono layouts identical; Phase 1's new hint is *shorter*
  than the old string, so width-capping stays safe — assert mono unchanged.
- **Trust model unchanged:** checksum-only, unsigned. Do not touch verify/replace.
- **Protected main:** branch `feat/update-ux-polish` → PR → squash-merge.

## Dependencies

- **Builds on (completed):** `plans/20260529-typeburn-self-update-command` —
  shipped `internal/update` Apply/download/verify + `cmd_update.go` (v2.3.0).
  Foundation merged; no blocking relationship.
- No open plans block or are blocked by this one (all prior plans completed).
