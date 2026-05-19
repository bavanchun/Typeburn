---
phase: 6
title: "Release v1.3.0"
status: completed
priority: P1
effort: "1h"
dependencies: [5]
---

# Phase 6: Release v1.3.0

## Overview
Ship v1.3.0 (minor — new feature) via the proven PR-based, fix-forward
runbook. Protected-main: PR → squash-merge → disposable dry-run → annotated
tag on the merged SHA.

## Requirements
- Functional: GitHub Release `v1.3.0` with 7 assets (6 archives +
  `checksums.txt`) + curated non-empty notes.
- Non-functional: never delete-and-re-tag a pushed version; tag pushed in a
  SEPARATE push (never `--follow-tags`).

## Architecture
Notes via `CHANGELOG.md [Unreleased]`→`[1.3.0]` extracted to
`.github/release-notes.md` (`--release-notes`). `.goreleaser.yaml` keeps the
`changelog.filters.exclude:['.*']` filter (NOT `disable:true`). `ci.yml`
does not run on tags → `release.yml` self-gates. GoReleaser pinned
`v2.15.4` (lockstep — unchanged).

## Related Code Files
- Modify: `CHANGELOG.md` (author `[Unreleased]` then promote → `[1.3.0] -
  <date>` + link refs), `.github/release-notes.md` (mirror `[1.3.0]`),
  `README.md` (Code mode now also via in-app paste — update the `--text`
  prose + keymap/usage), `docs/project-roadmap.md` (in-app paste → shipped
  v1.3.0; Conclusion → v1.3.0), `docs/system-architecture.md` /
  `docs/design-guidelines.md` (ScreenCodePaste + codetext.Normalize),
  `CLAUDE.md` (note ScreenCodePaste + that codetext exports Normalize)
- Create/Delete: none (version is ldflags/tag injected)

## Implementation Steps
1. Author `CHANGELOG.md [Unreleased]` (Added: in-app paste for Code mode —
   `ScreenCodePaste`, reached from the Code row when no `--text`; same
   normalization/caps; `--text` still supported) during P4/P5; here promote
   `[Unreleased]`→`[1.3.0] - <date>`, add link refs, regenerate
   `.github/release-notes.md`. Update README + docs + CLAUDE.md. Commit:
   `docs: changelog + release notes + docs for v1.3.0`.
2. Push branch; `gh pr create` → `main` (no AI refs). Wait `ci.yml` green.
3. **Squash-merge**; record merged `main` SHA.
4. Disposable dry-run: `git tag v0.0.0-rc.test <SHA>` → push → watch
   `release.yml` → verify **7 assets** + non-empty notes + checksums →
   `gh release delete v0.0.0-rc.test --yes --cleanup-tag`.
5. Real tag: `git tag -a v1.3.0 <same SHA> -m "Typeburn v1.3.0"` →
   `git push origin v1.3.0` (separate push, NEVER `--follow-tags`).
6. Verify published `v1.3.0` (7 assets, notes, not draft/pre). Land leftover
   plan-status/reports + journal via a `chore/...` PR (separate `git switch
   -c` step — hook evaluates HEAD pre-exec).
7. `/ck:journal`; update the `typeburn-release-runbook` memory with the
   v1.3.0 outcome.

## Success Criteria
- [ ] PR squash-merged, `ci.yml` green.
- [ ] Dry-run proved 7 assets + non-empty notes, then deleted.
- [ ] `v1.3.0` on the dry-run-proven SHA; pushed separately.
- [ ] Release live (7 assets, curated notes, not draft/pre).
- [ ] README/roadmap/docs/CLAUDE.md reflect v1.3.0 + in-app paste.

## Risk Assessment
- Empty release body — keep exclude-all changelog filter; dry-run asserts
  non-empty first.
- Re-tag temptation on defect — forbidden; fix forward to v1.3.1.
- `--follow-tags` accident — push branch and tag in distinct commands.
- `ck plan check` dirties tracked plan files mid-release — stash before
  switching to main; finalize via the chore PR (known v1.1.0/v1.2.0).
