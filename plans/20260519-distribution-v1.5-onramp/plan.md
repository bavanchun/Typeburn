---
title: "Distribution v1.5 — broadly-installable on-ramp"
description: >-
  Close the install-friction gap: confirm/lower the Go floor (verified bounded
  at 1.25 by deps), ship a hardened install.sh, add a Homebrew CASK via
  GoReleaser. App quality is ship-ready; the on-ramp is not. TDD per phase,
  protected-main PR flow. Hardened by red-team 2026-05-19.
status: completed
priority: P1
branch: "feat/distribution-v1.5-onramp"
tags: [release, distribution, goreleaser, homebrew, ci, tooling]
blockedBy: []
blocks: []
source: skill
---

# Distribution v1.5 — broadly-installable on-ramp

## Overview

Typeburn app quality is ship-ready. The **on-ramp** is the weak link:
`go install` needs Go 1.26.2, and the only fallback is a 5-step manual binary
download. No package managers.

**Reach thesis (corrected by red-team C2):** the Go-floor drop is NOT the
reach driver. Deps `charm.land/bubbletea/v2@v2.0.6` and `lipgloss/v2@v2.0.3`
both hard-pin `go 1.25.0` (verified, their `go.mod:5`), so the floor is bounded
at **1.25** — a 1.26.2→1.25 move, marginal. **WS2 (install.sh) + WS3 (Homebrew
cask) carry the reach.** WS1 is hygiene, not headline.

Target = "broadly installable for terminal users" (the TUI form-factor ceiling
excluding non-terminal users is accepted, out of scope).

Source brainstorm: [brainstorm-summary.md](./brainstorm-summary.md)

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [WS1 Go Floor Confirm + Lockstep](./phase-01-ws1-go-floor-spike-lockstep.md) | Complete |
| 2 | [WS2 Hardened install.sh + GoReleaser determinism](./phase-02-ws2-checksum-verified-install-sh.md) | Complete |
| 3 | [WS3 Homebrew Cask via GoReleaser](./phase-03-ws3-homebrew-tap-via-goreleaser.md) | Complete |
| 4 | [Docs Sync + Safe Release Dry-Run](./phase-04-docs-sync-release-dry-run.md) | Complete |

## Completion

**Shipped v1.5.0 — 2026-05-20** (tag `598f7ec` → commit `ac229c0`).
All 4 phases delivered, verified, code-reviewed (0 Critical/High).
PR #12 (feature) + PR #13 (doc sync) squash-merged to main.

- P1: Go floor 1.26.2→1.25.0, suite + `goreleaser build` green under an
  installed go1.25.10 toolchain; 5-file lockstep consistent; README
  caveats verbatim-preserved.
- P2: `.goreleaser.yaml` determinism pinned (`typeburn`); hardened POSIX
  `install.sh` + 14-case offline harness; shellcheck clean across
  versions; CI `installer` job added.
- P3: `homebrew_casks:` + `prerelease:auto` + `skip_upload:auto`;
  before-hook go-build removed; `HOMEBREW_TAP_TOKEN` step-env-only.
  G0 (burned token revoked) + G1 (fresh scoped PAT) completed by user.
- P4: safe disposable `v0.0.0-rc.test` dry-run passed every C1
  invariant (prerelease-flagged, 7 assets, zero tap commits,
  `releases/latest` unaffected) + full teardown; real v1.5.0 published
  (7 assets, 1106-char notes, cask `5c42971` committed to tap);
  BOTH channels clean-container verified (install.sh one-liner +
  `brew install bavanchun/tap-typeburn/typeburn` → `typeburn v1.5.0`).

## Build Order & Dependencies

- **P1 ∥ P2 run in parallel** (independent; red-team M1). P2 no longer blocks
  on P1 — the only coupling was the README Go-floor numeral, reconciled in P4's
  existing whole-plan wording sweep.
