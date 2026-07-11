---
phase: 4
title: "Prepare protected corrective release"
status: pending
effort: "M"
---

# Phase 4: Prepare protected corrective release

## Objective

Land through protected main, freeze one release SHA, and prove archive release behavior without changing stable channels.

## State Inventory

| Surface | Action |
|---|---|
| Corrective branch and PR | required CI then squash merge |
| `main` and `origin/main` | fetch, equalize, freeze merged SHA |
| Disposable prerelease | exact-SHA archive and workflow proof |
| Latest and Homebrew tap | prove isolation during prerelease |
| User journal | hash before and after; leave untracked and unstaged |
| Workspace artifacts | baseline all untracked files and exact stage allowlist |

## Implementation Steps

1. Record `git status --porcelain` and hashes for every pre-existing untracked
   journal/plan. Define stage allowlist and assert cached names; never use `git add .`.
   Re-run module tidy/verify and every prior gate.
2. Push corrective branch, open PR, and wait for every required check.
3. Squash-merge; fetch remote truth and freeze `RELEASE_SHA`.
4. Snapshot latest release ID/tag and tap commit/Cask hash. Verify named human
   credentials can revert the tap before any stable mutation.
5. Publish unique `v0.0.0-rc.test.v251.<UTC>` on that SHA for archive proof
   only. It is intentionally major-incompatible with `/v2`; never query proxy
   or sumdb for it, never reuse it, and never use a valid v2 prerelease here.
6. Capture workflow run ID and attempt; assert workflow name, push event, exact
   headBranch/tag, headSha, creation window, and every required job conclusion.
7. Verify status, body, seven assets, checksums, archive members, lowercase
   command, and disposable-version linker banner. Compare latest/tap snapshots.
8. Delete release and refs idempotently; prove absence and recompare snapshots.

## Verification Checklist

- PR is squash-merged with required checks green.
- Frozen SHA is on `origin/main` and drives the dry-run workflow.
- Stable latest remains v2.5.0 and tap remains unchanged.
- Cleanup does not claim proxy or sumdb erasure.
- Any source repair invalidates `RELEASE_SHA` and all evidence; merge a new PR,
  freeze the new SHA, and repeat every Phase 4 gate.

## Dependencies

- Requires Phases 1 through 3; blocks stable tagging.

## Success Criteria

- [ ] Frozen merged SHA passes local and disposable archive proofs.
- [ ] Seven assets and isolation checks pass; cleanup is verified.
- [ ] Commit requirement is satisfied by squash merge; no synthetic commit.
