---
phase: 4
title: "Wiring + Routing + Home"
status: pending
priority: P1
effort: "3h"
dependencies: [3]
---

# Phase 4: Wiring + Routing + Home (TDD)

## Overview
Add `ScreenCodePaste` to the Screen enum + routing; route `tea.PasteMsg` to
the paste sub-model when on that screen (keep the Typing branch intact);
apply `CodePastedMsg` (set `m.codeText`, clear `m.codeHint`, push it into
the existing Home via `WithCodeText` — NOT a NewHome rebuild, F3 —
screen→Home with Code still selected); esc→Home via the existing global
Back handler (no new code); Home Code-row Enter when empty →
`NavCodePasteMsg`. One commit.

## Requirements
- Functional:
  - `ScreenCodePaste` enum value; root `Update`/`View`/key-handler route it
    like the other screens (delegate keys + View to the sub-model).
  - `tea.PasteMsg` case in `model.go:131` (struct `tea.PasteMsg{Content}`):
    when `screen==ScreenTyping` → existing behaviour UNCHANGED; add
    `else if screen==ScreenCodePaste` → forward to the paste sub-model; else
    `return m, nil` (as today). Keep the Typing branch byte-intact.
  - `NavCodePasteMsg` → screen=ScreenCodePaste (fresh sub-model).
  - `CodePastedMsg{Text}` → `m.codeText=Text`, `m.codeHint=""`, **propagate
    the new codeText into the EXISTING Home sub-model without resetting its
    state**, screen=ScreenHome (see F3 in Architecture — do NOT blind-rebuild
    via `NewHome`, which resets `modeIdx` to `DefaultMode` and loses the Code
    selection).
  - `esc` from the paste screen needs **no new routing**: the global `Back`
    handler (`model_key_handler.go:64`, `else → m.screen = ScreenHome`)
    already covers any non-Home screen and leaves `codeText` untouched.
    Only verify `ScreenCodePaste` is not special-cased above that `else`.
  - `screen_home.go`: Code-row Enter when `codeText==""` emits
    `NavCodePasteMsg` (was nil/no-op). `--text`/enabled path unchanged
    (Enter still starts the test). Code-row hint text updated to invite
    paste (e.g. "press enter to paste a snippet").
- Non-functional: time/words/quote/Code(`--text`) behaviour unchanged;
  goldens unchanged; files <200 LOC; pure-core untouched.

## Architecture
`Screen` iota gains `ScreenCodePaste`. Root holds a `codePaste` sub-model
field (constructed on `NavCodePasteMsg`). Routing parallels Settings/History.

**F3 — preserve the Code selection (do NOT mirror onSettingsChange's blind
rebuild).** `NewHome` derives `modeIdx` from `s.DefaultMode`
(`screen_home.go:54-57`); rebuilding Home after paste would snap the
selector back to Time, contradicting "valid paste → Home, Code enabled,
press Enter to start". Instead add a small method on `HomeModel`, e.g.
`func (m HomeModel) WithCodeText(text, hint string) HomeModel` that returns
a copy with only `codeText`/`codeHint` replaced (modeIdx, lenIdx, sizes
preserved). App applies it to the existing `m.home` on `CodePastedMsg`. The
Code row was the active row when the user pressed Enter → paste, so on
return Code stays selected and Enter starts immediately. (`onSettingsChange`
keeps its full rebuild — different flow, leave it.)

**F4 — bracketed paste enablement.** `main.go:74` `tea.NewProgram(...)`
sets no paste option; the Typing `PasteMsg` tests only synthesize the msg,
they do not prove the terminal delivers it. Before relying on it: confirm
Bubble Tea v2 enables bracketed paste by default for this version (check the
charm-v2 cheatsheet / `tea` defaults); if it requires an explicit
`tea.NewProgram` option, add it in `main.go` (and note Typing paste also
depended on it). Phase 5 manual smoke is the real-terminal proof.

Only the empty-Code path opens paste (out-of-scope: replacing a `--text`
snippet). A failed `--text` load leaves `codeText==""` (with `codeHint`
set) → that IS the empty path, so Enter there also opens paste (recovery) —
intended and consistent with the entry rule.

