---
phase: 7
title: Release v1.2.0
status: completed
priority: P1
effort: 1h
dependencies:
  - 6
---

# Phase 7: Release v1.2.0

## Overview
Ship v1.2.0 (minor — new feature) via the proven PR-based, fix-forward
runbook. Protected-main: PR → squash-merge → disposable dry-run → annotated
tag on the merged SHA.

## Requirements
- Functional: GitHub Release `v1.2.0` with 7 assets (6 archives +
  `checksums.txt`) and curated notes (non-empty body).
- Non-functional: never delete-and-re-tag a pushed version; tag pushed in a
  SEPARATE push (never `--follow-tags`).

## Architecture
Notes via `CHANGELOG.md [Unreleased]`→`[1.2.0]` extracted to
`.github/release-notes.md` (`--release-notes`). `.goreleaser.yaml` keeps the
`changelog.filters.exclude:['.*']` filter (NOT `disable:true`). `ci.yml`
does not run on tags → `release.yml` self-gates. GoReleaser pinned
`v2.15.4` (lockstep — unchanged).

## Related Code Files
- Modify: `CHANGELOG.md` (author `[Unreleased]` then promote → `[1.2.0] -
  <date>` + link refs), `.github/release-notes.md` (mirror the `[1.2.0]`
  section), `docs/project-roadmap.md` (Code mode → shipped v1.2.0; update
  Conclusion to v1.2.0), `README.md` (document `--text` usage + Code mode),
  `docs/design-guidelines.md` / `docs/system-architecture.md` (Code renderer
  + codetext package + ModeCode), `CLAUDE.md` (note ModeCode/codetext in the
  architecture section)
- Create/Delete: none (version is ldflags/tag injected — no constant)

## Implementation Steps
1. Author `CHANGELOG.md [Unreleased]` (Added: Code mode + `--text`; note
   history-but-not-★; out-of-scope deferred items) — on the feature branch
   during Phase 5/6 ideally; here promote `[Unreleased]`→`[1.2.0] - <date>`,
   add link refs, regenerate `.github/release-notes.md`. Update README +
   docs. Commit: `docs: changelog + release notes + docs for v1.2.0`.
2. Push branch; `gh pr create` → `main` (no AI refs). Wait `ci.yml` green.
3. **Squash-merge**; record merged `main` SHA.
4. Disposable dry-run: `git tag v0.0.0-rc.test <SHA>` → push → watch
   `release.yml` → verify **7 assets** + non-empty notes + checksums →
   `gh release delete v0.0.0-rc.test --yes --cleanup-tag`.
5. Real tag: `git tag -a v1.2.0 <same SHA> -m "Typeburn v1.2.0"` →
   `git push origin v1.2.0` (separate push, NEVER `--follow-tags`).
6. Verify published `v1.2.0` (7 assets, notes, not draft/pre). Land any
   leftover plan-status/reports + journal via a `chore/...` PR (separate
   `git switch -c` step — hook evaluates HEAD pre-exec).
7. `/ck:journal`.

## Success Criteria
- [ ] PR squash-merged, `ci.yml` green.
- [ ] Dry-run proved 7 assets + non-empty notes, then deleted.
- [ ] `v1.2.0` on the dry-run-proven SHA; pushed separately.
- [ ] Release live (7 assets, curated notes, not draft/pre).
- [ ] Roadmap/README/docs reflect v1.2.0 + Code mode.

## Risk Assessment
- Empty release body — keep exclude-all changelog filter; dry-run asserts
  non-empty before the real tag.
- Re-tag temptation on defect — forbidden; fix forward to v1.2.1.
- `--follow-tags` accident — push branch and tag in distinct commands.
- `ck plan check` dirties tracked plan files mid-release — stash before
  switching to main; finalize via the chore PR (known from v1.1.0).
