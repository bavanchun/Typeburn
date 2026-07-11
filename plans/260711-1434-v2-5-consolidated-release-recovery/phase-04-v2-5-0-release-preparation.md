---
phase: 4
title: v2.5.0 Release Preparation
status: completed
priority: P1
effort: 2h
dependencies:
  - 3
---

# Phase 4: v2.5.0 Release Preparation

## Overview

Create a focused release-prep PR from proven main, harden the privileged tag
workflow, cut the combined `v2.5.0` metadata with the intended stable-tag UTC
date, merge after CI, freeze `RELEASE_SHA`, and prove local artifacts.

## Context Links

- Plan: [plan.md](./plan.md)
- Research: [Release flow](./research/release-flow.md)
- Runbook: `CONTRIBUTING.md`, `.github/workflows/release.yml`, `.goreleaser.yaml`

## Requirements

- Branch `chore/release-v2.5.0` from current `origin/main`.
- Modify only `CHANGELOG.md`, `.github/release-notes.md`, and
  `.github/workflows/release.yml` in release prep; keep `ci.yml` byte-identical.
- Cut all combined content from Unreleased to dated v2.5.0; leave Unreleased empty.
- Release-notes body matches the v2.5.0 CHANGELOG section exactly by defined extraction.
- Use GoReleaser exactly v2.15.4; prove build/archive locally.
- Any fix after SHA freeze requires a new PR and complete re-freeze/retest.
- Tag workflow must reject commits not reachable from `origin/main` before the
  privileged publish job.
- Prerelease GoReleaser step must not receive `HOMEBREW_TAP_TOKEN`; stable step
  receives it only after a non-mutating tap push-permission preflight.

## Architecture

Release prep is a bounded transaction. Metadata is prepared and merged only
when Phase 5 will immediately prove/publish it. Reserve one operator and a
four-hour release window. If dry-run cannot start within 30 minutes of prep
merge or cannot finish within two hours, create a corrective PR restoring honest
Unreleased state; never leave another phantom release.

### Dependency Map

```text
merged FEATURE_SHA
  ─> release-prep branch (metadata only)
  ─> CI + squash merge
  ─> immutable RELEASE_SHA
  ─> exact-SHA local gates/snapshot
  ─> Phase 5 disposable publish
```

## File Inventory

| Action | File | Change | Verification |
|---|---|---|---|
| Modify | `CHANGELOG.md` | Cut final v2.5.0 section/date/links | Extraction diff |
| Modify | `.github/release-notes.md` | Exact curated v2.5.0 section | Deterministic diff |
| Modify | `.github/workflows/release.yml` | Main ancestry gate; split prerelease/stable token paths; tap permission preflight | Disposable proof |
| Read-only | `.goreleaser.yaml` | Confirm matrix/notes/prerelease/tap | `goreleaser check` |
| Read-only | `Makefile`, installer scripts | Run local gates | Command exit status |

## Function and Interface Checklist

- [ ] Strict, punctuation/numbers, Code label and marker fixes all appear once.
- [ ] `[Unreleased]` link advances to `v2.5.0...HEAD` only in release prep.
- [ ] `[2.5.0]` link targets the future immutable release URL.
- [ ] Release notes omit raw git log and match curated CHANGELOG content.
- [ ] Define extraction bytes: v2.5 heading/body through the line before the
  next version heading, one blank line, then the `[2.5.0]` reference link.
- [ ] Workflow and GoReleaser pins remain v2.15.4 and otherwise untouched.
- [ ] `ci.yml` remains byte-identical per repository release constraint.

## Test and Artifact Matrix

| Priority | Gate | Expected |
|---|---|---|
| Critical | CHANGELOG vs release notes | Defined content byte-equal |
| Critical | Tag commit outside main | Test job fails before publish receives write/PAT |
| Critical | Disposable prerelease environment | Tap PAT absent; GitHub release path succeeds |
| Critical | Stable tap credential preflight | Token reports push permission without mutation |
| Critical | `RELEASE_SHA` | Equals origin/main after prep merge |
| Critical | Snapshot assets | 6 archives + checksums with expected names |
| Critical | Checksums | All downloaded/local assets verify |
| High | Archive members | lowercase binary + README/LICENSE/CHANGELOG |
| High | Native binary banner | Snapshot version + RELEASE_SHA |
| High | Installer harness/shellcheck | Success |
| Medium | Source tree | No feature changes in prep PR |

## Implementation Steps

1. Preflight release owner/window, GitHub access, tap availability, non-mutating
   tap push permission, and human rollback credential availability.
2. Fetch and branch from current `origin/main`; verify v2.5 tag absent.
3. Add tag-commit ancestry gate before privileged publish; split prerelease and
   stable GoReleaser steps so only stable receives the tap PAT.
4. Cut CHANGELOG using the intended stable-tag UTC date and exact combined scope.
5. Deterministically extract v2.5 heading/body plus reference link to release notes;
   compare with `cmp -s`/`diff -u` and document newline boundaries.
6. Install GoReleaser v2.15.4 into a temporary `GOBIN`, assert its reported
   GitVersion, and run focused workflow/config gates with that exact binary.
7. Push focused prep branch, open PR, watch CI, and immediately before merge
   verify its base SHA still equals the frozen feature main SHA.
8. Squash-merge, fast-forward main, freeze and record `RELEASE_SHA`.
9. Re-run exact-SHA artifact matrix, `goreleaser check`, and `make snapshot`.
10. Inspect asset count, names, members, checksum consistency and binary metadata.

## Gate Commands

```sh
make lint
make test-race
make size-check
make notui-noexit-check
shellcheck install.sh scripts/test-install-sh.sh
./scripts/test-install-sh.sh
TOOLS_DIR=$(mktemp -d)
GOBIN="$TOOLS_DIR" go install github.com/goreleaser/goreleaser/v2@v2.15.4
"$TOOLS_DIR/goreleaser" --version | grep -Eq 'GitVersion:[[:space:]]+2\.15\.4'
"$TOOLS_DIR/goreleaser" check
PATH="$TOOLS_DIR:$PATH" make snapshot
```

## Success Criteria

- [ ] Release-prep PR contains only the three approved metadata/workflow files.
- [ ] Curated notes and CHANGELOG pass exact-content comparison.
- [ ] Prep PR squash-merges with CI green; `RELEASE_SHA == origin/main`.
- [ ] Local exact-SHA snapshot and all artifact gates pass.
- [ ] No real or disposable tag remains before Phase 5 starts.

## Risk Assessment

| Risk | Impact | Mitigation |
|---|---|---|
| Metadata merge without publish | Phantom release repeats | Bounded transaction + correction PR if abandoned |
| Notes extraction differs | Wrong public release body | Byte-comparison gate |
| Snapshot run on stale SHA | False proof | Re-run after merge/freeze |
| Tool version drift | Different artifacts | Exact v2.15.4 assertion |
| Prep date crosses before stable tag | False release date | New prep PR, refreeze SHA, rerun gates |

## Security Considerations

Snapshot runs without publish credentials. Never expose GitHub or Homebrew
tokens locally. Prerelease receives no tap PAT; stable PAT remains scoped to
its single step after read-only permission preflight.

## Unresolved Questions

None.
