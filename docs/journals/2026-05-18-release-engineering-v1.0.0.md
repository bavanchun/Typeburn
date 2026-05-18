# Release Engineering v1.0.0 — First Real Release

**Date**: 2026-05-18 21:10
**Severity**: High (one-way-door: tag published, sumdb immutable)
**Component**: Release Pipeline, Versioning, CI/CD
**Status**: Resolved

## What Happened

Typeburn v1.0 shipped to production (feature-complete, tested, documented) but was never *released* — no tags, no published binaries, version unknown at runtime. Today we cut `v1.0.0` with professional release infrastructure: hybrid version package (ldflags + fallback), GoReleaser v2.15.4 cross-platform pipeline, self-gating tag-triggered CI, curated CHANGELOG as release notes source, and repo hygiene templates. Full outcome: 7 release assets published, checksum-verified, `go install @v1.0.0` works, release notes rendered from CHANGELOG.md (1869 chars, not git log).

Commit: `9f96ff8` (tag pinned, never re-tag per sumdb immutability).
Release: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.0

## The Brutal Truth

The plan was extensively red-teamed (15 findings, 3 severities, across 3 reviewers — Security Adversary, Failure Mode Analyst, Assumption Destroyer). All findings were incorporated. We caught supply-chain edge cases (Node20→Node24 deprecation), rollback failure modes (sumdb append-only), and self-gating logic. Yet **the empirical dry-run still surfaced a self-contradictory mechanism the plan itself encoded**.

The real frustration: plan rigor does not guarantee empirical truth. An otherwise sound design decision (`changelog.disable: true` to suppress git log, only use `--release-notes`) was broken at the mechanism level — GoReleaser docs confirm `disable:true` **also ignores `--release-notes`**, publishing an empty release body. The dry-run caught this exactly as designed (a disposable pre-release tag). But it took actually exercising the publish path to expose what static analysis and domain review missed. No shame in the miss — GoReleaser's behavior is non-obvious — but it's humbling: you can red-team a plan into logical consistency and still ship broken infrastructure if you skip the empirical gate.

## Technical Details

**The disposable dry-run finding:**
- Phase 5 plan-locked decision: `changelog.disable: true` + pass `--release-notes` to GoReleaser to use curated CHANGELOG.md instead of git log.
- Pre-tag dry-run on a transient `v0.0.0-rc.test` tag revealed: release body was empty (length=1, just newline).
- GoReleaser docs (checked post-failure): "When changelog.disable is set to true, the changelog is also ignored when using the `--release-notes` flag."
- Root: the config mechanism was self-contradictory (intended: disable git log, use curated notes; actual: disable *everything*).

**Code review caught pre-tag asymmetry:**
- `go install`: reported `typeburn v1.0.0` (from BuildInfo + ldflags `version.Version=v1.0.0`).
- Release archives: reported `typeburn 1.0.0` (from GoReleaser `{{.Version}}`, which strips leading `v`).
- Fix: normalize to `version.Version=v{{.Version}}` in `.goreleaser.yaml` with explicit user approval (CLI surface, locked decision, append-only release).

**Non-blocking, flagged:**
- Actions SHA-pinned to Node20 runtime; GitHub deprecates Node20 on 2026-06-02. Deliberate trade-off (supply-chain integrity > freshness). Pins must bump before deadline per CONTRIBUTING.md.

## What We Tried

1. **Plan-level red-team** (15 findings, all accepted): caught logic errors, self-gate requirement, rollback immutability, ordering constraints, deprecated schema. Did not catch the `changelog.disable` + `--release-notes` interaction empirically.

2. **Dry-run before real tag** (Phase 5 design): created disposable `v0.0.0-rc.test` pre-release, ran full publish pipeline, verified asset count (7), checked release body. Second dry-run confirmed fix (body_len 1→1869). Then deleted the pre-release and annotated-tagged the proven SHA.

3. **Post-code-review fixup**: normalized version banner across install paths (v-prefix).

4. **Irreversibility discipline**: 
   - Fix-forward only (sumdb append-only — never re-tag same version).
   - Disposable tags cleaned (`--cleanup-tag` after dry-run).
   - Annotated tag on exact CI-green SHA (separate push, not `--follow-tags`).
   - No force-push.

## Root Cause Analysis

The plan locked a **mechanism** (`changelog.disable: true` + `--release-notes`) that sounded logically correct (disable git log, use curated notes) but violated GoReleaser's actual semantics. The conflict wasn't caught because:

1. **Static analysis blind spot**: red-team focused on orchestration logic (self-gating, asset count, ordering) and security (supply chain, workflow scope), not vendor-library semantics.
2. **Documentation gap**: GoReleaser's behavior (disable flag kills *both* sources) is clear in official docs but not intuitive.
3. **No substitute for empirical exercise**: "snapshot proves the pipeline" (Phase 2 design) is false for the *publish* path. Only actually publishing (dry-run gate) tested it.

The lesson is not that red-teaming failed — it caught 15 real issues. The lesson is that **verification through exercise (the disposable dry-run) is the only gate for irreversible operations**. A carefully thought-out plan can still encode contradictions; only the real path tests them.

## Lessons Learned

1. **Plan rigor ≠ empirical truth.** Red-teaming catches logic errors, security gaps, and ordering violations. It does not catch vendor-specific behavior surprises. For irreversible operations (immutable sumdb), empirical exercise of the real path is non-negotiable, even if the plan seems airtight. The disposable dry-run design (F10) was the only gate that mattered.

2. **Mechanism vs. intent.** The plan's *intent* (curated CHANGELOG only, no git log) was sound. The *mechanism* (`changelog.disable: true`) was broken. When fixing, preserve intent, replace mechanism. (Applied: removed `disable:true`, added `exclude: ['.*']` filter to suppress git log while leaving `--release-notes` active.)

3. **CLI surface is append-only.** User approved a v-prefix normalization (v1.0.0 both paths) pre-tag because the release is immutable after sumdb. Pre-tag fixups are free; post-tag CLI changes are debt. This sharpened the discipline: code review *before* tag, not after.

4. **Deprecation runways are real.** GitHub Node20→Node24 deadline (2026-06-02) is not theoretical; it's a hard cutoff. SHA-pinned actions will fail after that date. Flagged in plan but not urgent — it's a separate follow-up task. Document deadlines explicitly in CONTRIBUTING.

5. **Irreversibility discipline is procedural, not automatic.** Sumdb append-only and re-tag poisoning are facts. The discipline (disposable tags, annotated commit pin, separate push, no force) is a *choice*. Document it in runbooks so the next release doesn't have to re-learn it.

## Next Steps

1. **Immediate (non-blocking):** bump SHA-pinned action versions before 2026-06-02 GitHub Node20 deprecation. Task: update `.github/workflows/release.yml` checkout v5, setup-go v6, goreleaser-action v7 (or newer if available).

2. **For v1.0.1+ releases:** reference this journal (§ Lessons Learned) when planning the next release cycle. The dry-run gate and irreversibility discipline saved this release from corruption.

3. **Documentation:** CONTRIBUTING.md already notes the Homebrew deferred-schema and action-pin maintenance. Add a "Release Runbook" section explicitly covering:
   - Disposable dry-run on a transient pre-release tag, never commit it.
   - Annotated-tag only, on the CI-green commit SHA, pushed separately.
   - Never `--follow-tags`; sumdb is append-only; re-tag is unrecoverable.

4. **Codebase cleanliness:** All 5 release-engineering phases are in main, committed and merged. No debris, no dead schema, no commented-out brews block. Ready for the next lifecycle.
