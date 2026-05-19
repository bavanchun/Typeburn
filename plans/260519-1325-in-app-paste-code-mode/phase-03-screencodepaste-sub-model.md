---
phase: 3
title: "ScreenCodePaste Sub-model"
status: completed
priority: P1
effort: "3h"
dependencies: [2]
---

# Phase 3: ScreenCodePaste Sub-model (TDD)

## Overview
New `internal/ui` paste sub-model: shows an instruction/waiting view,
captures one `tea.PasteMsg` (`msg.Content`), runs `codetext.Normalize`, and
on success emits `CodePastedMsg{Text}`; on error stays with the reason shown
(retry). `esc` returns to Home via the existing global Back handler — NOT
handled by this sub-model (F2). Approach A — the sub-model owns
normalization + error state. One commit.

## Requirements
- Functional: `CodePasteModel` with `Update(tea.Msg)(CodePasteModel,tea.Cmd)`
  + `View() string` (same shape as other sub-models). `tea.PasteMsg` is a
  **struct** — `tea.PasteMsg{Content string}` (verified: `model.go:131`,
  `screen_typing.go` uses `msg.Content`). On `tea.PasteMsg`:
  `codetext.Normalize(msg.Content)` → ok ⇒ emit `CodePastedMsg{Text}`;
  err ⇒ store a user-facing reason (map `ErrEmpty`/`ErrBinary`/`ErrTooLarge`
  via `errors.Is`), stay in error state, allow another paste.
  **NO esc / cancel handling here** — the global `Back` handler
  (`model_key_handler.go:64`) already routes esc on any non-Home screen →
  `ScreenHome` *before* the sub-model is reached, leaving `codeText`
  untouched. The sub-model handles ONLY `tea.PasteMsg` (and ignores
  everything else, returning itself + nil). No cancel message exists.
  No char echo; no typed input.
- Non-functional: NO_COLOR-safe (theme `Role` styles only, layout identical);
  pure sub-model (no I/O — codetext.Normalize is pure); files <200 LOC
  (split `_view.go` if needed).

## Architecture
States: `waiting` (instruction) and `errored(reason string)`. Mirrors the
existing degraded/quit-prompt rendering idiom. ONE message added in
`internal/ui/messages.go`: `CodePastedMsg{Text string}` (no cancel message —
esc is handled globally, see Requirements). The sub-model does NOT mutate
app state; it only emits `CodePastedMsg` on a valid paste — the app applies
it (Phase 4).

## Related Code Files
- Create: `internal/ui/screen_code_paste.go`,
  `internal/ui/screen_code_paste_view.go` (if >200 LOC),
  `internal/ui/screen_code_paste_test.go`
- Modify: `internal/ui/messages.go` (add `CodePastedMsg{Text string}` only)
- Delete: none

## Implementation Steps (tests-first)
1. **RED:** `screen_code_paste_test.go` (use `tea.PasteMsg{Content: ...}`):
   - valid paste: `tea.PasteMsg{Content:"func f(){}\n"}` → cmd yields
     `CodePastedMsg{Text:"func f(){}"}` (normalization trimmed `\n`).
   - empty paste (`Content:""`) → no `CodePastedMsg`; model in error state;
     View contains an empty-input reason.
   - over-cap paste (`Content` >10000 runes) → error state, ErrTooLarge
     reason; no `CodePastedMsg`.
   - binary paste (`Content` with NUL) → ErrBinary reason.
   - recovery: after an error, a subsequent valid `PasteMsg` → emits
     `CodePastedMsg` (state clears).
   - non-Paste msgs (a `tea.KeyPressMsg`) → model unchanged, nil cmd (esc is
     NOT handled here — global Back routes it; do NOT add an esc test).
   - View: waiting state shows the instruction + "esc to cancel" (the hint
     text is informational only — esc is consumed globally); NO_COLOR theme
     → same line structure.
   Run → red (undefined model/msg).
2. **GREEN:** implement the sub-model + messages minimally to pass.
3. Refactor; split view if >200 LOC; keep NO_COLOR parity.
4. `make fmt && make lint && make test-race`. Commit:
   `feat(ui): ScreenCodePaste sub-model (bracketed paste → codetext)`.

## Success Criteria
- [ ] Valid paste → `CodePastedMsg{normalized text}`; invalid → error state
  + reason, retry works. Non-paste msgs are no-ops (esc handled globally).
- [ ] Sub-model emits only `CodePastedMsg` (no app-state mutation, no I/O,
  no cancel message).
- [ ] NO_COLOR layout identical; gofmt/vet/`-race` green; files <200 LOC.
- [ ] One commit.

## Risk Assessment
- `tea.PasteMsg` is `struct{ Content string }` (verified — `model.go:131`,
  `phase09_polish_test.go:193` `tea.PasteMsg{Content: ...}`). Read
  `msg.Content`. Do NOT treat it as a string type.
- Chunked paste (one logical paste split across >1 PasteMsg on some
  terminals): define "each PasteMsg = a complete attempt, last wins"; test a
  second PasteMsg replaces the first. Flag for Phase 5 manual check.
- Error-reason mapping must use `errors.Is` against the codetext sentinels,
  not string matching.
