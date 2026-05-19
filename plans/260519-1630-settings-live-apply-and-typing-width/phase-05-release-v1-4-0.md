---
phase: 5
title: Release v1.4.0
status: completed
priority: P1
effort: 1.5h
dependencies:
  - 4
---

# Phase 5: Release v1.4.0

## Overview

Ship the two fixes via the proven protected-main PR + dry-run + tag runbook.

## Context Links
- Memory: `typeburn-release-runbook` (PR-based; GoReleaser v2.15.4 pinned;
  release.yml self-gates; tag pushes not branch-protection-blocked).

## Key Insights
- Version decision: **v1.4.0** recommended — a user-visible non-breaking
  behaviour/UX change (typing layout) beyond the pure live-apply bug fix.
  **Confirm with the user at this phase** whether they prefer v1.3.1 (treat
  as bug-fix-only patch). This is the one open question from brainstorm.
- Never re-tag a shipped version; fix-forward only. `changelog.disable:true`
  forbidden in `.goreleaser.yaml`. Push branch and tag in distinct commands
  (never `--follow-tags`). Prefix every git/gh call with `cd <repo>`.

## Requirements
- Functional: GitHub Release for the chosen tag with 7 assets + non-empty
  curated notes, not draft/pre.
- Non-functional: `main` left clean; plan/report/journal landed via a
  separate `chore/...` PR (cannot commit to `main`).

## Related Code Files
- Modify: `CHANGELOG.md` (new version block: Fixed = settings live-apply;
  Changed = typing width/centering)
- Modify: `.github/release-notes.md` (mirror the version block verbatim)
- Modify: `internal/version/version.go` if version is embedded there
- Modify: `README.md`, `docs/project-roadmap.md` (version bump + behaviour note)

## Implementation Steps
1. **Confirm version** with the user (v1.4.0 vs v1.3.1) via `AskUserQuestion`
   before writing CHANGELOG/tag.
2. Update `CHANGELOG.md` + `.github/release-notes.md` (mirrored, verbatim) +
   README/roadmap. Commit on the feature branch.
3. Push branch → open PR → `ci.yml` must pass green.
4. Squash-merge PR into `main`; capture merged SHA.
5. Disposable dry-run: `git tag v0.0.0-rc.test <SHA>` → push → `release.yml`
   → verify 7 assets + non-empty notes + checksum cross-check →
   `gh release delete v0.0.0-rc.test --yes --cleanup-tag`.
6. `git tag -a vX.Y.Z <same SHA> -m "…"` → `git push origin vX.Y.Z`
   (separate push, NEVER `--follow-tags`).
7. Verify the GitHub Release (7 assets, curated notes, not draft/pre).
8. Land finalized plan + Phase 4 reports + journal via a separate
   `chore/...` PR (stash `plans/` before switching if `ck plan check`
   dirtied it). Update `typeburn-release-runbook` memory with the outcome
   + any new gotchas. Clean local branches.

## Success Criteria
- [ ] Version confirmed with user
- [ ] CI green; PR squash-merged to `main`
- [ ] Dry-run verified (7 assets + notes) then deleted
- [ ] Real tag on merged SHA; GitHub Release live, not draft/pre
- [ ] `main` clean; chore PR landed; runbook memory updated

## Risk Assessment
- `ck plan check` rewrites tracked `plans/**` frontmatter → stash before
  `git switch main`; land via chore PR.
- protect-main hook denies chained `switch -c && commit` → separate calls.
- gh/git cwd reset between Bash calls → `cd <repo>` prefix every call.

## Next Steps
`/ck:journal` — concise technical journal entry; then plan archive.
