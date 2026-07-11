---
date: 2026-07-11
topic: "Typeburn v2.5.0 consolidated release planning"
status: planned
plan: "plans/260711-1434-v2-5-consolidated-release-recovery/plan.md"
---

# Typeburn v2.5.0 Consolidated Release Planning

## Context

Public stable remains `v2.4.1`. Strict Mode is merged but unreleased, while
punctuation/numbers remains in PR #57. The session aligned code display fixes,
documentation truth, integration, and release recovery into one deep plan.

## What Happened

- Accepted one consolidated `v2.5.0`; no separate or phantom `v2.6.0` release.
- Locked Code-mode display to exact `code` in History and Result. Stored rune
  count remains unchanged; non-empty unknown modes render raw and empty renders
  `unknown`.
- Defined pre-release docs truth: 8 themes, 8 persisted/CLI keys, 7 Settings UI
  rows, stable `v2.4.1`, and all upcoming work under `Unreleased`.
- Created a five-phase deep plan covering TDD fixes, docs reconciliation, PR #57
  integration, release preparation, disposable publish proof, and post-release
  verification.

## Reflection

The highest risks were release controls, not the small label bug. Red-team review
found that tag publication needs a trusted-`main` ancestry gate, disposable
prereleases must not receive `HOMEBREW_TAP_TOKEN`, and partial public publication
needs an explicit resumable recovery matrix. Real tags remain immutable: contain
bad output and fix forward instead of deleting or retagging `v2.5.0`.

## Decisions

- Extend PR #57 for code and pre-release docs; use a separate release-prep PR.
- Require a cleaned disposable-tag run before creating the annotated stable tag.
- Treat prerelease proof as GitHub-publish evidence only; stable Homebrew writes
  require permission preflight and rollback readiness.
- Keep the existing untracked punctuation/numbers journal untouched and unstaged.
- Add no storage migration, dependency, signing, or unrelated cleanup.

## Validation

Deep validation checked 78 claims: 77 passed initially and one tool-availability
failure was resolved by planning a temporary exact GoReleaser `v2.15.4` install.
Final verification has zero remaining failures, zero unresolved contradictions,
and zero open questions. The plan remains `pending`; no implementation, merge,
tag, or release occurred in this session.

## Next

Execute the plan sequentially from Phase 1 through Phase 5, preserving protected
`main`, exact-SHA evidence, immutable stable tags, and the documented recovery
boundaries.
