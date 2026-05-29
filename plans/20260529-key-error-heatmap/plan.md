---
title: "Per-Key Error Heatmap"
description: "Post-hoc per-key miss tally surfaced on the Result screen and in CLI JSON/table output. Pure replay over the existing keystroke log; computed on the fly, no persistence."
status: completed
priority: P2
branch: "main"
tags: [metrics, ui, cli, feature]
blockedBy: []
blocks: []
created: "2026-05-29T15:06:37.589Z"
createdBy: "ck:plan"
source: skill
---

# Per-Key Error Heatmap

## Overview

After each test, show the keys the user fumbled most — counting **every** wrong
keystroke against a target char (including ones later corrected via backspace),
case-folded, top N. Surfaces on the Result screen and in CLI JSON/table output.
Computed on the fly from the keystroke log; nothing persisted, no schema change.

**Why it fits:** the keystroke log already records `Target rune` + `Correct bool`
per event (`internal/typing/engine.go:11-16`). The heatmap is a pure tally over
that log — the same post-hoc replay pattern as `metrics.Compute`. It lives
entirely in pure-logic `internal/metrics` (no UI deps) and rides on the ephemeral
`metrics.Result` (never serialized to history), so "compute on the fly, no
persistence" is automatic.

**Design decisions (locked in brainstorm):**
- Miss = every wrong forward keystroke vs a real target (`Target != 0 && !Correct`), incl. corrected.
- Case-folded (`unicode.ToLower`): `a`/`A` merge.
- Top N = 8, sorted deterministically (misses desc → attempts desc → key asc).
- Surfaced on Result screen + CLI JSON/table.
- Not persisted to history (Result is ephemeral).

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Heatmap Logic](./phase-01-heatmap-logic.md) | Completed |
| 2 | [Result Screen Surfacing](./phase-02-result-screen-surfacing.md) | Completed |
| 3 | [CLI Output Surfacing](./phase-03-cli-output-surfacing.md) | Completed |
| 4 | [Docs and Verification](./phase-04-docs-and-verification.md) | Completed |

**Build order:** Phase 1 is the foundation (the `KeyMisses` field everything reads).
Phases 2 and 3 are independent of each other and both depend only on Phase 1.
Phase 4 closes out docs + the full CI gate.

## Dependencies

No cross-plan dependencies — all prior Typeburn plans are `completed`.

**External:** stdlib only (`unicode`, `sort`). No new module dependencies (honors
the allowed-deps constraint in CLAUDE.md).
