---
phase: 1
title: Config & Generator (TDD)
status: completed
effort: small
priority: P1
dependencies: []
---

# Phase 1: Config & Generator (TDD)

## Overview

Add `Punctuation`/`Numbers` bool fields to `config.Settings`, and a
deterministic post-generation transform in `internal/words` that applies
punctuation/number substitution to a word string. Both packages stay UI-free
per the layering rule. This is the core logic phase — no wiring to app/UI yet.

## Requirements

- Functional:
  - `config.Settings.Punctuation`, `config.Settings.Numbers` (bool, JSON tags
    `punctuation`/`numbers`), default `false` in `Defaults()`.
  - `words.Generator` gets a new method, e.g. `ApplyOptions(text string, punctuation, numbers bool) string`,
    using the generator's existing seeded `rng` (no new randomness source).
  - Punctuation transform: iterate space-separated tokens; ~15-20% chance per
    token to append `,`/`.`/`;` (weighted, periods rarer than commas); if a
    token follows a token ending in `.`, capitalize its first rune; ~2-3%
    chance to wrap a token in `"…"`. Trailing token always gets a `.` if
    punctuation is on and it doesn't already end in `,`/`.`/`;` (test finishes
    on a sentence-like beat).
  - Numbers transform: ~10-15% chance per token to replace it wholesale with a
    random 1-4 digit numeric string (e.g. `strconv.Itoa(rng.IntN(9000)+1)` style,
    biased toward shorter numbers).
  - Order of application when both on: numbers substitution first, then
    punctuation pass over the resulting token list (so a numeric token can also
    get a trailing comma/period).
- Non-functional:
  - Fully deterministic for a fixed seed (existing `TestWords_Deterministic`-style
    coverage must extend to the transform).
  - No new external deps — stdlib `math/rand/v2`, `strconv`, `strings` only.
  - File size: if `generator.go` exceeds ~200 LOC after this addition, split the
    transform into `internal/words/options.go` (kebab/snake per Go convention:
    `options.go`, since Go uses lowercase single-word or `snake_case` file
    names — check existing sibling files for the pattern before naming).

## Architecture

`words.Generator.ApplyOptions` is a pure function over its `rng` and the input
string — no coupling to `mode.Mode`. `ForMode` (phase 2 concern, not touched
here) will call it conditionally. Keeping the transform mode-agnostic means
Quote/Code modes never call it and stay untouched by construction, not by a
guard clause that could be forgotten.

## Related Code Files

- Modify: `internal/config/settings.go` (fields + `Defaults()`; `Normalize()`
  needs no change — bools have no invalid state, confirmed by `StrictMode`
  precedent)
- Modify: `internal/words/generator.go` (or new `internal/words/options.go` if
  LOC cap forces a split — check current LOC first: `wc -l internal/words/generator.go`)
- Modify/Create: `internal/words/generator_test.go` (or `options_test.go` if
  split) — TDD: write these tests FIRST, watch them fail, then implement.

## Implementation Steps

1. **RED**: Write failing tests in `generator_test.go` (or new file):
   - `TestApplyOptions_NoOp` — punctuation=false, numbers=false → input unchanged.
   - `TestApplyOptions_PunctuationDeterministic` — same seed, same output.
   - `TestApplyOptions_PunctuationAddsMarks` — with punctuation=true on a fixed
     seed + known word count, assert at least one token ends in `,`/`.`/`;`.
   - `TestApplyOptions_PunctuationCapitalizesAfterPeriod` — assert a token
     following a `.`-terminated token starts with an uppercase rune.
   - `TestApplyOptions_NumbersProduceDigits` — with numbers=true on a fixed
     seed, assert at least one token is all-digit via a small regex/loop check.
   - `TestApplyOptions_TokenCountPreserved` — token count in == token count out
     (punctuation/numbers modify tokens, never add/remove word slots — this is
     the invariant `ForMode`'s Words-mode exact-count contract in phase 2 relies
     on).
   - `TestApplyOptions_DeterministicAcrossOptions` — same seed + both flags on
     → identical output across two calls (regression guard for `TestWords_Deterministic`
     equivalent).
2. Run tests, confirm RED (compile failure or assertion failure is fine — must
   fail for the right reason, not a typo).
3. **GREEN**: Implement `config.Settings` fields + `Defaults()` update.
4. **GREEN**: Implement `ApplyOptions` on `Generator` satisfying the tests above.
5. Run `go test ./internal/words/... ./internal/config/... -race -count=1` —
   confirm GREEN.
6. **REFACTOR**: if `generator.go` > ~180 LOC, extract the transform + its
   constants (percentages, punctuation mark set) into a sibling file; re-run
   tests to confirm the split didn't break anything.
7. Run `gofmt -l internal/words internal/config` and `go vet ./internal/words/... ./internal/config/...` — must be clean.

## Success Criteria

- [x] `config.Settings` has `Punctuation`/`Numbers` bool fields, default false
- [x] `words.Generator.ApplyOptions` exists, is deterministic per seed
- [x] Token count is preserved (in == out) for any combination of flags
- [x] Punctuation-off / numbers-off is a true no-op (byte-identical output)
- [x] All new tests pass under `-race -count=1`
- [x] `gofmt -l` clean, `go vet` clean
- [x] No file exceeds ~200 LOC (split if needed)

## Risk Assessment

- **Risk:** Transform accidentally changes token count → breaks
  `ForMode`'s Words-mode "exactly N words" contract (existing `TestWords_ExactCount`).
  **Mitigation:** `TestApplyOptions_TokenCountPreserved` test written first (RED),
  transform never appends/removes tokens, only mutates them in place.
- **Risk:** Non-deterministic output breaks existing seed-based test patterns.
  **Mitigation:** Transform only consumes `g.rng`, never a new random source;
  deterministic tests written before implementation.
