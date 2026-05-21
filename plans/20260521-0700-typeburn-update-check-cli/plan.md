---
title: "Typeburn Update-Check CLI (v2.1.0)"
description: >-
  Add a stdlib-only update notifier: `typeburn version --check-update`
  (explicit, sync, human + JSON) plus an opt-in opportunistic check on TUI
  launch (24h cache, 1.5s timeout, silent-degrade) rendering on the Result
  screen footer. Pure-logic `internal/update/` package mirrors existing
  `internal/storage` atomic patterns. Zero new go.mod entries.
  Locked: opt-in default-OFF, all three install methods printed verbatim,
  pre-releases filtered, dev-build skip, no self-upgrade. Ships in v2.1.0
  (separate from v2.0.0 in flight via PR #18).
status: pending
priority: P2
branch: "feat/update-check-v2.1"
tags: [feature, cli, network, release]
blockedBy: []
blocks: []
created: "2026-05-21T06:52:29.557Z"
createdBy: "ck:plan"
source: skill
---

# Typeburn Update-Check CLI (v2.1.0)

## Overview

Notify users of newer releases via two surfaces — both stdlib-only, both
silent-degrade, both opt-in or explicit. No self-upgrade. Ships **after**
v2.0.0 is tagged.

## Context Links

- Brainstorm summary: [brainstorm-summary.md](./brainstorm-summary.md)
- Research: GitHub API/HTTP — [researcher-01](./research/researcher-01-github-api-http.md)
- Research: XDG/storage/semver — [researcher-02](./research/researcher-02-xdg-storage-semver.md)
- Scout: per-phase touchpoints — [scout-touchpoints.md](./research/scout-touchpoints.md)

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Update Package Pure-Logic](./phase-01-update-package-pure-logic.md) | Pending |
| 2 | [Config Integration](./phase-02-config-integration.md) | Pending |
| 3 | [Version --check-update Flag](./phase-03-version-check-update-flag.md) | Pending |
| 4 | [Opportunistic TUI Wiring + Result Footer](./phase-04-opportunistic-tui-wiring-result-footer.md) | Pending |
| 5 | [Docs Sync and Release Prep](./phase-05-docs-sync-and-release-prep.md) | Pending |

## Locked decisions (user-confirmed during brainstorm — do NOT reverse)

- **Default state:** `update_check` = **false** (opt-in). Scout §3 suggested
  `true`; that is reversed per user decision. Plan enforces `false`.
- **Zero new go.mod entries.** Scout §11 Q4 mentioned `Masterminds/semver/v3`;
  reversed. Stdlib comparator only (~30 LOC; see researcher-02 Part B).
- **Cache TTL** 24h, **HTTP timeout** 1.5s, silent on any error.
- **Both triggers:** explicit (`version --check-update`) AND opportunistic
  (config-gated). Render on Result screen footer only.
- **All 3 upgrade commands printed verbatim** — no install-method detection.
- **Pre-release / draft filter:** API responses with `prerelease=true` or
  tags matching `-rc/-beta/-alpha/-pre` or `v0.0.0-*` are treated as
  "no stable release available."
- **Dev-build skip:** when `version.Resolve()` returns `dev`/empty/pseudo-version
  (`debug.ReadBuildInfo` (devel) form), check is skipped entirely.
- **`--no-tui` path does NOT trigger the opportunistic check.** Scout §11 Q1
  flagged this; decision: keep no-TUI scriptable / outbound-traffic-free.

## Dependencies

- **Blocked on:** PR #18 (release-notes v2.0.0 refresh) merged and v2.0.0
  tagged. Branch this work off `main` after v2.0.0 ships.
- **Cross-plan:** none. pro-cli-v2 plan is `status: completed`.

## Success criteria (whole-plan)

- `typeburn version --check-update` returns within 1.6s on cache miss,
  ≤100ms on cache hit, prints the locked 3-command hint or a clean
  "up to date".
- Opportunistic check fires only when `update_check=on` AND a TUI is
  launched AND version is real (not `dev`/pseudo-version).
- All CI gates green: `make test-race`, `make lint`, `make size-check`,
  `notui-noexit-check`.
- `go.mod` diff = zero new direct dependencies.
- Binary size delta < 100KB (stdlib `net/http` already linked elsewhere).
- Each new/modified Go file < 200 LOC.
- `internal/update/` package has **zero** imports of `bubbletea`/`lipgloss`.

## Red Team Review

