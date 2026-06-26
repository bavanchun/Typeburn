---
title: "Strict (stop-on-error letter) typing mode"
description: "Optional letter-strict mode: wrong key is blocked (cursor does not advance) but still counted; toggle via Settings + CLI config"
status: pending
priority: P2
branch: "main"
tags: [typing-engine, feature, tdd, handoff]
blockedBy: []
blocks: []
created: "2026-06-26T15:29:26.157Z"
createdBy: "ck:plan"
source: skill
---

# Strict (stop-on-error letter) typing mode

## Overview

Add an optional **letter-strict** typing mode. When `strict_mode` is on, a wrong
keystroke is **blocked** — the cursor does not advance past an incorrect letter;
the user must press the correct key to proceed. The blocked error is **still
recorded** so it counts toward accuracy. Default off (preserves current
allow-continue behavior). Toggle in the Settings screen and via `typeburn config`.

This is the UX-depth feature chosen after deep-scout revealed the originally
planned "Vim keybindings" was already shipped (see
`plans/reports/brainstorm-260626-2213-vim-keybindings-report.md`, superseded).

## Locked design decisions (from brainstorm interview)

1. **Semantic = letter.** Wrong key blocked; cursor frozen at the current
   position until the correct key is pressed. (Not word-level, not stop-test.)
2. **Errors still counted.** A blocked wrong key is logged (`Typed`=wrong rune,
   `Target` set, `Correct=false`) but does **not** advance the typed buffer.
3. **`strict_mode` is a bool** (off | letter-strict), applied to **all** modes
   (Time/Words/Quote/Code). Enum (`off|letter|word`) deferred — YAGNI.
4. **Toggle surfaces:** Settings screen row **and** `typeburn config get/set`,
   persisted in `settings.json`, backward-compatible (missing key → `false`).

## Central design challenge — accuracy (READ THIS FIRST)

`metrics.Compute` derives `Accuracy` from the **final per-position char state**
after replaying the log (`internal/metrics/compute.go:38` — "A char typed wrong
then corrected via backspace counts as Correct → 100%"). In letter-strict mode
the buffer can never hold an uncorrected wrong char, so **final-state accuracy
would always be ~100%**, making it meaningless and violating decision #2.

**Resolution (this plan):** add a mode-agnostic, additive
`KeystrokeAccuracy float64` to `metrics.Result` = `100 * correctForward /
totalForward` computed directly from the log (every non-backspace keystroke,
including blocked errors). Non-strict behavior is unchanged (existing `Accuracy`
field untouched). Strict runs display/store keystroke accuracy so errors are
honestly reflected. This keeps `internal/metrics` mode-agnostic — it always
computes the field; the UI/storage decide which to show for strict runs.

## Architecture summary

- **Pure-logic only at the core.** `internal/typing` gains a `strict` flag set at
  construction; `Apply` branches. `internal/metrics` gains the additive
  `KeystrokeAccuracy` field. No UI deps enter these packages.
- **Single construction point.** `typing.New` is called once, at
  `internal/runner/session.go:46`. Strict is threaded settings → session →
  engine. The UI input handler is unchanged (engine self-blocks); the UI simply
  re-renders from `States()`.
- **Surfaces.** Settings screen gets a fixed-index "Strict mode" row;
  `cmd_config.go` gets a `strict_mode` key mirroring `blink_cursor`.

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Engine strict-letter policy + keystroke-accuracy (TDD)](./phase-01-engine-strict-letter-policy-tdd.md) | Completed |
| 2 | [Settings + CLI config surface (TDD)](./phase-02-settings-cli-config-surface-tdd.md) | Completed |
| 3 | [Wire setting + Settings toggle + goldens (TDD)](./phase-03-wire-setting-settings-toggle-goldens-tdd.md) | Completed |
| 4 | [Docs sync + full verification](./phase-04-docs-sync-full-verification.md) | Pending |

## Dependencies

- Strictly sequential: 1 → 2 → 3 → 4. Phase 3 wires the field (P1) and the
  setting (P2) together; Phase 4 documents the merged result.
- No cross-plan dependencies (other unfinished plans do not touch
  `internal/typing`, `internal/metrics`, settings, or the Settings screen).

## Acceptance Criteria (whole feature)

- [ ] With `strict_mode` off: behavior identical to today (regression-free).
- [ ] With `strict_mode` on: wrong key does not advance the cursor; correct key
      does; `States()` never shows `Incorrect` for in-target positions.
- [ ] Blocked errors are logged and reduce keystroke accuracy (not stuck at 100%).
- [ ] `strict_mode` persists across restart; missing key in old JSON → `false`.
- [ ] `typeburn config get strict_mode` / `set strict_mode on|off` work and are
      validated; `config list` includes the key.
- [ ] Settings screen shows a "Strict mode" row that toggles and live-applies.
- [ ] Strict runs are saved to History but never earn ★ (excluded from best,
      like Code mode); `storage.Record` carries a `strict` flag (back-compat).
- [ ] Result/History show keystroke accuracy for strict runs, final-state otherwise.
- [ ] `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` all clean;
      teatest goldens regenerated and reviewed.

## Constraints

- Pure-logic packages (`typing`, `metrics`) stay UI-free.
- Every Go file < 200 LOC — split if a touched file would exceed it.
- Core logic files `snake_case`.
- Conventional commits, **no AI references**.

## Handoff git-workflow rules (MANDATORY — for the implementing agent)

This plan is a handoff for a **different agent**. Enforce on every phase:

1. **One branch per phase**, off up-to-date `main`:
   `feat/strict-mode-p1-engine`, `-p2-config`, `-p3-wire`, `-p4-docs`
   (or a single `feat/strict-mode` branch with one commit per phase — but
   separate PRs per phase are preferred for reviewability).
2. **≥ 1 commit per phase** (conventional commits, no AI refs).
3. **On phase completion, run `/vchun-git prc`** to execute the full PR pipeline:
   branch → commit → push → open PR → CI green → **squash-merge** to protected
   `main` → branch auto-delete. **Never push to `main` directly** (local
   PreToolUse hook + GitHub branch protection both block it).
4. Mark the phase done with `ck plan check <phase-id>` only **after** its PR is
   squash-merged and CI is green on `main`.
5. Re-base the next phase branch on the freshly merged `main` before starting.

## Environment (handoff context)

- CWD: `/Users/vchun/Codes/My-projects/Typeburn`, branch `main`, macOS/zsh.
- Go 1.25, Bubble Tea v2 / Lip Gloss v2. `gh` authenticated; `ck` v4.5.0.
- Gates = `make test-race` (CI gate), `make lint` (gofmt + vet + no-TUI guard).
- TDD: write failing tests first each phase, then implement to green.

## Resolved decisions (via /ck:plan validate, 2026-06-26)

1. **★/best: strict runs are EXCLUDED from the per-mode ★ best**, exactly like
   Code mode. `internal/storage/new_best.go:61-63` already early-returns `false`
   for `r.Mode == "code"`; extend that guard with `|| r.Strict`. Strict runs are
   still saved to History; they just never earn ★. No separate bucket (KISS).
2. **Accuracy display:** strict runs show **keystroke accuracy**; non-strict runs
   show the existing **final-state accuracy**. Never show both (no Result/History
   clutter).
3. **Schema:** add `Strict bool json:"strict"` to `storage.Record`. Backward-
   compatible (legacy records lack the key → `false`), mirroring how `NetWPM` was
   added. Required for decisions #1 and #2 to work on reloaded records.

These are now folded into the phases below — **no open questions remain.**
