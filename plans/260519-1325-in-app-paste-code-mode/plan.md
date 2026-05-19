---
title: "In-app Paste for Code Mode (v1.3.0)"
description: "New ScreenCodePaste: supply the Code-mode snippet via in-TUI bracketed paste; codetext.Normalize refactor; TDD per phase, protected-main PR flow"
status: completed
priority: P2
branch: "feat/v1.3.0-in-app-paste"
tags: [feature, ui, tdd, release]
blockedBy: []
blocks: []
created: "2026-05-19T06:27:46.790Z"
createdBy: "ck:plan"
source: skill
---

# In-app Paste for Code Mode (v1.3.0)

## Overview

Add a 6th screen `ScreenCodePaste` so a Code-mode snippet can be supplied by
**bracketed paste inside the TUI** (no `--text` restart needed). Reuses the
v1.2.0 text-source seam (injected string, Home forks on availability) — no
engine/renderer change. Semver minor → **v1.3.0**.

Source of truth: [brainstorm-summary.md](./brainstorm-summary.md) (all
decisions locked: new full Screen, paste-only, Code-row-Enter-when-empty
entry with `--text` precedence, stay-and-retry on invalid, Approach A
paste-sub-model owns `codetext.Normalize`, valid paste→Home-Code-enabled).

## Execution model

`--tdd`: each impl phase is tests-first (pin new behaviour + lock adjacent
existing behaviour → red → implement → green). Protected-main: feature branch
`feat/v1.3.0-in-app-paste` → per-phase commits → PR → squash-merge → tag on
merged SHA. Sequential.

| Phase | Name | Status | Depends | TDD focus |
|-------|------|--------|---------|-----------|
| 1 | [Branch Setup](./phase-01-branch-setup.md) | Completed | — | gate, no commit |
| 2 | [codetext Normalize Refactor](./phase-02-codetext-normalize-refactor.md) | Completed | 1 | Normalize/Load parity → refactor |
| 3 | [ScreenCodePaste Sub-model](./phase-03-screencodepaste-sub-model.md) | Completed | 2 | paste→Normalize, waiting/error/esc states → impl |
| 4 | [Wiring + Routing + Home](./phase-04-wiring-routing-home.md) | Completed | 3 | Screen enum/routing/PasteMsg-route/CodePastedMsg/Home-row tests → impl |
| 5 | [Integration Verify](./phase-05-integration-verify.md) | Completed | 4 | full -race, goldens unchanged, tester+review |
| 6 | [Release v1.3.0](./phase-06-release-v1-3-0.md) | Completed | 5 | CHANGELOG/PR/dry-run/tag |

**Dependency:** 1 → 2 → 3 → 4 → 5 → 6.

## Key locked decisions

- New full `ScreenCodePaste` (6th `Screen`; own sub-model+view+routing).
- Bracketed-paste only — capture one `tea.PasteMsg`; no cursor/typing/edit.
- Entry: Home Code row Enter **when `codeText==""`** → `NavCodePasteMsg`.
  `--text` snippet still wins (Code enabled → Enter starts; paste not
  offered that run).
- Invalid paste → stay on screen, show codetext reason, retry/esc.
- **Approach A:** paste sub-model owns `codetext.Normalize` + error/retry
  state; emits `CodePastedMsg{Text}` only when valid. App stays thin.
  `tea.PasteMsg` is `struct{Content string}` — read `msg.Content` (F1).
- Valid paste → app sets `m.codeText`, clears `m.codeHint`, applies it to
  the **existing** Home via `HomeModel.WithCodeText` (NOT a `NewHome`
  rebuild — that resets the selector to DefaultMode and loses the Code row,
  F3), screen→Home with Code still selected; user presses Enter to start.
- `esc` on the paste screen → Home, `codeText` unchanged, handled by the
  **existing global Back handler** (`model_key_handler.go:64` else→Home).
  No cancel message, no sub-model esc handling (F2 — removed over-design).
- F4: confirm bracketed paste is enabled (Bubble Tea v2 default or explicit
  `tea.NewProgram` option in `main.go`) before relying on `PasteMsg`.
- `codetext`: refactor so `Load(path)` and new exported
  `Normalize(string)(string,error)` share `loadReader`; identical
  rules/caps/sentinels; `Load` behaviour unchanged (regression-locked).

## Out of scope

In-paste editing/cursor/typed input, multi-snippet library, syntax
highlighting, language detection, file-picker UI, replacing a `--text`
snippet from inside the TUI.

## Dependencies

No cross-plan deps — v1.1.0/v1.2.0 shipped (their plan.md top-level
`status` frontmatter is stale-cosmetic only); this extends already-merged
`internal/codetext`.
