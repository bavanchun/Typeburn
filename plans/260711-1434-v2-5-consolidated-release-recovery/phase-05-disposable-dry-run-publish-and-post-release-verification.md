---
phase: 5
title: Disposable Dry Run Publish and Post-Release Verification
status: completed
priority: P1
effort: 3h + proxy wait
dependencies:
  - 4
---

# Phase 5: Disposable Dry Run Publish and Post-Release Verification

## Overview

Prove the GitHub prerelease publish path with a unique disposable tag, capture
evidence, clean GitHub/git refs idempotently, then create immutable `v2.5.0` on
the same `RELEASE_SHA`. Stable-only Homebrew write remains a preflighted,
fix-forward risk and is verified immediately after the real publish.

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [Release flow](./research/release-flow.md)
- Workflows: `.github/workflows/release.yml`, `.goreleaser.yaml`

## Requirements

- Unique disposable tag; never reuse `v0.0.0-rc.test` names.
- Dry-run must be prerelease, non-draft, exactly 7 assets, exact notes, valid
  checksums, correct archive members/banners, and leave latest/tap unchanged.
- Capture evidence before cleanup; independently delete release, remote tag and
  local tag if present. Never claim deletion from proxy/sumdb/CDN history.
- Real `v2.5.0` tag must be annotated, absent remotely, and peel to `RELEASE_SHA`.
- Never delete/move/re-tag real v2.5.0 after push; fix forward to v2.5.1.
- Verify GitHub, installer, Go module, Homebrew tap, and update discovery.
- Publish a final docs/plan PR after remote truth succeeds: roadmap becomes
  published v2.5.0 and this completed plan directory becomes durable.

## Architecture

```text
RELEASE_SHA
  ─> unique disposable annotated tag
  ─> tag Release workflow + publish assertions
  ─> evidence + complete cleanup
  ─> immutable annotated v2.5.0 tag on same SHA
  ─> Release workflow success
  ─> multi-channel remote truth
```

### Dependency Map

- Blocked by exact-SHA local snapshot and release-prep truth.
- Real tag blocked by every disposable gate and cleanup assertion.
- Post-release verification blocked by successful real Release workflow.
- Any content correction after real tag becomes `v2.5.1`, never a moved tag.

## File and External Surface Inventory

| Action | Surface | Verification |
|---|---|---|
| Create/delete | Unique disposable git tag + GitHub prerelease | Full workflow proof |
| Create immutable | Annotated `v2.5.0` tag/release | SHA + workflow proof |
| Read | Release assets/checksums/body | Exact matrix |
| Read | `bavanchun/homebrew-tap-typeburn/Casks/typeburn.rb` | Dry unchanged; stable updated |
| Execute in temp dirs | `install.sh`, `go install` | Version/banner smoke |
| Read | GitHub latest/update endpoint/main CI/PRs | Final remote truth |
| Modify after publish | `docs/project-roadmap.md` | Published v2.5.0 evidence |
| Add after publish | This plan/research directory | Durable completed artifact |
| Add after publish | `docs/journals/260711-1457-v2-5-consolidated-release-planning.md` | Planning decision record |
| No change | User journal and other source files | Hash/status assertion |

## Function and Interface Checklist

- [ ] Locate workflow runs by exact tag/head SHA, not latest run order.
- [ ] Peel annotated tags and compare with frozen SHA.
- [ ] Validate six platform archives plus `checksums.txt`.
- [ ] Validate prerelease does not become `/releases/latest` or update tap.
- [ ] Validate stable release does become latest and updates tap correctly.
- [ ] Bound Go proxy retry to documented ingestion window; do not treat lag as corruption.
- [ ] Use isolated `XDG_STATE_HOME` and forced `version --check-update` path.
- [ ] Capture workflow run ID using Release workflow, push event, tag ref,
  creation window and head SHA; record run attempt and job conclusions.

## Publish and Channel Matrix

| Priority | Scenario | Expected |
|---|---|---|
| Critical | Disposable Release workflow | Success; prerelease=true; draft=false |
| Critical | Disposable assets/body | 7 assets; checksums/members/banner/notes valid |
| Critical | Disposable isolation | Latest remains v2.4.1; tap Cask unchanged |
| Critical | Disposable cleanup | GitHub release and git refs absent; external cache observability acknowledged |
| Critical | Real tag | Annotated v2.5.0 peels to RELEASE_SHA on main |
| Critical | Real workflow | Test and publish jobs success; stable/latest v2.5.0 |
| High | Installer pinned v2.5.0 | Temp install succeeds; lowercase binary banner correct |
| High | `go install @v2.5.0` | Bounded retry; capital binary banner correct |
| High | Homebrew tap | Cask version/URLs/checksums match release |
| High | Update discovery | Older released binary sees v2.5.0 |
| Medium | Final repo truth | PRs merged, workflows green, no release PR/tag drift |
| Medium | Post-release docs | Roadmap truth + completed plan PR merged after tag |

## Implementation Steps

1. Reassert `RELEASE_SHA`, absent v2.5 tag and intended release UTC date. If
   main advanced, require `RELEASE_SHA` remains on main; do not silently retarget.