- **P3 depends on P2** (P2 pins `.goreleaser.yaml` determinism — `builds.binary`
  + `archives.name_template` + `release.prerelease: auto` — that P3's cask and
  P4's safe dry-run both rely on).
- **P4 depends on P3** and is the **hard gate before the real v1.5 tag** — no
  real tag until P4's safe dry-run passes every invariant.
- No cross-plan dependency: existing pending plans touch `internal/app/*`; this
  plan touches `go.mod`, `.goreleaser.yaml`, `.github/`, `install.sh`, docs.

## Locked Decisions (brainstorm + red-team adjudication)

- **Homebrew = `homebrew_casks:`** (NOT `brews:`). Wraps the prebuilt release
  archive; no user Go/Xcode; matches committed `CONTRIBUTING.md:99`. (red-team H1)
- **Secret name = `HOMEBREW_TAP_TOKEN`** (aligns to `CONTRIBUTING.md:98`).
- **Tap repo = `bavanchun/homebrew-tap-typeburn`** (user-provisioned this
  session — authoritative; `CONTRIBUTING.md` example updated to match).
  User command: `brew install bavanchun/tap-typeburn/typeburn`.
- **`.goreleaser.yaml builds.binary: typeburn`** pinned (P2) — deterministic
  lowercase binary; eliminates the `Typeburn`/`typeburn` split AND the cask
  rename guesswork (red-team H2/C3).
- **Go floor = 1.25** (verified bounded by deps; confirm-not-spike).
- install.sh target: `~/.local/bin`, no sudo, `mkdir -p`, atomic install,
  warn on PATH absence/precedence.
- OUT this round: Scoop/winget, AUR, deb/rpm (nfpms), Nix, snap.

## Critical Constraints

- Release pipeline invariants: `assets == 7` (`release.yml:75-88`),
  SHA-pinned actions, GoReleaser pinned `v2.15.4`, least-privilege jobs,
  curated release notes. The cask commits to the tap repo, NOT a release asset
  — count stays 7 (must be verified pre-tag, not assumed; red-team H9).
- **Go-version lockstep — 5 files / 6 occurrences**: `go.mod:3`,
  `.github/workflows/ci.yml`, `.github/workflows/release.yml` (×2),
  `CONTRIBUTING.md`, `README.md:25`.
- **Safe dry-run (red-team C1, the load-bearing fix):** `release.yml:14-17`
  triggers on `v*`; the project disposable tag `v0.0.0-rc.test` matches it and
  `.goreleaser.yaml:71 draft: false`. Without mitigation, P4's "dry-run"
  publishes a REAL public release + live cask. Mitigation: `release.prerelease:
  auto` + cask `skip_upload: auto` + install.sh prerelease-reject guard + P4
  asserts zero tap commits during dry-run.
- Protected-main: branch → PR → squash-merge. Never commit to main.
- PAT: chat-pasted token is BURNED — two independent gates (G0 verify dead,
  G1 provision fresh scoped fine-grained PAT). Never committed/echoed.

## Open Questions

- None blocking. WS1 outcome is NOT open — floor is verified 1.25.
  GoReleaser default project/binary name is made moot by pinning
  `builds.binary: typeburn` in P2.

## Red Team Review

### Session — 2026-05-19
**Findings:** 14 (14 accepted, 0 rejected) — evidence-filtered, deduplicated
from 3 hostile reviewers (Security Adversary, Failure Mode Analyst, Assumption
Destroyer), all `file:line`-backed.
**Severity breakdown:** 3 Critical, 9 High, 2 Medium.

| # | Finding | Severity | Disposition | Applied To |
|---|---------|----------|-------------|------------|
| C1 | Disposable `v*` dry-run publishes real public release + live cask | Critical | Accept | P3, P4, plan |
| C2 | Go-floor thesis dead — deps pin 1.25; confirm not spike | Critical | Accept | P1, plan |
| C3 | install.sh hardcodes archive name GoReleaser doesn't produce | Critical | Accept | P2 |
| H1 | Plan overrode CONTRIBUTING.md spec (casks/secret/repo names) | High | Accept | P3, plan |
| H2 | Brew binary-name assumption unverified (no builds.binary) | High | Accept | P2, P3 |
| H3 | install.sh `releases/latest` poisoned by dry-run prerelease | High | Accept | P2, P3 |
| H4 | No rollback path for a bad tap cask | High | Accept | P3, P4 |
| H5 | PAT revocation is a comment, not an enforced gate | High | Accept | P3 |
| H6 | PAT blast radius + cask token wiring (default = wrong token) | High | Accept | P3 |
| H7 | Floor unvalidated under settled toolchain (snapshot false-green) | High | Accept | P1 |
| H8 | install.sh hardening: pipefail/mktemp/symlink/atomic/mkdir/PATH | High | Accept | P2 |
| H9 | `assets==7` post-publish & untestable by snapshot | High | Accept | P3, P4 |
| M1 | P2 falsely serialized behind P1 | Medium | Accept | P2 frontmatter, plan |
| M2 | Lockstep README rewrite drops caveats; checksum oversold | Medium | Accept | P1, P2 |

### Whole-Plan Consistency Sweep
- Files reread: plan.md, phase-01..04 (post-edit).
- Decision deltas checked: brews→homebrew_casks; secret→HOMEBREW_TAP_TOKEN;
  tap→homebrew-tap-typeburn; floor unknown→1.25 verified; P2 deps [1]→[];
  binary→pinned `typeburn`; dry-run→prerelease:auto+skip_upload:auto.
- Reconciled stale references: all `brews:`/`HOMEBREW_TAP_GITHUB_TOKEN`/
  "spike from 1.22"/"outcome unknown"/`Typeburn`-rename strings removed from
  phase files and plan.
- Unresolved contradictions: 0.

## Validation Log

### Session — 2026-05-19
Verification pass skipped: `## Red Team Review` already carries full
`file:line` evidence; no `[UNVERIFIED]` tags. 3 decision points resolved.

| Topic | Decision | Effect |
|-------|----------|--------|
| README install framing (red-team M2) | **curl\|sh primary + honest trust-boundary caveat + non-piped audit path beside it** | Confirms `phase-02` step 5 as written — no change |
| Artifact signing scope (red-team Sec#1) | **Defer to a later version** — v1.5 = honest docs only | Confirms current scope; tracked as follow-up below |
| `go.mod` toolchain pinning | **Resolved by existing convention** — bare `go` directive, no `toolchain:` line; CI floating `go-version: "1.25.x"` (matches current repo) | `phase-01` step 5 unchanged |

**Deferred follow-up (post-v1.5):** independent signing of `checksums.txt`
(cosign keyless or minisign) so artifact integrity is verifiable independent
of the GitHub release trust root. Out of v1.5 scope by user decision; v1.5
docs must not overclaim checksum verification (already enforced, `phase-02`).

### Whole-Plan Consistency Sweep
- Files reread: plan.md, phase-01..04 (post-validation).
- Decision deltas checked: 3 — all confirm existing plan direction; only
  additive change is this Validation Log + deferred-signing note.
- Reconciled stale references: 0 (no direction changed).
- Unresolved contradictions: 0.

## Dependencies

<!-- No cross-plan dependencies -->
