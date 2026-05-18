---
title: "v1.0.1 — new-best sub-WPM precision (M2) + remove dead MissedChars (m4)"
description: "Two independent, parallel-safe fixes for v1.0.1: float-precise new-best detection (M2) and removal of the always-zero MissedChars field (m4). Each phase self-commits."
status: completed
priority: P2
branch: "main"
tags: [v1.0.1, bugfix, refactor, parallel]
blockedBy: []
blocks: []
created: "2026-05-18T15:32:04.593Z"
createdBy: "ck:plan"
source: skill
---

# v1.0.1 — new-best sub-WPM precision (M2) + remove dead MissedChars (m4)

## Overview

Two roadmap items for the first fast-follow patch after v1.0.0:

- **M2 (MAJOR, user-visible):** new-best detection compares `Record.WPM`, an
  `int(math.Round(NetWPM))`. 75.4 and 75.0 both round to 75, so a strictly
  faster run does not earn the ★. Fix: persist float `NetWPM` and compare on it.
- **m4 (MINOR, API hygiene):** `metrics.Result.MissedChars` is hard-coded to 0
  (`compute.go:78`) and displayed as `missed 0` on every result screen. The
  package has no target to compute it from. Fix: **remove** the field and its
  display (KISS/YAGNI — plumbing the target through the engine is
  disproportionate for a patch release).

The two fixes share **zero files** (verified by scout) and are executed by
**parallel subagents** with strict file ownership. Each phase produces its own
commit. Phase 3 integrates, runs the full regression + release-pipeline gate,
and (on explicit user go-ahead) cuts `v1.0.1`.

## Phases

| Phase | Name | Status | Parallel group |
|-------|------|--------|----------------|
| 1 | [M2 sub-WPM new-best precision](./phase-01-m2-sub-wpm-new-best-precision.md) | Completed | A (parallel with 2) |
| 2 | [m4 remove dead MissedChars field](./phase-02-m4-remove-dead-missedchars-field.md) | Completed | A (parallel with 1) |
| 3 | [Integration regression and release gate](./phase-03-integration-regression-and-release-gate.md) | Completed | B (after 1 & 2) |

## File Ownership (parallel safety — non-negotiable)

| Phase 1 (M2) owns | Phase 2 (m4) owns |
|---|---|
| `internal/storage/history_record.go` | `internal/metrics/compute.go` |
| `internal/storage/new_best.go` | `internal/metrics/compute_test.go` |
| `internal/storage/history_store_test.go` | `internal/ui/screen_result_view.go` |
| `internal/app/model_history.go` | `internal/ui/screen_result_test.go` |
| `internal/ui/history_table.go` | `internal/ui/test_helpers_test.go` |

Both phases edit files inside the `internal/ui` package but **different files**;
no symbol is shared between the two changes (M2 touches `storage.Record`
plumbing; m4 removes a `metrics.Result` field). The package compiles
independently per file. If a subagent finds it must edit a file the other owns,
STOP and escalate — do not cross the boundary.

## Locked Decisions

- M2: add `NetWPM float64 \`json:"net_wpm"\`` to `storage.Record`; keep `WPM int`
  for compact display/JSON back-compat. Comparisons switch to effective NetWPM.
- **M2 legacy back-compat rule (critical — roadmap under-specified this):**
  records written by v1.0.0 have no `net_wpm` key → unmarshal to `0.0`.
  Comparing raw `NetWPM` would let any new run beat a legacy 80-WPM record
  (0.0). Effective value = `if rec.NetWPM == 0 { float64(rec.WPM) } else
  { rec.NetWPM }`. Apply this in BOTH `storage.IsNewBest` and the history-table
  ★ calc so old and new records compare on the same scale.
- m4: delete the field, not compute it. No engine/target plumbing.
- Each phase ends with exactly one conventional commit (no AI refs):
  Phase 1 `fix:`, Phase 2 `refactor:`, Phase 3 `test:`/release commits.
- `v1.0.1` release is **fix-forward** off `main`; reuse the proven v1.0.0
  pipeline. Tag only on explicit user approval (irreversible, append-only sumdb).

## Dependencies

Phase 3 `blockedBy: [1, 2]`. Phases 1 and 2 are mutually independent and run
concurrently. No cross-plan dependencies (release-engineering-v1 is completed).

## Risk Summary

| Risk | Mitigation |
|---|---|
| Legacy `net_wpm:0` falsely wins new-best | Effective-value fallback rule (locked above); explicit test case |
| History golden tests shift if ★ row changes | Phase 1 re-runs/regenerates teatest goldens; uses distinct WPMs |
| m4 struct-literal removals miss a test ref | Phase 2 greps `MissedChars` repo-wide post-edit; build must be clean |
| Parallel package breakage | Strict file ownership; Phase 3 is the only compile-both-together gate |
| Releasing a bad patch (irreversible) | Phase 3 reuses disposable-tag dry-run before the real `v1.0.1` tag |