2. Record journal SHA-256/metadata and tap Cask commit/content; confirm rollback operator.
3. Create/push unique timestamped disposable annotated tag.
4. Resolve/persist exact run ID; inspect/download/verify prerelease evidence.
5. Confirm latest release and tap did not change and the tap PAT was absent.
6. Run resumable cleanup: delete GitHub release if present, remote tag if present,
   local tag if present; verify each repository-visible state independently.
7. Reassert tag absence and `RELEASE_SHA` ancestry on main; create annotated v2.5.0.
8. Push only `refs/tags/v2.5.0`; resolve/watch exact Release run and attempt.
9. If the workflow is red after any external mutation, execute the recovery matrix.
10. Verify assets in temp dirs; use clean Go caches and proxy-only module checks;
    use isolated XDG state for forced update discovery.
11. Verify Homebrew stable mutation; revert a bad tap commit with human credential.
12. Create post-release docs PR updating roadmap to published v2.5.0 and adding
    the completed plan/research files plus the planning journal; leave the user
    punctuation/numbers journal untouched.
13. Record URLs, SHAs, run IDs, asset count, tap commit and bounded proxy lag.
14. Any content correction after real tag becomes v2.5.1; never retag.

## Command Skeleton

```sh
DRY_TAG="v0.0.0-rc.test.$(date -u +%Y%m%d%H%M%S)"
git tag -a "$DRY_TAG" "$RELEASE_SHA" -m "Typeburn disposable release test $DRY_TAG"
git push origin "refs/tags/$DRY_TAG"
# watch exact run, verify assets/body/latest/tap, then:
gh release view "$DRY_TAG" -R bavanchun/Typeburn >/dev/null 2>&1 && \
  gh release delete "$DRY_TAG" -R bavanchun/Typeburn --yes || true
git ls-remote --exit-code --tags origin "refs/tags/$DRY_TAG" >/dev/null 2>&1 && \
  git push origin ":refs/tags/$DRY_TAG" || true
git rev-parse -q --verify "refs/tags/$DRY_TAG" >/dev/null && git tag -d "$DRY_TAG" || true

TAG=v2.5.0
git tag -a "$TAG" "$RELEASE_SHA" -m "Typeburn v2.5.0"
test "$(git rev-parse "$TAG^{}")" = "$RELEASE_SHA"
git push origin "refs/tags/$TAG"
```

## Success Criteria

- [ ] Disposable publish passes every matrix row; GitHub release/git refs are removed,
      tag is never reused, and possible external-cache observability is recorded.
- [ ] Real annotated tag and release point to the proven main SHA.
- [ ] Release is stable/latest with exactly 7 verified assets and exact notes.
- [ ] Installer, Go module, Homebrew Cask and update discovery converge on v2.5.0.
- [ ] Main CI and Release workflow are green; no unresolved release PR remains.
- [ ] Journal remains untouched/untracked; no real tag is deleted or moved.
- [ ] Post-release docs/plan PR is merged and does not alter the release tag SHA.

## Partial-Publish Recovery Matrix

| Observed state | Immediate containment | Recovery rule |
|---|---|---|
| Workflow fails before release exists | Preserve evidence; clean disposable refs idempotently | For real tag, preserve tag and rerun only unchanged transient failure |
| GitHub release exists but assets/notes invalid or partial | Mark release draft if possible; stop announcements/install tests | Preserve real tag; correct content only in v2.5.1 |
| Tap changed during disposable run | Block real tag; revert exact tap commit with human credential | Verify prior Cask commit restored before cleanup continues |
| Stable release valid but tap step fails | Keep valid tag/release; repair/retry unchanged tap publication | Do not cut patch unless artifact/content changes |
| Stable release and tap both bad | Draft/contain release, revert tap, publish incident note | Preserve v2.5.0; fix forward v2.5.1 |
| Operator interrupted | Re-run state audit, not the full sequence blindly | Resume from observed GitHub/tag/tap state |

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Partial stable release before asset assertion | Public broken release | Disposable full proof; patch fix-forward incident path |
| Reusing/deleting real tag | Permanent Go proxy breakage | Immutable-tag hard rule |
| Prerelease mutates tap/latest | Users receive dry build | Verify `prerelease:auto` and `skip_upload:auto` empirically |
| Homebrew Cask bad after stable | Breaks installs/upgrades | Human tap revert + v2.5.1 fix-forward |
| Go proxy delay | False failure/re-tag temptation | Bounded retry; never retag |
| Wrong workflow run observed | False evidence | Match exact tag and SHA |

## Security Considerations

- Keep `HOMEBREW_TAP_TOKEN` step-scoped and never inspect/log its value.
- Release binaries remain unsigned; checksums detect corruption, not host compromise.
- Download and inspect assets in temporary directories; verify checksums before execution.
- “Verified” means checksum-consistent with the same unsigned release host, not
  independent publisher authenticity.

## Unresolved Questions

None.

## Supersession Outcome

The GitHub/archive/installer/Homebrew/updater portions completed for v2.5.0,
but `go install @v2.5.0` is impossible because that immutable tag has a v1-style
module path. Its Go-module success criterion above intentionally remains
unchecked. Corrective v2.5.1 migrated to `/v2`, published from merged SHA
`8307ee6c`, and passed isolated public proxy installs for both exact and latest.
