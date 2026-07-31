---
phase: 2
title: "Release Dry-Run + Tag v2.6.0"
status: pending
priority: P1
effort: "1-2h"
dependencies: [1]
---

# Phase 2: Release Dry-Run + Tag v2.6.0

## Overview

Prove the release pipeline on a disposable tag, then cut the immutable
`v2.6.0` on the exact proven SHA. Follows CONTRIBUTING.md §release runbook
verbatim — this phase adds no new procedure.

## Requirements

- Functional: `release.yml` publishes 7 assets (6 archives + checksums.txt)
  with the 2.6.0 notes from `.github/release-notes.md`; final release is a
  normal (non-prerelease) release on the dry-run-proven SHA.
- Non-functional: tag pushed in a separate push (never `--follow-tags`);
  `v2.6.0` never deleted/re-tagged regardless of outcome (fix-forward only);
  only `v0.0.0-rc.test` is deletable.

## Architecture

`ci.yml` does not run on tag pushes — `release.yml`'s own least-privilege
`test` job is the only CI on the tagged commit; it gates the publish job.
GoReleaser pinned `v2.15.4` (three lockstep places — do not touch). The
dry-run and the real tag MUST point at the same SHA so the proven pipeline
state equals the released state.

## Related Code Files

- None modified. Git tags + GitHub releases only.

## Implementation Steps

1. `git checkout main && git pull`; record `RELEASE_SHA=$(git rev-parse HEAD)`
   (must be the Phase-1 merge commit or newer — confirm nothing unexpected
   landed).
2. Dry-run: `git tag v0.0.0-rc.test $RELEASE_SHA && git push origin v0.0.0-rc.test`.
3. Watch `release.yml` run to green (`gh run watch`). Verify: 7 assets,
   checksums.txt entries match archives, release body = 2.6.0 notes,
   flagged prerelease (rc tag → `prerelease: auto`).
4. Delete dry-run completely: `gh release delete v0.0.0-rc.test --cleanup-tag --yes`.
5. Real tag on the SAME SHA:
   `git tag -a v2.6.0 -m "v2.6.0" $RELEASE_SHA && git push origin v2.6.0`.
6. Watch `release.yml` to green; verify 7 assets + notes + NOT prerelease.

## Success Criteria

- [x] Dry-run published 7 correct assets, then release + tag fully deleted.
- [x] `v2.6.0` annotated on `$RELEASE_SHA`; `release.yml` green.
- [x] Release page: 7 assets, correct notes, latest, not prerelease.

## Risk Assessment

- **Pipeline failure on the real tag after a green dry-run:** only possible
  via non-determinism (runner outage, GH API). Do NOT delete `v2.6.0`; fix
  the workflow forward and re-run the failed job (`gh run rerun`) — the tag
  ref is fine, only the workflow re-executes.
- **Accidental `--follow-tags` or tagging the wrong SHA:** steps pin
  `$RELEASE_SHA` explicitly and push the tag ref alone.
- **Concurrent merge between steps 1 and 5:** irrelevant — tag targets
  `$RELEASE_SHA`, not the branch tip.
