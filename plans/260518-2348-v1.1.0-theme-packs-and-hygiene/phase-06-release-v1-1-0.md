---
phase: 6
title: "Release v1.1.0"
status: pending
priority: P1
effort: "1h"
dependencies: [5]
---

# Phase 6: Release v1.1.0

## Overview
Ship v1.1.0 via the PR-based, fix-forward release runbook. Protected-main:
PR → squash-merge → disposable dry-run → annotate + push the real tag.

## Requirements
- Functional: GitHub Release `v1.1.0` published with 7 assets
  (6 archives + `checksums.txt`) and curated notes (non-empty body).
- Non-functional: never delete-and-re-tag a pushed version (sumdb
  append-only); tag pushed in a separate push (never `--follow-tags`).

## Architecture
Release notes come from `CHANGELOG.md` → extracted to
`.github/release-notes.md`, passed to GoReleaser `--release-notes`.
`.goreleaser.yaml` keeps the `changelog.filters.exclude: ['.*']` filter
(NOT `disable: true` — that silences `--release-notes`). `ci.yml` does NOT
run on tags; `release.yml` self-gates with its own test job. GoReleaser
pinned `v2.15.4` (lockstep: `.goreleaser.yaml`, `release.yml`,
`CONTRIBUTING.md`) — unchanged this release.

## Related Code Files
- Modify: `CHANGELOG.md` (**promote** the existing `[Unreleased]` block
  authored by Phase 4 → `[1.1.0] - 2026-05-18`; do not re-author content),
  `.github/release-notes.md` (mirror the `[1.1.0]` body),
  `docs/project-roadmap.md` (mark v1.1.0 shipped — if not already in phase 4)
- Create / Delete: none (no version constant — version is ldflags/tag
  injected per CONTRIBUTING)

## Implementation Steps
1. On the feature branch, rename `CHANGELOG.md` `[Unreleased]` (authored in
   Phase 4) → `[1.1.0] - 2026-05-18`; regenerate `.github/release-notes.md`
   from that section. Commit: `docs: changelog + release notes for v1.1.0`.
2. Push branch: `git push -u origin feat/v1.1.0-theme-packs-hygiene`
   (hook allows non-main branch push).
3. Open PR → `main`: `gh pr create` (no AI references in title/body).
   Wait for `ci.yml` green (required check).
4. **Squash-merge** the PR (squash is the only enabled mode; branch
   auto-deletes). Record the merged `main` SHA.
5. Disposable dry-run: `git tag v0.0.0-rc.test <merged-SHA>` →
   `git push origin v0.0.0-rc.test` → watch `release.yml` → verify **7
   assets**, non-empty notes, checksums cross-check →
   `gh release delete v0.0.0-rc.test --yes --cleanup-tag`.
6. Real tag on the **same** SHA: `git tag -a v1.1.0 <merged-SHA>
   -m "Typeburn v1.1.0"` → `git push origin v1.1.0` (separate push, NEVER
   `--follow-tags`).
7. Verify the published `v1.1.0` release: 7 assets, notes, checksums; `go
   install …@v1.1.0` resolves (allow ~1h proxy lag).
8. Run `/ck:journal` for the release entry.

## Success Criteria
- [ ] PR squash-merged into `main` with `ci.yml` green.
- [ ] Disposable dry-run proved 7 assets + non-empty notes, then deleted.
- [ ] `v1.1.0` tag on the dry-run-proven SHA; pushed separately.
- [ ] GitHub Release `v1.1.0` live with 7 assets + curated notes.
- [ ] Roadmap marks v1.1.0 shipped.

## Risk Assessment
- **Empty release body:** mitigated — keep exclude-all changelog filter;
  dry-run asserts non-empty before the real tag.
- **Re-tag temptation on defect:** forbidden — fix forward to v1.1.1. Only
  `v0.0.0-rc.test` may be deleted/re-pushed.
- **`--follow-tags` accident:** push branch and tag in distinct commands;
  the protect-main hook allows tag pushes but not main branch pushes.
- **Asset count drift:** `release.yml` asserts expected=7; investigate before
  the real tag if it differs.
