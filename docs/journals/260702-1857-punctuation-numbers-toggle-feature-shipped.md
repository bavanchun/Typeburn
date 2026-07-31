---
title: Punctuation & Numbers Toggle for Words/Time Mode (PR #57)
date: 2026-07-02
type: journal
---

# Punctuation & Numbers Toggle for Words/Time Mode

## Context

Shipped PR #57 (branch `feat/punctuation-numbers-toggle`, commit 00286ed2): two new persisted settings (`Punctuation` and `Numbers`, both bool, default `false`) for Words/Time mode word generation. Mirrors StrictMode wiring exactly — settings-only control surface (no new CLI flags), config-driven threading down through the call chain to the generator transform. Plan: `/plans/260702-1824-punctuation-numbers-toggle/`.

## What Happened

**Phase 1–3 execution:** Config fields added to `Settings`, generator transforms implemented (`applyPunctuation`/`applyNumbers` in `internal/words/generator.go` with deterministic seeding), threading through `runner.NewSession` → constructor chain → `TypingModel` fields, Settings UI rows added, CLI `config get/set punctuation|numbers` wired.

**Validation gate earned its keep:** Before implementation, `/ck:plan validate` (Standard tier: Fact Checker + Contract Verifier) caught 4 factual errors in the Phase 2 "touchpoint list":
- Claimed `internal/app/model.go` calls `words.ForMode` directly → FALSE (call chain is `ui.NewTyping() → newTypingWithSeed → runner.NewSession → words.ForMode`)
- Claimed `internal/cli/cmd_run_validate.go` is a call site → FALSE (real site is `cmd_run_notui.go:54`)
- Omitted `internal/ui/screen_typing_actions.go:30` (ctrl+r restart path also calls `newTypingWithSeed` and must carry punctuation/numbers forward)
- Omitted `NewTyping`/`newTypingWithSeed` signature updates as required-edit files

**These were corrected before implementation,** not after, preventing rework. Interview resolved 2 decisions: (a) yes, add `punctuation`/`numbers` fields to `TypingModel` to mirror `strict` behavior on restart, (b) two separate positional bools, not a struct, matching existing call-site style.

**Post-implementation code review** (subagent) found 4 issues:

| Severity | Finding | Resolution |
|---|---|---|
| HIGH | Quote-wrap feature spec'd in acceptance criteria but never coded. No test caught it. | Implemented: `applyPunctuation` now wraps ~3% of tokens in quotes. User chose to implement. |
| MEDIUM | `applyNumbers` off-by-one: `rng.IntN(max)+1` could produce a 5-digit number when `digits==4` (e.g., `10000`). | Fixed: `rng.IntN(max-1)+1`. |
| MEDIUM | Capitalization triggers after both `.` and `;`, not just `.` as literally spec'd. Undocumented deviation. | User chose to keep as-is (reasonable stylistic choice). Locked in with `TestApplyOptions_PunctuationCapitalizesAfterSemicolon`. |
| LOW | Stale "5 fixed settings rows" doc comment in `settings_rows.go` (should be 7). | Fixed. |

Full re-verification post-fixes: `go test ./... -race -count=1`, `gofmt -l .`, `go vet ./...` all green. Squash-merged to `main`.

## The Friction Points

The quote-wrap gap was the stinger. It was written into the acceptance criteria in the plan (`rare word wrapped in quotes`) and should have been caught during Phase 1 code review — but wasn't. This is a reminder that acceptance criteria in markdown and actual test assertions don't automatically sync. The test suite passed without it because no test explicitly checked for the feature; the logic was simply missing from the generator.

The off-by-one in `applyNumbers` was a classic modulo mistake: `rng.IntN(max)` returns [0, max), so adding 1 shifts it to [1, max+1) — fine for max=10 (digits 1–10), catastrophic for max=10000 (digits 1–10001, i.e., 5 digits). The fix is trivial, but this is why deterministic seeding + explicit test cases for boundary values matter.

The semicolon capitalization deviation sneaked in during implementation. The spec said "period only" but the code capitalized after both `.` and `;`. Rather than debate style, the user owned the call: keep it (semicolons are sentence-like terminators anyway), and test it so the behavior is locked in.

## Why the Validation Gate Worked

The plan's Phase 2 file list was detailed enough to fail productively — it made claims about call chains and file paths that could be verified by actually tracing the code. The validation agent didn't just check "are the files reasonable" but "do the actual call paths match the claim?" That's the difference between a weak gate and one that catches rework-sized mistakes before they compound.

This is a lesson for future plans: specificity in the draft (exact file paths, exact function call chains, exact line numbers) creates testable claims. Vague plans ("update related screens") don't fail; specific plans can.

## Next

None. Feature shipped, all tests green, docs updated (README.md, project-roadmap.md).

## Unresolved Questions

None.
