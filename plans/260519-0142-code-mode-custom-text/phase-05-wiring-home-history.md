---
phase: 5
title: "Wiring + Home + History"
status: pending
priority: P1
effort: "4h"
dependencies: [2, 3, 4]
---

# Phase 5: Wiring + Home + History (TDD)

## Overview
Wire the pieces: `main.go --text` flag → `codetext.Load` → injected into
`app.Model` → `StartTestMsg` → `NewTyping` (Code path uses the loaded text +
the code renderer, bypassing `words.ForMode`). Home shows Code in the cycle
(disabled+hint when no text, no length selector). `IsNewBest` excludes
`Mode=="code"`. One commit.

## Requirements
- Functional:
  - `typeburn --text <file>` / `--text -` parsed by `decide()` (extend the
    existing tested FlagSet; unknown-flag/`-h` behaviour preserved). Load
    failure (ErrEmpty/ErrTooLarge/ErrBinary) → app starts normally with Code
    disabled and the error reason as the hint (no crash, no os.Exit beyond
    existing `--version` semantics).
  - Home: `modeOrder` includes `ModeCode`; when no code text → row shows
    disabled style + hint `pass --text <file> · in-app paste coming soon`,
    Enter is a no-op (stays on Home); when code text present → Enter emits
    `StartTestMsg` for Code. **No length selector** rendered on the Code row
    (Code row has no value-cycler).
  - Typing screen uses the code renderer + viewport for `ModeCode`
    (`screen_typing.go` selects renderer by mode); Tab/Enter reach the engine
    mid-test (verify `model_key_handler.go` already delegates all keys while
    typing — add a regression test, do not change behaviour).
  - Result/History: a Code run persists a `Record{Mode:"code",
    Length:runeCount,…}`; `IsNewBest` returns false for `Mode=="code"`;
    History screen lists it.
- Non-functional: pure-core untouched (loader is in `codetext`; `words`
  unchanged); files <200 LOC; goldens for other modes unchanged.

## Architecture
`StartTestMsg` gains a code payload (e.g. `CodeText string` + the existing
`Mode`). `app.Model` holds the loaded code text + an availability flag from
`New`/startup wiring (text passed in from `main.go`). `NewTyping` for
`ModeCode`: `target = codeText` (skip `words.ForMode`), renderer = code
renderer. `IsNewBest` (`new_best.go`): early `if r.Mode == "code" { return
false }`. Home forks rendering on `codeAvailable`.

## Related Code Files
- Modify: `main.go` (`decide()` `--text`, load, pass into app),
  `internal/app/model.go` (carry code text/availability),
  `internal/app/model_view.go`/wiring + `internal/ui/messages.go`
  (StartTestMsg payload), `internal/ui/screen_home.go` +
  `screen_home_view.go` + `settings_rows.go`-style row (Code disabled/hint,
  no length), `internal/ui/screen_typing.go` (renderer select by mode),
  `internal/storage/new_best.go` (exclude code)
- Create: tests alongside each (`*_test.go`)
- Delete: none

## Implementation Steps (tests-first)
1. **RED:**
   - `main_test.go`/`decide` test: `--text path` populates the parsed
     value; `--text -` recognized; absent → zero value; unknown flags / `-h`
     / `--version` behaviour unchanged (extend existing decide tests).
   - `new_best_test.go`: a `Record{Mode:"code"}` that would otherwise be a
     PB → `IsNewBest==false`; non-code unaffected (existing tests stay green).
   - Home test: no code → Code row disabled, hint text present, Enter →
     still Home (no StartTestMsg); with code → Enter emits StartTestMsg{Mode:
     code}; Code row has no length cycler; Tab cycle includes Code.
   - app/key regression test: during a ModeCode test, a Tab and an Enter
     keypress are delivered to the engine (not consumed as nav).
   - app integration: feeding code text + completing exact match →
     ResultMsg → persisted Record Mode=="code", visible via NavHistory.
   Run → red.
2. **GREEN:** implement wiring minimally to pass; reuse Phases 2–4 APIs.
3. Refactor; keep files <200 LOC (split screen_home if it grows).
4. `make fmt && make lint && make test-race`. Commit:
   `feat: wire --text code mode through Home, typing, history`.

## Success Criteria
- [ ] `--text file`/`-` works end to end; bad input → Code disabled + reason,
  no crash.
- [ ] Home: Code in cycle; disabled+hint w/o text; no length selector;
  Enter no-op w/o text, starts test w/ text.
- [ ] Tab/Enter reach engine mid-code-test (regression test green).
- [ ] Code Record persisted, never ★, listed in History.
- [ ] Other modes’ goldens/tests unchanged; gofmt/vet/`-race` green;
  files <200 LOC.
- [ ] One commit.

## Risk Assessment
- `decide()` FlagSet must keep `ContinueOnError`/discarded-output semantics
  (no `os.Exit(2)` on unknown) — assert via the preserved decide tests.
- StartTestMsg payload growth could ripple to ResultModel re-emit
  (`QuoteLen` precedent) — mirror that pattern; test ctrl+r restart on a
  code test re-uses the same text.
- Home view LOC creep → split view/row helpers.
- Ensure `words.ForMode` is NOT called for code (keeps core pure) — assert
  no `words` import added to the code path.
