---
title: monkeytype-tui implementation
description: >-
  Go terminal typing app (Monkeytype clone) on Bubble Tea v2 — Elm arch, pure
  metrics, XDG persistence.
status: pending
priority: P2
effort: ~46h
branch: main
tags:
  - go
  - tui
  - bubbletea
  - typing
blockedBy: []
blocks: []
created: '2026-05-18T07:54:36.734Z'
createdBy: 'ck:plan'
source: skill
---

# monkeytype-tui implementation

## Overview

Distraction-free terminal typing test (Monkeytype-style) in Go. 5 screens: Home, Typing, Result, Settings, History. Modes: Time / Words / Quote. Built incrementally — every phase ends compiling, running, and (where applicable) test-passing.

## Stack

Go 1.26 · Bubble Tea v2 + Lip Gloss v2 + Bubbles v2 (Charm, exact module path verified in Phase 1) · pure `internal/metrics` · XDG JSON persistence · teatest + GitHub Actions CI.

## Locked Decisions (digest)

- Elm arch; root model in `internal/app`, screen-enum routing → per-screen sub-models.
- Timer = `tea.Tick` wall-clock deltas (≈100ms metric sample, ≈250ms header repaint). Never tick-count.
- Metrics pure & post-hoc from per-keystroke log + per-second snapshots. Clock starts on first keystroke.
- Net=correct/5/min · Raw=all/5/min · Acc=100·correct/(correct+incorrect) FINAL state · Consistency=100·tanh(1−CV). AFK trailing-trim (>7s) Time mode only.
- Error handling: allow-continue + backspace ONLY. No stop-on-error, no toggle.
- Settings v1 (all functional): Theme, Default mode, Default length, Blink cursor. Auto-persist. Nothing else.
- Persistence: settings → XDG_CONFIG, history → XDG_DATA (cap 200 rotate). Atomic temp+rename; corrupt/missing → defaults, never crash.
- Theme = `map[Role]lipgloss.Color`; screens use roles only. Ship "default" dark + "mono" (greyscale/attribute-leaning) so the Theme setting is functional; `NO_COLOR` → attributes-only swap, layout unchanged. Add-a-theme = one map; "solarized-dark" reserved (not v1).
- Keybindings exactly per design §8, centralized in `internal/config`.
- Files <200 lines, kebab-case; rune-safe; Linux + macOS.

## Phases

| Phase | Name | Status | Depends |
|-------|------|--------|---------|
| 1 | [Scaffold theme & app skeleton](./phase-01-scaffold-theme-app-skeleton.md) | Pending | Completed |
| 2 | [Typing engine & metrics (TDD)](./phase-02-typing-engine-metrics-tdd.md) | Pending | Completed |
| 3 | [Words & embedded quotes](./phase-03-words-embedded-quotes.md) | Pending | Completed |
| 4 | [Typing test screen](./phase-04-typing-test-screen.md) | Pending | Completed |
| 5 | [Home screen](./phase-05-home-screen.md) | Pending | Completed |
| 6 | [Result summary screen](./phase-06-result-summary-screen.md) | Pending | Completed |
| 7 | [Settings & persistence](./phase-07-settings-persistence.md) | Pending | 6 |
| 8 | [History & persistence](./phase-08-history-persistence.md) | Pending | 6 |
| 9 | [Polish resize & NO_COLOR](./phase-09-polish-resize-no-color.md) | Pending | 7, 8 |
| 10 | [Tests teatest & CI](./phase-10-tests-teatest-ci.md) | Pending | 9 |

## Key Risks

1. Charm v2 module path / API drift — Phase 1 verifies via `go get`; escalate if v2 unresolvable.
2. Consistency coefficient (1−CV) upstream-uncertain — pinned as accepted v1 decision, table-tested.
3. Rune/multi-byte handling in word-stream + cursor — rune-iteration enforced, Unicode tests.
4. Resize / small-terminal partial paint — gated degraded mode on every screen.
5. teatest module path instability — verified in Phase 10, golden files tolerant.

## Reference Docs

- `plans/reports/researcher-01-bubbletea-architecture.md` — Charm v2 arch, timing, testing.
- `plans/reports/researcher-02-typing-metrics.md` — exact metric formulas + data model.
- `docs/design-guidelines.md` — theme §2.1, components §5, keybindings §8 (authoritative), layout §4.
- `docs/wireframe/mockups.md` — per-screen ASCII mockups.

## Dependencies

Each phase blocks on the previous. 2 enables 4 & 6; 3 enables 4; 7 & 8 need 6. No external/cross-plan deps.
