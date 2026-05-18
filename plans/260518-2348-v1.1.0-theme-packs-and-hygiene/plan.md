---
title: v1.1.0 — Theme Packs + Hygiene
description: >-
  6 theme packs + bundled hygiene fixes; parallel phases, per-phase commits,
  protected-main PR flow
status: pending
priority: P2
branch: feat/v1.1.0-theme-packs-hygiene
tags:
  - theme
  - hygiene
  - release
blockedBy: []
blocks: []
created: '2026-05-18T16:50:50.539Z'
createdBy: 'ck:plan'
source: skill
---

# v1.1.0 — Theme Packs + Hygiene

## Overview

Minor release: 6 new theme packs (Solarized Dark/Light, Dracula, Nord,
Gruvbox Dark/Light) + bundled hygiene fixes (stale docs, silent-persistence
notice, misleading word-wrap comment, core-import doc rule). New feature ⇒
**v1.1.0** (semver minor), not a patch.

Source of truth: [brainstorm-summary.md](./brainstorm-summary.md) (audit
validation, architecture, locked decisions: 6 packs + DRY Approach A).

## Execution model

Protected-main (hard-enforced): all phases commit to feature branch
`feat/v1.1.0-theme-packs-hygiene` → PR → squash-merge → tag on merged SHA.
**Each phase = its own commit.** Phases 2/3/4 are **independent disjoint work
units** developed in parallel but **committed sequentially** onto the single
shared branch — file ownership is disjoint so sequential commits are
conflict-free; do NOT run concurrent writers in one working tree (no
worktrees needed; the task is small and disjoint). 5 joins them; 6 releases.

**CHANGELOG.md ownership:** among the parallel set, only **Phase 4** writes
`CHANGELOG.md` (it authors a single `[Unreleased]` block covering theme
packs + persistence notice + doc fixes). Phases 2 and 3 do **not** touch
`CHANGELOG.md`. Phase 6 (sequential, after 5) promotes `[Unreleased]` →
`[1.1.0]` — no parallel conflict since 6 runs after 4.

| Phase | Name | Status | Parallel group | Owns (file ownership) |
|-------|------|--------|----------------|----------------------|
| 1 | [Branch Setup](./phase-01-branch-setup.md) | Pending | — (gate) | branch only |
| 2 | [Theme Packs](./phase-02-theme-packs.md) | Pending | A ∥ | `internal/theme/*`, `internal/config/settings.go`, theme/settings tests, `docs/design-guidelines.md` |
| 3 | [Persistence Notice](./phase-03-persistence-notice.md) | Pending | A ∥ | `internal/app/model_history.go`, `internal/app/model_settings.go`, `internal/ui/persistence-notice.go` (new) + its test |
| 4 | [Docs Hygiene](./phase-04-docs-hygiene.md) | Pending | A ∥ | `README.md`, `docs/project-roadmap.md`, `docs/system-architecture.md`, `CLAUDE.md`, `CHANGELOG.md` (`[Unreleased]` block), `internal/ui/word_stream_renderer.go` (comment only) |
| 5 | [Integration Verify](./phase-05-integration-verify.md) | Pending | join | none (read+test only) |
| 6 | [Release v1.1.0](./phase-06-release-v1-1-0.md) | Pending | release | Completed |

**Dependency:** 1 → {2 ∥ 3 ∥ 4} → 5 → 6.

## Key locked decisions

- 6 packs, total 8 selectable (+ `default` + `mono`).
- **DRY Approach A:** keep `config/settings.go` hardcoded valid-theme switch
  (core layering forbids config→theme import); add a sync test asserting it
  matches `theme.Available()` + a code comment.
- NO_COLOR auto-inert (Load short-circuits before name switch) — no work.
- Light themes need explicit per-Role contrast review (Phase 2 step).

## Out of scope

Code mode, custom wordlists, history import/export, real word-aware wrap, CJK
width, parent-dir fsync (roadmap-accepted), online sync.

## Dependencies

No cross-plan deps — prior plans (`260518-1454`, `-2110`, `-2232`) all
completed.