## Related Code Files
- Modify: `internal/app/model.go` (enum, `codePaste` field, NavCodePasteMsg
  + CodePastedMsg routing, PasteMsg `else if ScreenCodePaste` branch),
  `internal/app/model_key_handler.go` + `internal/app/model_view.go` (route
  new screen for keys+View; **verify** the existing Back `else` already
  sends ScreenCodePaste→Home — likely no esc code needed),
  `internal/ui/screen_home.go` (Code-row Enter→NavCodePasteMsg; add
  `HomeModel.WithCodeText(text,hint) HomeModel` preserving modeIdx/lenIdx),
  `internal/ui/screen_home_view.go` (Code-row hint wording),
  `internal/ui/messages.go` (`NavCodePasteMsg`; `CodePastedMsg` from P3 — no
  cancel msg)
- Create: tests alongside (`*_test.go`)
- Delete: none

## Implementation Steps (tests-first)
1. **RED:**
   - app routing test: `NavCodePasteMsg` → `m.screen==ScreenCodePaste`;
     `CodePastedMsg{"x"}` → `m.codeText=="x"`, `m.codeHint==""`,
     `m.screen==ScreenHome`, **and the Home selector is still on the Code
     row** (assert `m.home.currentMode()==config.ModeCode` via a tiny
     accessor, or that Home `startCmd()` for the post-paste state is
     non-nil i.e. Code-enabled); modeIdx/lenIdx not reset to defaults.
   - F3 unit: `HomeModel.WithCodeText("x","")` returns a Home with
     codeText=="x" and the SAME modeIdx/lenIdx as the receiver.
   - esc routing: from `ScreenCodePaste`, a Back/esc key → `m.screen==
     ScreenHome` and `m.codeText` unchanged (proves the existing global
     Back `else` covers the new screen — no new esc code).
   - PasteMsg routing (`tea.PasteMsg{Content:...}`): on `ScreenTyping` the
     existing Typing paste behaviour is unchanged (the
     `phase09_polish_test.go` PasteMsg tests stay green — do not modify
     them); on `ScreenCodePaste` a PasteMsg reaches the paste sub-model; on
     Home it is ignored (`nil` cmd).
   - Home test: with `codeText==""`, Code-row Enter emits `NavCodePasteMsg`
     (not nil); with codeText set, Enter still emits `StartTestMsg` (v1.2.0
     unchanged); Code row has no length cycler still.
   - view/key routing: `ScreenCodePaste` View renders the sub-model; keys
     route to it; other screens unaffected (table-test all 6).
   Run → red.
2. **GREEN:** implement minimally to pass; reuse Phase 3 sub-model/messages.
3. Refactor; keep files <200 LOC.
4. `make fmt && make lint && make test-race`. Commit:
   `feat(app): route ScreenCodePaste; Home Code-row opens paste`.

## Success Criteria
- [ ] End-to-end (no `--text`): Home→Code→Enter→paste screen→valid paste→
  back on Home **with the Code row still selected**→Enter→Code test on the
  pasted snippet (persists Mode=code, excluded from ★).
- [ ] `--text` precedence + all v1.2.0 Code behaviour unchanged; Typing
  PasteMsg branch byte-unchanged (phase09 paste tests green, untouched);
  time/words/quote goldens unchanged.
- [ ] esc on paste → Home, codeText unchanged (via the existing global Back
  handler — no new esc code added).
- [ ] `HomeModel.WithCodeText` preserves modeIdx/lenIdx (F3).
- [ ] Bracketed paste confirmed enabled (default or explicit `tea` option in
  main.go) — F4.
- [ ] gofmt/vet/`-race` green; files <200 LOC; one commit.

## Risk Assessment
- 6th-screen routing regressions — table-test all screens’ key/View routing;
  keep the Typing PasteMsg branch byte-intact (phase09 paste tests are the
  lock — must stay green, do not edit them).
- **F3:** do NOT blind-rebuild Home via `NewHome` on `CodePastedMsg` (resets
  modeIdx→DefaultMode, loses Code selection). Use `WithCodeText` on the
  existing Home; unit-test selection preservation.
- **F4:** bracketed paste must actually fire in a real terminal — verify
  default/option in Phase 4, prove in Phase 5 manual smoke. Unit tests only
  prove the handler, not delivery.
- **F2 over-design removed:** no cancel message / no sub-model esc handling
  — the global Back handler already does esc→Home. Confirm with the esc
  routing test rather than adding code.
- StartTestMsg/ResultMsg unaffected (paste only sets codeText pre-start; the
  v1.2.0 restart-same fix still applies once a test runs).
