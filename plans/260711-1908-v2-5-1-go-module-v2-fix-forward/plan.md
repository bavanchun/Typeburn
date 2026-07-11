---
title: "v2.5.1 Go module v2 fix-forward"
description: "Repair the broken v2 Go-install channel without mutating v2.5.0."
status: completed
priority: P1
branch: "fix/v2-module-path"
tags: [release, go-modules, corrective]
blockedBy: []
blocks: [260711-1434-v2-5-consolidated-release-recovery]
created: "2026-07-11T12:08:26.317Z"
createdBy: "ck:plan"
source: skill
---

# v2.5.1 Go module v2 fix-forward

## Overview

Publish corrective `v2.5.1` as a valid Go v2 module, preserve lowercase
`typeburn`, reconcile install guidance, and prove the fix through the public Go
proxy. Keep the published `v2.5.0` tag and release untouched.

## Locked Decisions

- Module path: `github.com/bavanchun/Typeburn/v2` at repository root.
- Command package: `cmd/typeburn`; canonical install is
  `go install github.com/bavanchun/Typeburn/v2/cmd/typeburn@latest`.
- Repository, release, raw-content, API, and security URLs do not gain `/v2`.
- Fix forward as `v2.5.1`; never delete, move, recreate, or rerun `v2.5.0`.
- Keep workflow files byte-identical unless implementation proves a coverage gap.
- No features, dependency changes, storage migration, signing, or unrelated cleanup.
- Preserve the user's untracked punctuation/numbers journal byte-for-byte.

## Phases

| Phase | Name | Status |
|---|---|---|
| 1 | [Lock module and command contract](./phase-01-lock-module-and-command-contract.md) | Done |
| 2 | [Migrate module imports and command entrypoint](./phase-02-migrate-module-imports-and-command-entrypoint.md) | Done |
| 3 | [Reconcile install documentation and release contracts](./phase-03-reconcile-install-documentation-and-release-contracts.md) | Done |
| 4 | [Prepare protected corrective release](./phase-04-prepare-protected-corrective-release.md) | Done |
| 5 | [Publish and verify v2.5.1 fix-forward](./phase-05-publish-and-verify-v2-5-1-fix-forward.md) | Done |

## Dependencies

- Sequential: Phase 1 → 2 → 3 → 4 → 5.
- Blocks `260711-1434-v2-5-consolidated-release-recovery`.
- Phase 5 requires the squash-merged, CI-green SHA frozen in Phase 4.
- Disposable archive proof and stable public-proxy proof are separate gates.

## Scope Challenge

**HOLD.** Minimum repair: module/import migration, command relocation,
linker/release config, exact runtime/docs commands, regression tests, and one
fix-forward release. Broad replacement is forbidden because valid GitHub URLs
must not change. Five phases preserve the immutable publish boundary.

## Whole-Plan Success Criteria

- Module identity is `/v2`; no tracked Go file uses the old internal import prefix.
- Local and release outputs are lowercase `typeburn` with correct v2.5.1 metadata.
- Full format, vet, race, size, and snapshot gates pass.
- Runtime guidance and current docs use `/v2/cmd/typeburn`; historical v2.5.0
  facts and non-module GitHub URLs remain intact.
- Protected PR and disposable archive release prove the exact merged SHA,
  seven assets, checksums, latest/tap isolation, and idempotent cleanup.
- Stable v2.5.1 succeeds across GitHub, installer, Homebrew, updater, and a clean
  proxy-only exact and `@latest` install producing `typeburn` v2.5.1.
- Old recovery plan closes as superseded/recovered, never as a false statement
  that v2.5.0 passed its Go-install criterion.

## Research

- [Go v2 command contract](./research/go-v2-command-contract.md)
- [Migration impact inventory](./research/migration-impact-inventory.md)
- [Release recovery](./research/release-recovery.md)
- [Phase 1 RED contract proof](./reports/phase-01-red-contract-proof.md)

## Red Team Review

- Three hostile reviews covered Go semantics, implementation/test completeness,
  and release immutability. All evidence-backed findings were accepted.
- Critical amendments: archive-only unique v0 disposable tag; exact public-proxy
  environment and module metadata proof; Homebrew rollback readiness; partial
  publish containment; exact tag/SHA/run evidence; workspace stage allowlist.
- Precision amendments: correct moved-main invariant, tidy/verify/root-command
  guards, exact archive matrix, scoped documentation scans, and supported
  cross-plan completion semantics.
- Rejected findings: none. No reviewer supplied evidence requiring reversal of
  the user's `/v2` or v2.5.1 fix-forward decisions.

## Validation Log

- **Requirements:** PASS — module path, lowercase command, docs, and immutable
  fix-forward map directly to user decisions.
- **Architecture:** PASS — root v2 module plus `cmd/typeburn` is valid and the
  smallest layout that preserves command identity.
- **Implementation:** PASS — exact file groups, 279 imports/133 files, linker,
  runtime, tests, and docs surfaces are inventoried.
- **Testing:** PASS — red contract, module hygiene, full local gates, archive
  workflow proof, and two clean public-proxy installs cover failure layers.
- **Release/Security:** PASS — protected merge, tag immutability, credential
  readiness, containment, and fix-forward boundaries are explicit.
- **Consistency sweep:** PASS after corrective proof — 5 phases, 0 broken
  dependencies, dedicated Phase 1 evidence commit, Phase 2 migration commit,
  Phase 3 documentation commits, and no criterion depends on mutating v2.5.0.

## Rollback Boundary

Before stable tagging, repair the corrective PR. After stable mutation, never
move `v2.5.1`; use `v2.5.2` for any further correction. Treat proxy state as
append-only.

## Completion Evidence

- Protected PR: #59, squash merge `8307ee6c9384dceb27aaa639bdac980b43906e0b`.
- Main CI: run `29158912514`, success.
- Disposable archive proof: run `29158988295`, seven assets verified, latest
  and tap isolated, release/tag cleanup verified.
- Stable release: annotated `v2.5.1` peels to the merged SHA; run
  `29159099750` succeeded with seven assets.
- Channels: pinned installer, Homebrew tap commit `eb517cdac4febeb3cbdf5496618edd46c7b15c73`,
  updater discovery, and public proxy-only exact/latest installs all verified.
