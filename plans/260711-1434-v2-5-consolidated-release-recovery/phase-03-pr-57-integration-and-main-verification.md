---
phase: 3
title: PR 57 Integration and Main Verification
status: completed
priority: P1
effort: 1h
dependencies:
  - 2
---

# Phase 3: PR 57 Integration and Main Verification

## Overview

Prove PR #57 is current, reviewable, complete and CI-green; squash-merge it to
protected `main`, then verify remote main CI and freeze the feature integration
SHA. Fix the review-discovered Code-default Settings panic before the merge. No
release metadata is cut in this phase.

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [Release flow](./research/release-flow.md)
- PR: `https://github.com/bavanchun/Typeburn/pull/57`

## Requirements

- Existing PR #57 is extended; do not replace it or force-push reviewed history.
- A persisted `default_mode=code` must render Settings safely with an explicit
  no-length value; add a regression test before merge.
- Satisfy live branch-protection requirements; resolve actionable conversations.
- Merge only via squash; branch auto-delete; never push directly to main.
- Verify main by remote SHA and exact workflow run, not local assumptions.
- Preserve journal as the only unrelated untracked file.

## Architecture

```text
feature branch + explicit commits
  ─> PR #57 required checks/review
  ─> squash merge protected main
  ─> origin/main CI success
  ─> FEATURE_SHA evidence
  ─> release-prep branch may begin
```

### Dependency Map

- Requires Phase 1 behavior/tests and Phase 2 truth docs on PR #57.
- Blocks release-prep branch creation.
- Any post-review code change resets required check evidence.

## File Inventory

| Action | Surface | Expected state |
|---|---|---|
| Modify | `internal/ui/settings_rows.go`, `internal/ui/screen_settings.go`, `internal/ui/screen_settings_test.go` | Code default has a safe no-length Settings row and regression test |
| Modify | `docs/cli-reference.md`, `docs/system-architecture.md`, `docs/project-overview-pdr.md`, `docs/codebase-summary.md` | Explain how the three-choice TUI represents a persisted Code default |
| External update | PR #57 title/body | Consolidated v2.5 scope and gate evidence |
| External mutation | GitHub merge | Squash merge after checks/reviews |
| Local ref update | `main`, `origin/main` | Fast-forward to squash SHA |

## Function and Interface Checklist

- [ ] Review complete PR diff, including shared formatter and best eligibility callers.
- [ ] Verify the Code-default Settings regression covers the CLI-persisted mode.
- [ ] Confirm no storage schema, dependency, workflow or release-note change slipped in.
- [ ] Confirm required branch-protection checks by name and conclusion.
- [ ] Confirm PR merge commit equals updated `origin/main`.

## Verification Matrix

| Priority | Gate | Expected |
|---|---|---|
| Critical | `gh pr view 57` | OPEN/CLEAN/MERGEABLE before merge; MERGED after |
| Critical | Required checks | Ubuntu, macOS, installer/release config all success |
| Critical | Merge method | Squash only; origin/main at squash SHA |
| High | Main workflow | Exact push run for squash SHA succeeds |
| High | Workspace | Journal untouched/untracked; no accidental staged path |
| Medium | Branch deletion | Remote feature branch auto-deleted |

## Implementation Steps

1. Fetch/prune/tags; verify branch divergence and clean intended diff.
2. Update PR #57 title/body with consolidated scope and evidence.
3. Fix and test the review-discovered Code-default Settings panic, then run one
   final local integration gate and explicit staging audit.
4. Capture the protected journal SHA-256/metadata. Confirm the plan directory
   and new `260711-1457-v2-5-consolidated-release-planning.md` are the only other
   authorized untracked artifacts; both remain local until Phase 5.
5. Watch PR checks by exact head SHA; verify actual protection requirements.
6. Squash-merge PR #57 using repository policy.
7. Switch to main and `git pull --ff-only origin main`.
8. Capture `FEATURE_SHA`; select/watch the exact main CI run ID to success.
9. Recheck tags/releases: stable must still be v2.4.1.

## Command Skeleton

```sh
git fetch origin --prune --tags
git rev-list --left-right --count origin/main...HEAD
gh pr checks 57 -R bavanchun/Typeburn --watch
gh pr merge 57 -R bavanchun/Typeburn --squash --delete-branch
git switch main && git pull --ff-only origin main
FEATURE_SHA=$(git rev-parse HEAD)
test "$(git rev-parse origin/main)" = "$FEATURE_SHA"
# Resolve the run by workflow + push event + headSha; persist its databaseId.
```

## Success Criteria

- [ ] PR #57 merged by squash with all required checks successful.
- [ ] Exact main CI push run succeeds for `FEATURE_SHA`.
- [ ] No v2.5.0 tag/release exists yet.
- [ ] Feature branch deletion and workspace/journal state verified.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Mergeability changes while waiting | Stale approval/check evidence | Refresh immediately before merge |
| Force-push invalidates review | Hidden history change | Normal commits or GitHub update branch |
| Watching wrong CI run | False green | Match exact head SHA/event |
| Local main diverges | Release from wrong commit | ff-only pull + SHA equality |

## Security Considerations

Branch protection and required checks are release supply-chain controls. Do not
bypass administrators, dismiss checks, or use privileged direct pushes.

## Unresolved Questions

None.
