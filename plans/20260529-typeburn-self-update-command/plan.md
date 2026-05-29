---
title: Typeburn Self-Update Command (typeburn update)
description: ''
status: completed
priority: P2
branch: feat/typeburn-self-update
tags: []
blockedBy: []
blocks: []
created: '2026-05-29T16:04:43.896Z'
createdBy: 'ck:plan'
source: skill
---

# Typeburn Self-Update Command (typeburn update)

## Overview

Add `typeburn update`: a stdlib-only self-replacing updater. Detects a newer
stable release (reusing `internal/update.Check`), and for self-managed installs
downloads the matching release archive, verifies sha256 against `checksums.txt`,
extracts the binary, and atomically swaps the running executable in place. For
package-manager-managed installs (Homebrew, `go install`) it refuses and prints
the correct upgrade command instead. Cross-platform (linux/darwin/windows ×
amd64/arm64). No new go.mod entries; every Go file <200 LOC; strict layering
(download/verify/extract/replace in `internal/update`, CLI wiring in
`internal/cli`).

Design source: [brainstorm-summary.md](./brainstorm-summary.md).

**TDD:** each phase writes tests first (httptest servers, temp-dir fixtures,
table-driven), then implements to green. Security-sensitive verify/replace paths
are locked by tests before the code lands.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Download and Verify](./phase-01-download-and-verify.md) | Completed |
| 2 | [Archive Extraction](./phase-02-archive-extraction.md) | Completed |
| 3 | [Selfpath and Atomic Replace](./phase-03-selfpath-and-atomic-replace.md) | Completed |
| 4 | [CLI Command Wiring](./phase-04-cli-command-wiring.md) | Completed |
| 5 | [Docs and CI Gate](./phase-05-docs-and-ci-gate.md) | Completed |

## Key Constraints (from CLAUDE.md + brainstorm)

- **Deps:** stdlib + charm + cobra + `golang.org/x` only. No `minio/selfupdate`.
- **Layering:** no UI deps in `internal/update`; CLI-only logic in `internal/cli`.
- **Security bar = install.sh:** HTTPS, official-repo URL guard (`releaseURLPrefix`),
  reject prereleases, mandatory sha256, download size cap, tar path-traversal
  guard. Honest caveat: unsigned binaries → trust == `curl install.sh | sh`.
- **Asset naming:** `typeburn_<ver-no-v>_<os>_<arch>.{tar.gz|zip}` (windows=zip).
  Note GoReleaser strips the leading `v`; the updater must too.
- **File size:** <200 LOC each; snake_case core filenames; build-tagged
  `replace_unix.go` / `replace_windows.go`.

## Dependencies

- **Builds on (completed):** `plans/20260521-0700-typeburn-update-check-cli`
  — shipped `internal/update` (Check/FetchLatest/Compare/Result, v2.1.0). This
  plan extends that package; no blocking relationship (foundation already merged).
- No open plans block or are blocked by this one.

## Red Team Review

### Session — 2026-05-29
**Reviewers:** Security Adversary, Failure Mode Analyst, Assumption Destroyer (code-reviewer, Full tier, evidence-required)
**Findings:** 15 (15 accepted, 0 rejected) — 22 raw, deduped to 15
**Severity breakdown:** 4 Critical, 9 High, 2 Medium
**Reports:** `reports/from-code-reviewer-to-planner-red-team-{security-adversary,assumption-destroyer}-plan-review-report.md`

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | Redirect model misread `client.go` (`ErrUseLastResponse`≠block); CDN host unverified | Critical | Accept | Completed |
| 2 | TOCTOU/symlink-race on replace; no O_EXCL; chown is escalation | Critical | Accept | Completed |
| 3 | Windows interrupted swap leaves no binary; no rollback | Critical | Accept | In Progress |
| 4 | EXDEV: extract temp must land in target dir, not OS temp | Critical | Accept | Phase 2 |
| 5 | checksums/`releaseURLPrefix` not independent trust anchors (wording) | High | Accept | Phase 1 |
| 6 | Symlink/non-regular archive member not rejected | High | Accept | Phase 2 |
| 7 | Downgrade: `Compare` returns 0 on malformed/strips suffixes; add string guard | High | Accept | Phase 1 |
| 8 | Cache-poisoning steers tag; Apply must use live forced fetch only | High | Accept | Phase 1/4 |
| 9 | `Result.Latest` v-prefix ambiguity → 404; pin authoritative tag | High | Accept | Phase 1 |
| 10 | No pre-flight writability probe; EACCES after full download | High | Accept | Phase 3/4 |
| 11 | Managed-classification incomplete (GOBIN-unset, scoop/choco, basename casing) | High | Accept | Phase 3 |
| 12 | dev build + `--check` should exit-0 skip, not refuse | High | Accept | Phase 4 |
| 13 | Non-tty prompt: `InOrStdin()` has no fd for isatty | High | Accept | Phase 4 |
| 14 | Concurrent-update race; no lock (cache already locks) | Medium | Accept | Phase 3 |
| 15 | Binary size-cap checkpoint after Phase 2 (~17% headroom) | Medium | Accept | Phase 2 |

**Note on #5:** accepted as a *wording* correction only — does NOT reverse the
user-confirmed unsigned/checksum-only trust model; code signing stays deferred.

### Whole-Plan Consistency Sweep
Re-read `plan.md` + all five phase files after applying findings. Reconciled
duplicate prose so bodies match the fixes:
- Phase 1: redirect bullet now "follow to verified github-owned allowlist" (not
  hardcoded single host); single redirect host de-hardcoded.
- Phase 2: `extractBinary` signature gained `destDir` (extract into target dir).
- Phase 3: unix replace = O_EXCL temp + lstat-refuse-symlink + reapply mode, no chown.
- Phase 4: dev/`--check` exits 0 (skip); `(nil,nil)` handled only in step 1.
- No stale references to a hardcoded redirect host, `preserve owner`, or the old
  2-arg `extractBinary` remain (grep-verified). **Zero unresolved contradictions.**

## Implementation Results (2026-05-29)

All five phases implemented TDD on branch `feat/typeburn-self-update`. Full CI
gate green: `gofmt -l` clean, `go vet ./...` clean, `go test ./... -race` all
pass, `make size-check` exit 0 (binary ~8.87 MB).

Post-implementation code review caught two defects the plan's red-team did not,
both fixed with regression tests:
- **C1 (Critical):** `Apply` built the asset name from the *current* version, not
  the *target* tag — every real upgrade would have requested
  `typeburn_<old>_…` under the new tag and 404'd. Tests masked it by holding the
  version constant. Fix: `downloadVerified` now derives the asset name from
  `rawTag` itself (dropped the redundant `version` param); `Apply` tests now
  assert a genuine current≠target (`2.2.0` → `v2.3.0`) swap.
- **H1 (High):** the redirect allowlist didn't enforce the `https` scheme, so a
  302 to cleartext `http` on a GitHub host would have been followed — undermining
  the checksum trust anchor. Fix: `newDownloadClient` refuses non-https redirects
  (loopback exempt for test servers).
