---
title: Typeburn v2.5.0 Consolidated Release Recovery
description: >-
  Fix Code/Strict result metadata, restore documentation truth, merge PR #57,
  and publish one verified v2.5.0 release.
status: completed
priority: P1
branch: feat/punctuation-numbers-toggle
tags:
  - bugfix
  - docs
  - release
  - critical
blockedBy: []
blocks: []
created: '2026-07-11T07:39:23.409Z'
createdBy: 'ck:plan'
source: skill
---

# Typeburn v2.5.0 Consolidated Release Recovery

## Overview

Deliver one consolidated `v2.5.0` containing already-merged Strict Mode,
PR #57 punctuation/numbers, the Code-mode label fix, best-marker consistency,
and documentation reconciliation. Preserve protected-main and immutable-tag
rules. Public stable remains `v2.4.1` until the real release workflow succeeds.

## Locked Decisions

- One release: `v2.5.0`; do not create or claim `v2.6.0`.
- History and Result render Code exactly as `code`; no rune count in label.
- Unknown persisted mode IDs render raw; empty renders `unknown`.
- Code and Strict records remain stored but never show or set a personal-best star.
- Extend existing PR #57; use a separate release-prep PR.
- Keep `docs/journals/260702-1857-punctuation-numbers-toggle-feature-shipped.md`
  untouched and untracked.
- No storage migration, dependency addition, signing, or unrelated cleanup.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Code Mode Display and Best Marker TDD](./phase-01-code-mode-display-and-best-marker-tdd.md) | Completed |
| 2 | [Pre-Release Documentation Truth](./phase-02-pre-release-documentation-truth.md) | Completed |
| 3 | [PR 57 Integration and Main Verification](./phase-03-pr-57-integration-and-main-verification.md) | Completed |
| 4 | [v2.5.0 Release Preparation](./phase-04-v2-5-0-release-preparation.md) | Completed |
| 5 | [Disposable Dry Run Publish and Post-Release Verification](./phase-05-disposable-dry-run-publish-and-post-release-verification.md) | Completed |

## Dependencies

- Completion is blocked by corrective plan
  `260711-1908-v2-5-1-go-module-v2-fix-forward`: the published v2.5.0 source
  can never satisfy its original Go-install criterion. Close this plan as
  superseded/recovered only after v2.5.1 passes clean proxy-only installation.
- Sequential execution: Phase 1 → 2 → 3 → 4 → 5.
- Prior Strict and punctuation/numbers plans are completed inputs, not blockers.
- PR #57 must remain reviewable and CI-green before merge.
- Real tag creation is blocked by a successful, cleaned disposable-tag publish.

## Outcome / Supersession

- v2.5.0 shipped successfully on GitHub, installer, Homebrew, archives, and updater.
- Its original Go-install criterion failed permanently because the tagged
  `go.mod` lacked the required `/v2` module suffix. That criterion is not
  checked or rewritten as passed.
- Corrective plan `260711-1908-v2-5-1-go-module-v2-fix-forward` completed the
  recovery through immutable v2.5.1. Public proxy-only exact and latest installs
  now succeed with lowercase `typeburn`.
- This plan is terminally completed by documented fix-forward recovery, not by
  claiming every original v2.5.0 channel gate succeeded.

## Scope Challenge

- **Existing reuse:** `storage.Record`, `IsNewBest`, semantic theme/config
  authorities, PR #57, release workflow, and GoReleaser configuration.
- **Minimum set:** one shared formatter, best eligibility consistency, regression
  tests, named docs reconciliation, PR integration, release prep, publish proof.
- **Complexity:** many files because release truth spans code, evergreen docs,
  changelog, GitHub Actions, and distribution channels; no new service/package.
- **Selected scope:** HOLD. No adjacent feature or historical journal rewrite.

## Cross-Plan Dependencies

| Relationship | Plan | Status |
|---|---|---|
| Input | `260626-2213-strict-stop-on-error-mode` | completed |
| Input | `260702-1824-punctuation-numbers-toggle` | completed |

## Whole-Plan Success Criteria

- Code renders as `code` on History and Result; Time/Words/Quote unchanged.
- Code/Strict records never display or earn ★; persisted schema unchanged.
- Docs state 8 themes, 8 persisted/CLI keys, 7 TUI Settings rows, and distinguish
  grayscale `mono` from attribute-only `NO_COLOR`.
- Pre-release truth says stable `v2.4.1`, upcoming combined `v2.5.0`; no phantom
  v2.5/v2.6 release claim.
- PR #57 and release-prep PR squash-merge with required CI green.
- Disposable release proves 7 assets, checksums, notes, prerelease isolation,
  and no Homebrew tap mutation; GitHub release/tag cleanup is idempotently verified.
- Annotated `v2.5.0` points to the proven `main` SHA and release workflow succeeds.
- GitHub latest, assets, installer, Go module, Homebrew tap, and update check agree.
- Full race, vet, format, installer, release-config, and size gates pass.
- Untracked journal remains byte-untouched and unstaged.
- Post-release docs PR records published v2.5.0 truth and commits the completed
  plan artifacts plus this session's planning journal; the release tag itself
  remains on the proven release SHA.

## Research

- [Code/display research](./research/code-mode-display-and-best-marker.md)
- [Documentation truth research](./research/documentation-and-release-truth.md)
- [Release-flow research](./research/release-flow.md)

## Red Team Review

