---
title: "Release Engineering v1.0.0"
description: "Tag v1.0.0 + professional release pipeline: hybrid version, GoReleaser, release CI, CHANGELOG, repo hygiene"
status: completed
priority: P2
branch: "main"
tags: [release, goreleaser, ci, go, tooling]
blockedBy: []
blocks: []
created: "2026-05-18T14:11:48.850Z"
createdBy: "ck:plan"
source: skill
---

# Release Engineering v1.0.0

## Overview

Typeburn v1.0 is feature-complete but never released (no tags, no version in binary, test-only CI).
This plan cuts a real `v1.0.0` and adds professional release engineering: a hybrid version
mechanism (ldflags override + `debug.ReadBuildInfo` fallback), GoReleaser cross-platform binary
pipeline, tag-triggered release CI that **self-gates with its own test job**, a Keep-a-Changelog
`CHANGELOG.md` (authoritative release-notes source), and repo hygiene (badges, install docs,
SECURITY.md, CONTRIBUTING, issue/PR templates).

Source brainstorm: `plans/reports/brainstorm-20260518-release-engineering-v1.md`.
Red-team reviewed 2026-05-18 — 15 findings accepted & applied (see `## Red Team Review`).
TDD mode: Phase 1 is tests-first; config/doc phases use `goreleaser check` / snapshot /
render verification gates (no unit-testable surface — honest).

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Version package & --version flag (TDD)](./phase-01-version-package-version-flag-tdd.md) | Completed |
| 2 | [GoReleaser config & Makefile wiring](./phase-02-goreleaser-config-makefile-wiring.md) | Completed |
| 3 | [Release CI workflow](./phase-03-release-ci-workflow.md) | Completed |
| 4 | [CHANGELOG & repo hygiene](./phase-04-changelog-repo-hygiene.md) | Completed |
| 5 | [Release execution & verification](./phase-05-release-execution-verification.md) | Completed |

## Execution Outcome — 2026-05-18

**`v1.0.0` released:** https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0
(tag on `9f96ff8`, 7 assets = 6 archives + checksums.txt, curated CHANGELOG notes,
checksum cross-check verified, `go install …@v1.0.0` → `typeburn v1.0.0`).

**Deviation from locked plan (empirically forced, intent preserved):** the locked
F11 mechanism `changelog.disable: true` + `--release-notes` was proven defective
by the disposable dry-run — GoReleaser docs confirm `changelog.disable:true`
*also ignores* `--release-notes` and publishes an EMPTY release body. Fix: dropped
`disable:true`, added `changelog.filters.exclude: ['.*']`; `--release-notes`
(always passed by release.yml) now supplies the curated body while git log still
never leaks. Plan **intent** (curated CHANGELOG = sole notes source, never git
log) is preserved; only the broken mechanism changed. Caught by the dry-run
exactly as F10 designed. A second dry-run verified body_len 1→1869 before the
real tag.

**Other resolved item:** version-banner v-prefix asymmetry (`go install`→`v1.0.0`
vs archive→`1.0.0`) fixed pre-tag by `version.Version=v{{ .Version }}` in
`.goreleaser.yaml` (user-approved). Both install paths now report `v1.0.0`.

**Non-blocking observation:** SHA-pinned actions (checkout v4 / setup-go v5 /
goreleaser-action v6) run on Node20, GitHub-deprecated (forced Node24 by
2026-06-02). Release unaffected; bump pins deliberately per CONTRIBUTING before
that date.

## Locked Decisions (from brainstorm + red-team)

- Version number: **v1.0.0**.
- Version source: **hybrid** — `internal/version` vars set by `-ldflags -X`; empty → `debug.ReadBuildInfo()`.
  Module path `github.com/bavanchun/Typeburn` (capital T) kept; ldflags case verified correct.
- Distribution: GitHub Release binaries + `go install` live; **Homebrew deferred** — NOT a commented
  dead-schema block; prose TODO in CONTRIBUTING instead (`brews:` is removed-schema in newer GoReleaser).
