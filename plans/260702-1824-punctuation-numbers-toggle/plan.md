---
title: Punctuation & Numbers Toggle for Words/Time Mode
description: >-
  Monkeytype-parity punctuation/numbers toggles for Words and Time mode word
  generation.
status: completed
priority: P2
branch: main
tags:
  - feature
  - words
  - config
  - ui
blockedBy: []
blocks: []
created: '2026-07-02T11:32:31.061Z'
createdBy: 'ck:plan'
source: skill
---

# Punctuation & Numbers Toggle for Words/Time Mode

## Overview

Two new persisted settings toggles — `Punctuation` and `Numbers`, both bool,
default `false` — applied to Words/Time mode word generation only. Wiring shape
mirrors `StrictMode` (PR #52-54): `config.Settings` field → generator transform
→ call-site threading → Settings-screen toggle row → CLI `config` key. Quote/Code
modes and the typing/metrics engine are untouched.

Brainstorm report: `plans/reports/feature-recommendation-260702-1824-punctuation-numbers-toggle-report.md`

## Phases

| Phase | Name | Status |
|-------|------|--------|
| 1 | [Config & Generator (TDD)](./phase-01-config-generator-tdd.md) | Completed |
| 2 | [Wiring Call Sites (TDD)](./phase-02-wiring-call-sites-tdd.md) | Completed |
| 3 | [Settings UI & CLI (TDD)](./phase-03-settings-ui-cli-tdd.md) | Completed |
| 4 | [Docs Sync](./phase-04-docs-sync.md) | Completed |

## Acceptance Criteria

- Punctuation ON: ~15-20% of words get trailing `,`/`.`/`;`; word after a `.`
  is capitalized; rare word wrapped in quotes. Deterministic given a fixed seed.
- Numbers ON: ~10-15% of word-slots replaced with a random 1-4 digit token.
- Both default `false`; legacy `settings.json` without these fields loads
  safely (Go zero-value `false`, same as `StrictMode` precedent).
- Quote mode, Code mode, typing engine, metrics engine: byte-for-byte unchanged.
- `go test ./... -race -count=1`, `go vet ./...`, `gofmt -l .` all green.

## Dependencies

None — all prior plans in `./plans/` are `status: completed`, no overlap detected.

## Validation Log

### Verification Results (Standard tier: Fact Checker + Contract Verifier)
- Claims checked: 12 (call chain, file paths, line numbers, signatures)
- Verified: 8 | Failed: 4 | Unverified: 0
- Tier: Standard (4 phases)
- Failures:
  - Phase 2 claimed `internal/app/model.go` calls `words.ForMode` directly —
    FALSE. It calls `ui.NewTyping(...)`, which calls `newTypingWithSeed` →
    `runner.NewSession` → `words.ForMode`. (`internal/app/model.go:86,89`,
    `internal/ui/screen_typing.go:56,71`, `internal/runner/session.go:25`)
  - Phase 2 claimed `internal/cli/cmd_run_validate.go` is a `ForMode` call site
    — FALSE. That file only computes `runLengthForMode`; the real CLI call
    site is `internal/cli/cmd_run_notui.go:54` (`runSession` → `runner.NewSession`).
  - Phase 2 omitted `internal/ui/screen_typing_actions.go:30` — the ctrl+r
    restart path also calls `newTypingWithSeed` and must carry
    punctuation/numbers forward (mirrors existing `m.strict` field), or the
    setting would silently drop on restart.
  - Phase 2 omitted `internal/ui/screen_typing.go`'s `NewTyping`/`newTypingWithSeed`
    signatures entirely as a required-edit file.
  - 5 test files call these constructors directly and need signature updates:
    `internal/ui/screen_typing_test.go`, `internal/ui/screen_typing_actions_test.go`,
    `internal/ui/phase09_polish_test.go`, `internal/app/anim_driver_test.go`,
    `internal/runner/session_test.go`.

### Interview (4 questions, mode=prompt)
1. **Fix Phase 2 file list?** → Yes, rewrite with corrected call chain
   (words/for_mode.go, runner/session.go, ui/screen_typing.go,
   ui/screen_typing_actions.go, app/model.go, cli/cmd_run_notui.go + 5 test
   files); dropped `cmd_run_validate.go` reference.
2. **ctrl+r restart scope?** → Yes, include: `TypingModel` gains
   `punctuation`/`numbers` fields alongside existing `strict`, so restart
   preserves them (matches StrictMode precedent).
3. **Param shape for the 2 new bools?** → Two separate positional bool params
   (`punctuation, numbers bool`), NOT a bundled options struct — matches
   existing style (no structs used in this call chain today; introducing one
   would be a new pattern not otherwise justified).
4. **CLI flag scope?** → Settings-only, zero new `cmd_run.go` flags — matches
   verified StrictMode precedent (no `--strict` flag exists either). Only
   `config set punctuation|numbers` is a control surface.

### Whole-Plan Consistency Sweep
Re-read `plan.md` + all 4 phase files after propagation. No stale references
to `cmd_run_validate.go` remain outside this log (was only in Phase 2, now
corrected there). Phase 1's `ApplyOptions(text, punctuation, numbers bool)`
signature is consistent with Phase 2's corrected threading (two bools, not a
struct). No unresolved contradictions.

## Code Review Log

Post-implementation `code-reviewer` subagent review found 4 issues:
- **HIGH:** Quote-wrap requirement (`ApplyOptions` acceptance criterion above)
  was implemented in Phase 1's plan text but never coded — no test caught it.
  User decision: implement it. Fixed in `internal/words/generator.go`
  `applyPunctuation` (~3% chance to wrap a token in `"…"`), with new tests
  `TestApplyOptions_PunctuationWrapsSomeTokensInQuotes`.
- **MEDIUM:** `applyNumbers` off-by-one — `rng.IntN(max)+1` could produce a
  5-digit number when `digits==4` (e.g. `10000`), violating the "1-4 digit"
  contract. Fixed: `rng.IntN(max-1)+1`.
- **MEDIUM:** Capitalization triggers after both `.` and `;`, not just `.` as
  literally spec'd. User decision: keep as-is (reasonable stylistic choice),
  locked in with `TestApplyOptions_PunctuationCapitalizesAfterSemicolon`.
- **LOW:** Stale "5 fixed settings rows" doc comment in `settings_rows.go`
  (sibling files already said 7). Fixed.

All 4 findings resolved. Full re-verification: `go test ./... -race -count=1`,
`gofmt -l .`, `go vet ./...` all green after fixes.