### Session — 2026-07-11

**Findings:** 15 deduplicated (11 accepted, 3 accepted with modification, 1 rejected)
from 33 raw findings. **Severity:** 7 Critical, 7 High, 1 Medium.

| # | Finding | Severity | Disposition | Applied To |
|---|---|---|---|---|
| 1 | Partial public release lacks recovery state machine | Critical | Accept | Completed |
| 2 | Disposable cleanup is not resumable/idempotent | Critical | Accept | Completed |
| 3 | Tap rollback operator/credential not preflighted | Critical | Accept | Completed |
| 4 | Tag workflow does not enforce trusted-main ancestry | Critical | Accept | Completed |
| 5 | Prerelease unnecessarily receives tap PAT | Critical | Accept | In Progress |
| 6 | Post-release roadmap/plan truth would remain stale | Critical | Accept | Phase 5 |
| 7 | Ineligible historical runs can suppress normal PB | High | Accept | Phase 1 |
| 8 | Journal diff cannot prove untouched untracked content | High | Accept | All phases |
| 9 | Exact workflow-run selection was underspecified | High | Accept | Phases 3, 5 |
| 10 | Proxy and update checks can reuse stale caches | High | Accept | Phase 5 |
| 11 | Release-note extraction and tool pin were ambiguous | High | Accept | Phase 4 |
| 12 | Dry run cannot prove stable Homebrew write path | High | Accept (modified) | Phases 4–5 |
| 13 | Disposable public tag cannot be made globally absent | High | Accept (modified) | Phase 5 |
| 14 | Tie semantics and phase gate ownership unclear | High | Accept (modified) | Phases 1–3 |
| 15 | Raw unknown mode needs sanitization/truncation | Critical | Reject | Locked UX/threat model |

Rejected/modified rationale:

- Unknown fallback remains the user-approved raw semantic identifier. History is
  local 0600 XDG data, not network input. Layout/control-character hardening is
  documented as accepted risk and requires a separate user decision.
- History ★ means current eligible record-holder including ties; Result ★ means
  a newly established strictly-greater PB. The distinction is now explicit.
- Disposable tags are unique and never reused, but may remain observable in Go
  proxy/sumdb/CDN history; cleanup claims only GitHub release and git refs.
- Prerelease proves GitHub publish behavior, not stable-only Homebrew mutation.
  Stable tap permission preflight and manual rollback readiness cover that gap.
- CI mutable action tags remain outside scope: `CLAUDE.md` requires `ci.yml`
  byte-identical for release-infra work, while the privileged tag workflow is
  SHA-pinned. Revisit only with explicit user approval.

### Whole-Plan Consistency Sweep

- Files reread: `plan.md`, all five phase files, three research reports.
- Decision deltas checked: trusted-main gate, prerelease PAT isolation, recovery
  matrix, post-release docs, PB eligibility, artifact ownership, cache isolation.
- Reconciled stale references: integration/release ownership, cleanup wording,
  Homebrew proof boundary, release-date definition, caller-count invariant.
- Unresolved contradictions: 0.

## Validation Log

### Session 1 — 2026-07-11

**Trigger:** Automatic `--deep` post-red-team validation.
**Questions recorded:** 3 prior user decisions; no new unresolved decision.

#### Questions and Answers

1. **[Release scope]** Should Strict and punctuation/numbers ship separately or
   in one release?
   - Options: sequential v2.5/v2.6 | one consolidated release
   - **Answer:** one consolidated release.
2. **[UX contract]** Should Code render as `code` or include rune count?
   - Options: `code` | `code <rune-count>`
   - **Answer:** `code`.
3. **[Scope]** Should the untracked shipped journal be corrected/tracked?
   - Options: keep untouched/untracked | edit wording and include
   - **Answer:** keep untouched/untracked.

#### Verification Results

- **Tier:** Full (5 phases; Fact Checker, Flow Tracer, Scope Auditor,
  Contract Verifier).
- **Claims checked:** 78 across paths, symbols, callers, branch/PR state,
  release workflow, config/theme authorities and release artifacts.
- **Verified:** 77 | **Failed:** 1 | **Unverified:** 0.
- Resolved failure: no global `goreleaser` binary exists. Phase 4 now installs
  exact v2.15.4 into a temporary `GOBIN`, asserts GitVersion, and uses that path.
- Live evidence: PR #57 open/mergeable; head `00286ed2`; base `55958224`;
  latest public release `v2.4.1`; plan status `pending`.

#### Confirmed Decisions

- Target version: one combined `v2.5.0`; no phantom `v2.6.0`.
- Code label: exact `code`; stored rune count unchanged.
- Journal: hash-preserved, untracked, never staged.
- Release-prep PR includes trusted-main/tag-token hardening discovered by red team.
- Stable-only Homebrew write cannot be proven by prerelease; preflight and
  fix-forward recovery are explicit.

#### Whole-Plan Consistency Sweep

- Files reread: `plan.md`, five phase files, three research reports.
- Searched stale terms: metadata-only prep, total/global cleanup, two-caller
  invariant, phantom v2.6, early release notes, unresolved placeholders.
- Reconciled phase ownership: Phase 2 code/docs, Phase 3 PR/merge, Phase 4
  workflow+release prep, Phase 5 publish/recovery/post-release docs.
- Unresolved contradictions: 0.
- Implementation eligibility: yes; verification failures remaining: 0.

## Unresolved Questions

None.
