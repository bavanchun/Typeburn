---
phase: 1
title: "Release-Prep PR"
status: pending
priority: P1
effort: "1h"
dependencies: []
---

# Phase 1: Release-Prep PR

## Overview

One small PR that rolls the CHANGELOG to 2.6.0, syncs the release-notes
extract and roadmap, and sweeps in the stray v2.5.0 journal. No code changes.

## Requirements

- Functional: CHANGELOG section `[2.6.0] - <today>` replaces `[Unreleased]`
  (keep an empty `## [Unreleased]` heading above it, matching the file's
  Keep-a-Changelog convention); `.github/release-notes.md` = verbatim copy of
  the new 2.6.0 section (heading style mirrors the existing 2.5.1 extract);
  `docs/project-roadmap.md` Result-redesign entry updated from "unreleased on
  main" to shipped v2.6.0 with ship date.
- Non-functional: PR-only (protected main); conventional commits, no AI refs;
  docs ≤ 800 LOC each.

## Architecture

Docs-only branch `chore/release-v2.6.0-prep` off fresh `origin/main`. The
CHANGELOG is the authority; release-notes.md is derived from it, roadmap
references it. The untracked `docs/journals/260702-1857-punctuation-numbers-
toggle-feature-shipped.md` (finished docs of shipped v2.5.0 work) rides along
to zero out working-tree dirt.

## Related Code Files

- Modify: `CHANGELOG.md` (roll Unreleased → 2.6.0)
- Modify: `.github/release-notes.md` (replace 2.5.1 extract with 2.6.0)
- Modify: `docs/project-roadmap.md` (entry → shipped v2.6.0)
- Add: `docs/journals/260702-1857-punctuation-numbers-toggle-feature-shipped.md` (already written, untracked)

## Implementation Steps

1. `git checkout main && git pull` → branch `chore/release-v2.6.0-prep`.
2. Roll CHANGELOG; copy the 2.6.0 section verbatim into
   `.github/release-notes.md` (title format: `## [2.6.0] - <date>` + optional
   ` — <short label>` matching 2.5.1 style).
3. Update roadmap entry (shipped v2.6.0 + date). Grep docs for any other
   "unreleased" reference to the redesign.
4. `git add` all four files; commit
   `chore(release): prepare v2.6.0` (+ journal note in body).
5. Push, open PR, wait CI green, squash-merge, pull main.

## Success Criteria

- [x] PR squash-merged; CI green; `git status` clean on updated main.
- [x] `diff` between CHANGELOG 2.6.0 section and release-notes.md content: none
      (modulo the heading-line label convention).
- [x] Roadmap no longer says "unreleased on main".

## Risk Assessment

- **Notes drift** (CHANGELOG vs release-notes.md): copy mechanically, verify
  with a diff before commit — this file becomes the public release body.
- **Another PR merges concurrently:** rebase/retarget before merge; the
  release SHA is decided in Phase 2, not here.
