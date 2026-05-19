---
phase: 3
title: "ModeCode + Completion Seam"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 3: ModeCode + Completion Seam (TDD)

## Overview
Add `ModeCode` across the mode seam (config enum, LengthsFor, Normalize,
storage Record doc) and make the typing engine complete a code test by exact
full-text match — reusing the Quote rule. One commit.

## Requirements
- Functional: `config.ModeCode Mode = "code"`. `LengthsFor(ModeCode)` →
  `nil` (no length options). `Settings.Normalize` accepts `"code"`.
  `typing` completion: `case ModeCode` behaves identically to `ModeQuote`
  (`runesEqual(typed,target)`), incl. literal `\n`/`\t` runes; no word-count,
  no AFKTrim. Metrics path unchanged (chars derive from keystroke log).
- Non-functional: mode-seam duplication guarded by a sync test (mirror the
  theme `theme_available_sync_test.go` discipline); files <200 LOC.

## Architecture
`config.Mode` is a string enum used in switches at: `LengthsFor`
(`settings.go:17`), `Normalize` (`settings.go:62`), `modeOrder`
(`screen_home.go:24`, Phase 5), `completion.go` (isComplete switch),
`storage.Record.Mode` doc. Code reuses Quote's completion arm:
`case ModeQuote, ModeCode: return runesEqual(e.typed, e.target)`.
`countCompletedWords`/AFKTrim are never reached for code (not Words/Time).

## Related Code Files
- Modify: `internal/config/settings.go` (const + LengthsFor + Normalize),
  `internal/typing/completion.go` (ModeCode arm),
  `internal/storage/history_record.go` (Mode doc comment: add "code")
- Create: `internal/config/mode_seam_sync_test.go` (asserts every mode in
  `modeOrder`-equivalent set is handled by LengthsFor+Normalize; and that
  Normalize accepts exactly the known modes incl. code, unknown→ModeTime)
- Modify (tests): `internal/typing/completion_test.go` (ModeCode cases),
  `internal/config/settings_test.go` (LengthsFor/Normalize code)

## Implementation Steps (tests-first)
1. **RED:** add completion_test cases — a ModeCode engine with target
   containing `\n` and `\t`: not complete until `typed==target` exactly;
   wrong char ≠ complete; trailing extra runes ≠ complete (same semantics as
   the existing Quote tests — copy/adapt). settings_test: `LengthsFor("code")
   == nil`; `Normalize` keeps `"code"`, maps unknown→`time`. New
   `mode_seam_sync_test.go`: list the known modes
   {time,words,quote,code}; assert each is non-panicking in `LengthsFor`
   and preserved by `Normalize`; assert a bogus mode → `ModeTime`. Run → red.
2. **GREEN:** add `ModeCode` const; `LengthsFor` `case ModeCode: return nil`;
   `Normalize` add `ModeCode` to the valid switch; `completion.go` add
   `ModeCode` to the Quote arm; update `history_record.go` Mode comment.
3. Refactor; keep files <200 LOC.
4. `make fmt && make lint && make test-race`. Commit:
   `feat(mode): add ModeCode with exact-match completion`.

## Success Criteria
- [ ] ModeCode completes only on exact full-text match incl `\n`/`\t`;
  Quote/Words/Time completion tests unchanged & green.
- [ ] `LengthsFor(ModeCode)==nil`; Normalize accepts "code"; sync test red
  if a mode is added to the enum but missed in LengthsFor/Normalize.
- [ ] gofmt/vet/`-race` green; <200 LOC.
- [ ] One commit.

## Risk Assessment
- Hidden Words/Time assumptions on code path: code is neither, so
  countCompletedWords/AFKTrim unreachable — assert via a test that a ModeCode
  run with no word boundaries still completes on exact match.
- Sync-test placement: `package config` internal or `config_test` external —
  must compile without importing `ui`; keep it in `internal/config`.
