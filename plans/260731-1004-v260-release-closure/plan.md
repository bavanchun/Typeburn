---
title: "v2.6.0 Release Closure"
description: >-
  Close the Result-redesign delivery loop: release-prep PR (CHANGELOG roll,
  release-notes extract, roadmap sync, stray journal), disposable dry-run,
  immutable v2.6.0 tag, post-release verification, and feedback-loop kickoff.
status: completed
priority: P1
effort: "0.5d"
tags: [release, process]
blockedBy: []
blocks: []
created: 2026-07-31
---

# v2.6.0 Release Closure

## Overview

The Monkeytype-style Result redesign (PR #61, squash `1f77848c`) is merged to
`main` but unreleased — CHANGELOG holds it under `[Unreleased]`, latest tag is
`v2.5.1`. This plan cuts `v2.6.0` (minor: user-visible feature, no breaking
change) following the CONTRIBUTING runbook exactly, then verifies install
channels and opens the feedback loop the roadmap calls for.

**Hard invariants (from CLAUDE.md / CONTRIBUTING):**
- `main` protected: every file change via branch → PR → squash-merge.
- Tags cut on `main` only after merge; tag pushed **separately** (never
  `--follow-tags`); tags immutable — defects fix-forward as v2.6.1.
- Only `v0.0.0-rc.test` may ever be deleted (`gh release delete --cleanup-tag`).
- Expected published assets = 7 (6 archives + `checksums.txt`); `release.yml`
  asserts this.
- `.github/release-notes.md` = verbatim extract of the latest CHANGELOG
  section (GoReleaser `--release-notes` reads it; `changelog.filters.exclude`
  stays as-is — never "simplify" to `disable: true`).

## Goals

| # | Goal | Priority |
|---|------|----------|
| 1 | Release-prep PR merged (CHANGELOG 2.6.0 + release-notes + roadmap + stray journal) | P1 |
| 2 | v2.6.0 published with 7 verified assets on the dry-run-proven SHA | P1 |
| 3 | Install channels verified; feedback-loop next-step recorded in roadmap | P2 |

## Phases

| # | Phase | Status |
|---|-------|--------|
| 1 | [Release-Prep PR](./phase-01-start.md) | Completed |
| 2 | [Release Dry-Run + Tag v2.6.0](./phase-02-release-dry-run-tag-v260.md) | Completed |
| 3 | [Post-Release Verify + Feedback Loop Kickoff](./phase-03-post-release-verify-feedback-loop-kickoff.md) | Completed |

## Success Criteria

- [x] CHANGELOG `[Unreleased]` → `[2.6.0] - <date>`; `.github/release-notes.md`
      matches it verbatim; roadmap entry says shipped v2.6.0; stray v2.5.0
      journal committed. All via one squash-merged PR, CI green.
- [x] Dry-run `v0.0.0-rc.test` publishes 7 assets with correct notes, then is
      fully deleted (release + tag).
- [x] Annotated `v2.6.0` on the exact dry-run SHA; `release.yml` green;
      7 assets + checksums verified; release NOT marked prerelease.
- [x] `typeburn update --check` from v2.5.1 sees v2.6.0; `go install
      ...@v2.6.0` succeeds (allow ~1 h proxy lag before declaring failure).
- [x] No local dirty state; plan archived via journal.

## Unresolved Questions

None — procedure is fully documented in CONTRIBUTING.md; this plan only
sequences it.

<!-- slug: v260-release-closure -->
