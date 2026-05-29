---
title: "Notui Time-Mode Auto-End + Defect Cleanup (v2.1.3)"
description: "Fix the --no-tui Time-mode loop that never auto-ends at the time limit (ship-blocker on the default run path), close the notui runner test gap, validate update.Check ReleaseURL on the forced path, and minor hygiene (comment + model.go LOC split). effWPM zero-sentinel reviewed and documented as accepted — no schema change."
status: completed
priority: P1
branch: "fix/notui-time-mode-auto-end"
tags: [bugfix, notui, cli, release]
blockedBy: []
blocks: []
created: "2026-05-29T08:49:51.203Z"
createdBy: "ck:plan"
source: skill
---

# Notui Time-Mode Auto-End + Defect Cleanup (v2.1.3)

## Overview

Brainstorm source: `plans/reports/20260529-defect-scan-brainstorm-summary.md`.

Deep code-level defect scan of Typeburn @ v2.1.2 found **one ship-blocker** and minor cleanups. Codebase is otherwise clean (`-race` green, gofmt/vet clean, 0 TODO, no outdated direct deps).

**Ship-blocker (confirmed, reachable via default path):** `typeburn run --no-tui` defaults to Time mode (`settings.DefaultMode`); the notui `runLoop` blocks on `ReadEvent` every iteration and only checks completion *after a keystroke*, so a Time test never auto-ends at the limit — it hangs until the next keypress (or forever in pipe/script). A late keystroke also corrupts result metrics. The notui package has **no `runner_test.go`** (the 52%-coverage gap), so this path was never exercised.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Notui Time-Mode Auto-End](./phase-01-notui-time-mode-auto-end.md) | Completed |
| 2 | [Update URL Validation](./phase-02-update-url-validation.md) | Completed |
| 3 | [Hygiene Cleanup](./phase-03-hygiene-cleanup.md) | Completed |

Phases are independent (no inter-phase code dependency); ordered by severity. Phase 1 is mandatory; 2-3 are low-risk follow-ons that can ship in the same PR.

## Key Constraints (from CLAUDE.md)

- **Strict layering:** notui lives in `internal/cli/notui`; must not pull UI (`bubbletea`/`lipgloss`) into pure-logic packages. Time logic already lives in `runner.Session` / `metrics` — keep it there.
- **Determinism in tests:** the existing `now func() time.Time` seam in `runLoop` must be preserved/extended so the new timer path is testable without real sleeps. No wall-clock flakiness.
- **File size:** keep every Go file < 200 LOC.
- **`-race` green, gofmt empty, `go vet` clean** are the CI gate — all three must pass.
- **Protected main:** branch `fix/notui-time-mode-auto-end` → PR → squash-merge. No direct commit to main.

## Out of Scope

- TUI Time-mode path (already correct via `tea.Tick`).
- Dependency upgrades (only indirect updates pending; no action).
- effWPM schema change — see Phase 3 (reviewed, intentionally NOT changed).
- New features / backlog (Vim motions, custom wordlists, CJK width m5).

## Success Criteria (whole plan)

- `typeburn run --no-tui` (Time/30s) auto-ends at the limit with **zero trailing input**.
- No late-keystroke inflation in the trailing-path result metrics.
- New deterministic `runner_test.go` covers Time-mode auto-completion; full suite `-race` green.
- `update.Check` rejects a non-prefixed `ReleaseURL` on the forced path too.
- `completion.go` comment matches the position-based logic; `app/model.go` < 200 LOC.
- gofmt empty, `go vet` clean.

## Dependencies

None. No unfinished plans overlap (all prior plans completed).
