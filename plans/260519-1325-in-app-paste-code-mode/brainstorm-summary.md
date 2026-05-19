# Brainstorm Summary — In-app Paste for Code Mode

Date: 2026-05-19 · Status: agreed, ready for `/ck:plan`

## Problem statement

Code mode (v1.2.0) only accepts a snippet via CLI `--text <file>`/stdin.
Add an in-TUI paste path so the snippet can be supplied without restarting
the binary. The v1.2.0 seam already injects the snippet as a string and Home
forks on availability — paste slots into that seam, no engine/renderer change.

## Scout-grounded context

- Flow today: `main.go` → `codetext.Load(path)` → `app.New(...,codeText,
  codeHint)` → `Model` → `ui.NewHome(...)`. Home `startCmd` for Code returns
  **nil when codeText==""**; `renderCodeHint()` shows the disabled hint.
- `codetext` exports only `Load(path string)`; the normalize/validate core
  `loadReader(io.Reader)` is **unexported** → must expose a string normalizer.
- `Screen` enum (iota): Home/Typing/Result/Settings/History. Adding a 6th is
  the chosen pattern (vs the `quitPrompt` overlay precedent).
- `tea.PasteMsg` is **already handled** for `screen==Typing` (paste into a
  running test) → bracketed paste already functions in the program; the new
  screen just needs `PasteMsg` routed to it.
- Runtime codeText mutation + Home rebuild pattern already exists
  (`model_settings.go:33` `onSettingsChange` rebuilds Home).

## Locked decisions

1. **New full `ScreenCodePaste`** (6th `Screen`; own sub-model + view +
   routing + nav, like Settings/History).
2. **Bracketed-paste only** — capture one `tea.PasteMsg`, normalize,
   accept/reject. No cursor/typing/editing. Re-paste replaces.
3. **Entry + precedence:** on the Home Code row with no snippet, Enter
   (currently a no-op) navigates to `ScreenCodePaste`. If `--text` already
   supplied a snippet, Code is enabled and Enter starts the test (paste not
   offered that run). CLI path unchanged and takes precedence.
4. **Invalid paste** (`ErrEmpty`/`ErrBinary`/`ErrTooLarge`): stay on
   `ScreenCodePaste`, show the reason, allow paste-again or esc-cancel. No
   silent failure.
5. **Approach A — paste sub-model owns normalization**: it calls
   `codetext.Normalize`, holds its own error/retry state, emits a success
   message (validated text) only when good. App stays thin.

## Forced defaults (approved)

- Valid paste → return to **Home with Code enabled** (snippet ready); user
  presses Enter to start (preserves Home→Enter→test flow; no auto-start).
- `esc` on `ScreenCodePaste` → Home, `codeText` unchanged.
- Screen content: instruction line ("Paste your snippet · esc to cancel"),
  a waiting state, and on error the reason + "paste again" hint. No char
  echo (bracketed paste is not keystroke-echoed).
- On valid paste the app sets `m.codeText`, clears `m.codeHint`, applies it
  to the EXISTING Home via `HomeModel.WithCodeText` (NOT a `NewHome`
  rebuild — that resets modeIdx to DefaultMode and loses the Code row),
  screen→Home with Code still selected. [red-team F3]
- `esc` on the paste screen is handled by the existing global Back handler
  (model_key_handler.go:64 else→Home); no cancel message / no sub-model esc
  handling. [red-team F2]
- `tea.PasteMsg` is `struct{Content string}`; read `msg.Content`.
  [red-team F1]

## Architecture

- **`internal/codetext`:** refactor so `Load(path)` and a new exported
  `Normalize(string) (string, error)` share the existing `loadReader` core.
  Identical rules/caps/sentinels (BOM, CRLF→LF, trim one trailing `\n`,
  empty/whitespace→ErrEmpty, NUL/invalid-UTF8→ErrBinary, >10k runes / >500
  lines→ErrTooLarge). Pure; the string path does no file I/O.