- **Signing out of scope (user-locked)**: no cosign. Disclosure only — SECURITY.md + README
  "binaries unsigned" note. Integrity = HTTPS transport + `checksums.txt` only, stated plainly.
- Release CI **self-gates**: `ci.yml` does NOT fire on tag push (verified `ci.yml:3-7`,
  `branches:["**"]`, no `tags:`), so `release.yml` runs its own `test` job before publishing.
- Pinned tooling: actions SHA-pinned in `release.yml`; GoReleaser pinned to the exact version
  validated by local `make snapshot`.
- First-release notes come from curated `CHANGELOG.md`, NOT GoReleaser git changelog
  (repo history has zero `feat:`/`fix:` commits — auto notes would be empty).
- Constraints: Go 1.26, files <200 lines kebab-case, `ci.yml` untouched, conventional commits, no AI refs.

## Critical Ordering

Phases 1→4 committed+pushed BEFORE Phase 5 tags. GoReleaser builds the tagged commit; the
tagged commit gets **no `ci.yml` coverage** (tag push doesn't trigger it) — `release.yml`'s
own `test` job is the gate. Phase 5 pins the exact CI-green SHA, runs a **disposable
pre-release tag dry-run** of the full publish path, then cuts `v1.0.0`. Rollback for any tag
that reached the module proxy is **fix-forward to v1.0.1** (sumdb is append-only — never re-tag).

CHANGELOG.md is created in Phase 4 but Phase 2's `make snapshot` runs first: Phase 2 archive
`files:` lists only README+LICENSE; Phase 4 adds CHANGELOG.md to that list when it creates the file.

## Dependencies

Strictly sequential: 1 → 2 → 3 → 4 → 5. Each blocks the next.
No cross-plan deps (only other plan `260518-1454-monkeytype-tui-implementation` is `completed`).
External: GoReleaser CLI (exact pinned version, local Phase 2/5 verification) +
SHA-pinned `goreleaser/goreleaser-action` (CI).

## Red Team Review

### Session — 2026-05-18
**Findings:** 15 (15 accepted, 0 rejected). 8 raw findings deduped from 23 across 3 reviewers
(Security Adversary, Failure Mode Analyst, Assumption Destroyer).
**Severity breakdown:** 3 Critical, 7 High, 5 Medium.

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | `before.hooks` mutates go.sum + runs arbitrary test code in `contents:write` job | Critical | Accept | Phase 2, 3 |
| 2 | `ci.yml` never fires on tag push → tagged ref has zero CI; release.yml must self-gate | Critical | Accept | Phase 3, 5 |
| 3 | Rollback "re-tag v1.0.0" poisons immutable sumdb — must fix-forward | Critical | Accept | Phase 5, plan |
| 4 | Phase 2 snapshot archives CHANGELOG.md before Phase 4 creates it → gate fails | Critical | Accept | Phase 2, 4 |
| 5 | `go install` proxy lag + capital-T case sensitivity (rename declined by user) | High | Accept | Phase 4, 5 |
| 6 | No `concurrency:`; `git push --follow-tags` double-fires; re-run upload race | High | Accept | Phase 3, 5 |
| 7 | Partial asset upload → checksums≠binaries, no detection/decision tree | High | Accept | Phase 3, 5 |
| 8 | `~> v2` floating tool + mutable-tag actions on public-binary pipeline | High | Accept | Phase 2, 3 |
| 9 | Deprecated `format/brews` schema → plural `formats:`, drop dead brews block | High | Accept (mod) | Phase 2, 4 |
| 10 | "snapshot proves exact pipeline" false — publish path untested | High | Accept (mod) | Phase 2, 5 |
| 11 | GoReleaser git-changelog empty for first release (no feat:/fix: history) | Medium | Accept | Phase 2, 4, 5 |
| 12 | `flag.Parse()` unknown-arg/`-h` exits 2 — regression, untested | Medium | Accept | Phase 1 |
| 13 | `-v`=version freezes public CLI contract — drop `-v`, `--version` only | Medium | Accept (mod) | Phase 1 |
| 14 | Unsigned binaries + trust badges, no disclosure | Medium | Accept (disclosure only; cosign NOT added — user scope) | Phase 4 |
| 15 | `contents:write` workflow-scoped not job-scoped; repo default token unverified | Medium | Accept | Phase 3, 5 |

**User decisions on one-way-door / scope items:**
- Finding 5: keep `Typeburn` name; README case note + proxy-lag caveat only (no rename).
- Finding 14: keep signing cut; disclosure (SECURITY.md + note) only, no cosign.

### Whole-Plan Consistency Sweep — 2026-05-18

Re-read `plan.md` + all 5 phase files after applying findings. Grepped superseded terms
(`go mod tidy`, `-v` alias, re-tag same version, "coexist", `~> v2`, "ci.yml unaffected").

**Result: zero unresolved contradictions.** All matches are correctly-negated current text
(e.g. Phase 1 `-v` lines are the new fallthrough *tests*; Phase 5 "re-tag" is "NEVER
re-tag"; Phase 4 "no coexisting claim"; plan.md `~> v2` is the finding description).

Decision deltas reconciled across phases:
- before.hooks → build-only (Ph2) ↔ test job moved to release.yml (Ph3) ↔ plan Locked Decisions ✓
- CHANGELOG ordering: Ph2 archive `files:`=README+LICENSE → Ph4 adds CHANGELOG ↔ Critical Ordering ✓
- changelog.disable (Ph2) ↔ CHANGELOG canonical + `.github/release-notes.md` (Ph4/Ph5) ↔ release.yml `--release-notes` (Ph3) ✓
- Homebrew = prose TODO / `homebrew_casks:` future (Ph2/Ph4/Ph5), no dead `brews:` block ✓
- fix-forward rollback (Ph5) ↔ Critical Ordering ↔ Locked Decisions ✓
- Pinned GoReleaser `v2.x.y` placeholder consistent across Ph2/Ph3/Ph4 (implementer fills concrete) ✓

**One implementation parameter to confirm (not a contradiction):** the Phase 3 post-publish
asset-count assertion and Phase 5 verification use "6 archives + checksums.txt" = 7 assets,
which assumes the full linux/darwin/windows × amd64/arm64 matrix. If windows/arm64 is
dropped at implementation, the expected count must be recomputed from the final build
matrix. Treat the expected count as derived from the matrix, not hardcoded.

Plan is internally consistent and ready for implementation.

## Validation Log

### Session — 2026-05-18
Step 2.5 verification pass **skipped per guard**: `## Red Team Review` already carries
full codebase `file:line` evidence from 3 reviewers; no `[UNVERIFIED]` tags present.
Interview limited to genuine open decision points (locked decisions NOT re-litigated).

**3 questions asked — all confirmed the plan as written:**

| Decision point | Answer | Effect |
|---|---|---|
| Build matrix | Full 6: linux/darwin/windows × amd64/arm64 | **Asset count LOCKED = 7** (6 archives + `checksums.txt`). Resolves the sweep's open parameter. Windows builds accepted as lower-tested. |
| Publish mode | Auto-publish (`draft: false`) | As planned. Safety rests on disposable dry-run + Phase 3 asset-count assertion. |
| Disposable dry-run | Yes, on the public repo | As planned (Phase 5). Transient public pre-release `v0.0.0-rc.test` accepted, then deleted. |

No phase files required propagation edits — answers matched existing plan content. The
sole change is parameter resolution: the Phase 3 / Phase 5 asset-count assertion expected
value is now fixed at **7** (was "derived from matrix").

### Whole-Plan Consistency Sweep — 2026-05-18 (post-validation)
Re-read all files. No new contradictions introduced (answers were confirmations, not
changes). Earlier sweep's only open item — asset-count parametrization — is now closed:
expected = 7, consistent across Phase 3 (assertion) and Phase 5 (verification). Windows ×
arm64 retained, so "6 archives" wording in Phase 2/3/5 is exact, not approximate.

**Recommendation: PROCEED.** Zero unresolved contradictions; Verification Failed: 0.