### Session — 2026-05-21
**Findings:** 15 (15 accepted, 0 rejected from cap of 15; 3 below-cap rejections logged)
**Severity breakdown:** 2 Critical, 7 High, 6 Medium
**Reviewers:** Security Adversary, Failure Mode Analyst, Assumption Destroyer (3 hostile lenses, Full-tier verification)

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| 1 | Bare `typeburn` uses `runtime.go:81 → app.NewFromDisk`, NOT `cmd_run.go`'s `app.New`; Phase 4 misses it | Critical | Accept | Phase 4 |
| 2 | `storage.atomicWrite` requires parent dir to exist; `cache.Save` will silently fail on fresh installs | Critical | Accept | Phase 1 |
| 3 | `strconv.ParseBool` does NOT accept `on/off`; existing `parseBool` (cmd_config.go:138-147) only takes `true/false/1/0` | High | Accept | Phase 2 |
| 4 | `cmd_replay.go` does NOT call `app.New`; Phase 4 Step 5 lists it incorrectly | High | Accept | Phase 4 |
| 5 | No `http.Client.CheckRedirect` — hostile 302 could exfiltrate UA, downgrade scheme | High | Accept | Phase 1 |
| 6 | Cached `Latest`/`ReleaseURL` rendered without re-validation; seeded cache → ANSI/text injection in TUI footer | High | Accept | Phase 1+4 |
| 7 | `storage.AtomicWrite` export widens TOCTOU/symlink blast radius (no `O_EXCL`/`O_NOFOLLOW`) | High | Accept (modified) | Phase 1 |
| 8 | Phase 5 `.github/release-notes.md` pre-stage can leak v2.1 notes into v2.0 release if ordering not strict | High | Accept | Phase 5 |
| 9 | Opportunistic 1.5s + TLS-handshake hang; no `Transport.TLSHandshakeTimeout`/`DialContext` | High | Accept | Phase 1+4 |
| 10 | `"(devel)"` sentinel unreachable — `internal/version.go:46` filters it out (dead branch) | Medium | Accept | Phase 1 |
| 11 | Prerelease filter denylist misses SemVer-2 numeric/canary forms (`v2.0.0-1`, `v2.0.0-canary.4`) | Medium | Accept | Phase 1 |
| 12 | Cache schema not versioned — repo convention is `schema_version: 1` (cmd_replay.go:18) | Medium | Accept | Phase 1 |
| 13 | Clock-skew breaks 24h TTL — future-dated cache = fresh forever; backward jump = stale forever | Medium | Accept | Phase 1 |
| 14 | Dev-skip not whitespace/case-insensitive — `"dev "` bypasses | Medium | Accept | Phase 1 |
| 15 | `internal/version.Resolve()` shape contradiction within plan (Phase 1 prose says "three strings"; verified struct `Info{Version,Commit,Date}` at version.go:28-32) | Medium | Accept | Phase 1+3 |

**Below-cap rejections (logged, not applied):**
- `TYPEBURN_OFFLINE=1` env override — scope creep; user explicitly invokes `--check-update` knowing they want a network call.
- UA-strip-build-version privacy concern — low impact; opt-in default-OFF already mitigates fingerprinting.
- `DisallowUnknownFields` on JSON decoder — would reject GitHub's normal multi-field response.

### Whole-Plan Consistency Sweep

After applying findings, re-read every plan file to reconcile cross-phase claims:

- ✓ `app.New(...)` ctor change spec is consistent across Phase 4 and any phase mentioning callers (only Phase 4 mentions, fixed in finding 4).
- ✓ `Resolve()` accessor pattern unified — now both Phase 1 and Phase 3 use `version.Resolve().Version`.
- ✓ Dev-skip predicate spec is single-sourced — Phase 1 step 9 normalized; plan.md "Locked decisions" updated accordingly (see finding 10 + 14 application).
- ✓ Cache schema version (`schema_version: 1`) referenced consistently in Phase 1 + Phase 5 (docs).
- ✓ Phase 5 release-notes ordering precondition aligned with plan.md "Dependencies" (v2.0.0 tag must exist on GitHub).
- ✓ `update_check` config syntax: plan now standardizes on `true/false` (or extends parseBool to accept on/off per finding 3 decision) — verify docs in Phase 5 match the implementation in Phase 2.
- ✓ Inline supersession markers added at 4 highest-impact contradiction sites: Phase 1 Step 1 (atomic helper), Phase 1 Step 9 (dev-skip pseudocode), Phase 2 Step 4 (parseBool), Phase 4 Step 5 (caller list). Lower-impact prose (HTTP client snippet, TTL math, opportunistic timeout, release-notes wording) retains its original form for context but is fully superseded by the Red Team Review Updates sections at the end of each phase — implementers MUST read the red-team section before coding.

### Decisions locked post-red-team

- **Phase 2 finding 3 — `on/off` semantics:** resolution **(a) extend `parseBool`**
  to accept `on/off/yes/no` (locked by user 2026-05-21). Phase 2 implementation
  must update both the new `update_check` set path AND the existing `blink_cursor`
  set path (they share the parser).

No unresolved contradictions remain.