- **`internal/ui/screen_code_paste.go`** (+ `_view.go` if >200 LOC, +
  `_test.go`): a sub-model with states {waiting, error(reason)}. Update
  handles `tea.PasteMsg` (`msg.Content`) → `codetext.Normalize` → on ok emit
  `CodePastedMsg{Text}`; on err set local error state (stay). Does NOT
  handle esc (global Back covers it — F2). View renders instruction/waiting/
  error using existing `theme.Role` styles (NO_COLOR-safe).
- **`internal/ui/messages.go`:** add `NavCodePasteMsg{}` (Home→app: open
  paste) and `CodePastedMsg{Text string}` (paste screen→app: accepted text).
  No cancel message.
- **`internal/app/model.go`:** `ScreenCodePaste` enum value; route
  `NavCodePasteMsg` → screen=ScreenCodePaste; route `tea.PasteMsg` to the
  paste sub-model with `else if screen==ScreenCodePaste` (keep the Typing
  branch byte-intact); `CodePastedMsg` → set `m.codeText`, clear
  `m.codeHint`, apply via `HomeModel.WithCodeText` to the EXISTING Home
  (NOT NewHome — F3), screen=ScreenHome with Code still selected. esc/back
  from paste → ScreenHome is already done by the existing global Back
  handler (`model_key_handler.go:64`) — verify, no new esc code (F2).
  `model_view.go` routes the new screen's View; key delegation is moot
  (sub-model only consumes PasteMsg).
- **`internal/ui/screen_home.go`:** Code-row Enter when `codeText==""` emits
  `NavCodePasteMsg` instead of returning nil. `--text`/enabled path
  unchanged.

## Expected output

`typeburn` (no `--text`) → Home → Tab to Code (shows "press enter to paste a
snippet") → Enter → `ScreenCodePaste` ("Paste your snippet · esc to cancel")
→ user pastes → valid → back on Home, Code enabled → Enter → Code test on the
pasted snippet (full-literal, exact-match, History-but-not-★, all v1.2.0
behaviour). Invalid paste → reason shown on the paste screen, retry/esc.
`--text` supplied → unchanged (Code enabled, Enter starts; paste not used).

## Acceptance criteria

- New `ScreenCodePaste` reachable only via Code-row Enter when no snippet;
  esc returns to Home with codeText unchanged.
- Valid paste → Home, Code enabled, snippet == normalized paste; starting a
  test uses it; persisted Record Mode="code", excluded from ★.
- Invalid paste (empty/binary/over-cap) → stays on screen, shows the matching
  codetext reason; a subsequent valid paste recovers.
- `codetext.Normalize` and `Load` share one core (no rule divergence) — a
  test asserts parity on representative inputs.
- `--text` path and all v1.2.0 Code behaviour unchanged; time/words/quote
  goldens unchanged; gofmt/vet/`-race` green; prod files <200 LOC.
- Ships protected-main PR flow; semver minor → **v1.3.0**.

## Risks

- Bracketed paste must reach the new screen — proven to work for Typing;
  ensure `main.go`/program config doesn't gate it per-screen; add a routed
  PasteMsg test for the paste screen.
- Very large paste pasted as one `PasteMsg` → `Normalize` cap rejects
  cleanly (no partial state) — test the over-cap path.
- Multi-message / chunked paste on some terminals (a paste split across
  >1 PasteMsg) — define behaviour: treat each PasteMsg as a complete attempt
  (last one wins); document; flag for plan research if evidence of chunking.
- Screen-routing regressions (6th screen) — table-test routing for all
  screens incl. the new one; keep Typing PasteMsg branch intact.
- codetext refactor must not change `Load` behaviour — keep its tests green
  unchanged (regression lock).

## Out of scope (deferred)

In-paste editing/cursor/typed input, multi-snippet library, syntax
highlighting, language detection, file-picker UI, replacing a `--text`
snippet from inside the TUI (only the empty-Code path opens paste this round).

## Open questions

None — surface, input scope, entry/precedence, invalid-UX, normalization
ownership, and post-paste nav all locked with the user.
