---
title: Settings Live-Apply Fix + Typing Width
description: >-
  Fix theme/settings live-apply (callback+pointer bound to orphaned New() copy →
  message refactor) and make the typing stream %-width + vertically centered.
  TDD per phase, protected-main PR flow.
status: pending
priority: P1
branch: fix/v1.4.0-settings-live-apply-and-typing-width
tags:
  - bugfix
  - ui
  - refactor
  - tdd
  - release
blockedBy: []
blocks: []
created: '2026-05-19T09:33:10.268Z'
createdBy: 'ck:plan'
source: skill
---

# Settings Live-Apply Fix + Typing Width

## Overview

Two user-reported defects: one root-caused bug + one UX layout change.
Source of truth: [brainstorm-summary.md](./brainstorm-summary.md) (design
approved; Fix #2 = message refactor, Fix #1 = %-based width + vertical
center, both confirmed by user).

**Fix #2 (P1 bug):** changing Theme (also Blink/Default-mode, and the
`persistErr` toast) in Settings has no visible effect in-session. Root cause
(~95%): `app.New()` (`internal/app/model.go:75`) binds the pointer-receiver
method value `m.onSettingsChange` and `&m.settings` to `New`'s **local** `m`,
then `return m` copies the struct. `tea.NewProgram` drives the copy; the
callback/pointer mutate the orphaned original. Fix by converting to the
codebase's existing message pattern (`AbortMsg`/`StartTestMsg` style).

**Fix #1 (UX):** on wide terminals the typing stream is hard-capped at 80
cols (`width_tier.go:42`) and top-anchored with a full-height bottom spacer
(`screen_typing_view.go:66-77`) → text feels small/lost. Fix: TierWide
content width becomes `clamp(round(termW*0.82), 80, termW-8)` (never narrower
than today, grows on big screens), and the typing view stops filling full
height so the root's existing `lipgloss.Place(Center,Center)` actually
centers it vertically.

## Execution model

`--tdd`: each impl phase is tests-first — pin new behaviour + lock adjacent
existing behaviour (RED) → implement (GREEN). `phase09_polish_test.go:43`
(`ContentWidth(100,TierWide)==80`) is **intentionally rewritten** in Phase 3
to the new contract — a deliberate, called-out behaviour change, never a
bypass. Protected-main: feature branch → per-phase commits → PR →
squash-merge → tag on merged SHA (per `typeburn-release-runbook`). Sequential.

## Phases

| Phase | Name | Status | Depends | TDD focus |
|-------|------|--------|---------|-----------|
| 1 | [Branch Setup](./phase-01-branch-setup.md) | Pending | — | Completed |
| 2 | [Settings Live-Apply Refactor](./phase-02-settings-live-apply-refactor.md) | Pending | 1 | Completed |
| 3 | [Typing Width Bounded Fill](./phase-03-typing-width-bounded-fill.md) | Pending | 2 | rewrite ContentWidth/layout contract → impl |
| 4 | [Integration Verify](./phase-04-integration-verify.md) | Pending | 3 | full -race, tester + code-reviewer |
| 5 | [Release v1.4.0](./phase-05-release-v1-4-0.md) | Pending | 4 | CHANGELOG/PR/dry-run/tag |

**Dependency:** 1 → 2 → 3 → 4 → 5.

## Key locked decisions

- **Fix #2 = message refactor** (not minimal patch). `SettingsModel` drops
  `s *config.Settings` + `onChange func(...)`; holds settings by value;
  emits `SettingsChangedMsg{Settings}` as a `tea.Cmd`. Root `Model.Update`
  handles it on the live `m`. Pointer-receiver `onSettingsChange` deleted.
- Preserve `SettingsModel.sel` across the post-change rebuild (current code
  resets it to row 0 — fix that wart in the same change).
- Same fix restores live Blink-cursor, live Default-mode/length, and the
  in-session `persistErr` toast (all share the broken path) — verify each.
- Replace the false `smoke_test.go:189-196` rationalisation comment; add a
  **real** regression test asserting `m.(Model).theme` changes AND `View()`
  output differs after a theme cycle (not just the "mono" string in the
  settings view).
- **Fix #1 width contract:** `ContentWidth(termW, TierWide) =
  clamp(round(termW*0.82), 80, termW-8)`. Floor 80 = never regress below the
  old cap (note `0.82*88≈72 < 80` at the wide boundary). Narrow/Mid/Degraded
  tiers and `WidthTier` classification are **unchanged**.
- **Fix #1 centering:** remove the full-height bottom-spacer fill in
  `screen_typing_view.go`; emit a compact block so the root
  `model_view.go:50-53` `lipgloss.Place(m.w,m.h,Center,Center)` centers it
  vertically. Footer is no longer terminal-bottom-pinned (intended).
- No glyph-size change (terminal-owned; documented expectation).
- Version: **v1.4.0** (a visible non-breaking behaviour/UX change beyond a
  pure fix). Patch (v1.3.1) is the fallback if the user prefers at release;
  decide in Phase 5.

## Out of scope

Big ASCII "zen" glyph mode; per-screen width config; CJK width; any change
to Narrow/Mid/Degraded tiers; horizontal layout of non-typing screens;
keymap changes.

## Dependencies

No cross-plan deps. Pending-status v1.1.0/v1.2.0 plan frontmatter is
stale-cosmetic (shipped). This touches `internal/app/{model,model_settings}`,
`internal/ui/{screen_settings,messages,width_tier,screen_typing_view}` — no
overlap with other unfinished plans.
