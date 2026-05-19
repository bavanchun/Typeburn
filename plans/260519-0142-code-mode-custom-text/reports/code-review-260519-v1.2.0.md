# Code Review — v1.2.0 Code Mode (`feat/v1.2.0-code-mode`)

Date: 2026-05-19 · Reviewer: code-reviewer · Scope: `git diff main...HEAD` excl. 330637c (plans)
Commits: 5c79536, 9b559d3, 45fdb34, f5540eb · Contract: brainstorm-summary.md

## Verdict

Implementation is high quality and faithful to the locked decisions. Pure-core
layering, mode seam, completion, loader normalization, Home disabled state, and
persistence/new-best exclusion all verified correct. **One real correctness
defect found** (HIGH): restart-same of a Code test from the Result screen loses
the snippet text and produces an empty, instantly-completed test — and the
implementation contradicts its own documented intent.

Score: **8/10** · Critical: **0** · High: **1** · Medium: **1** · Low: **2**

---

## (a) internal/codetext — PASS

- `codetext.go:11-19` imports only stdlib (`bytes errors fmt io os strings unicode/utf8`); no config/ui/typing. PASS.
- BOM strip (`:62`) before CRLF→LF (`:69`); single trailing `\n` via `TrimSuffix(s,"\n")` (`:72`) — exactly one, not all. PASS.
- `ErrEmpty/ErrBinary/ErrTooLarge` sentinels (`:35-38`); empty/whitespace (`:74`), binary = invalid-UTF8 OR NUL byte (`:65`), caps 10000 runes (`:25,77`) / 500 lines (`:26,80`). PASS.
- `internal/words` & `internal/typing` diffs: no new `os/io/bufio` imports (grep-verified). PASS.
- Note (LOW): BOM is stripped on raw bytes before the binary guard — a file that
  is *only* a BOM normalizes to `""` → `ErrEmpty` (sensible). Order is correct.

## (b) ModeCode completion + mode seam — PASS

- `completion.go:26-27` `case config.ModeQuote, config.ModeCode: return runesEqual(e.typed, e.target)` — literal `\n`/`\t` are ordinary target runes, exact full-text match, reuses the Quote arm verbatim. PASS.
- `settings.go:25-27` `LengthsFor(ModeCode)` → `nil`. `:67-71` Normalize accepts `ModeCode`. PASS.
- `mode_seam_sync_test.go`: iterates `knownModes` (`:9`), asserts LengthsFor nil-for-code/quote + Normalize round-trip + unknown→ModeTime. Genuinely catches drift **for any mode listed in `knownModes`**. LOW limitation (inherent to this discipline, same as theme sync test): the guard is only as good as the hand-maintained `knownModes` slice — adding a `Mode` const without adding it to `knownModes` is not caught. Acceptable, matches accepted pattern; noted only.

## (c) Renderers — PASS

- `word_stream_renderer.go` byte-untouched — not in `git diff main...HEAD --name-only`. PASS.
- CharState→Role styler duplicated inline in `code_stream_renderer.go:42-64` (NOT extracted) — deliberate per contract. PASS.
- Literal `\n` → new row (`:92-98`); `\t` = 2 cols (`tabVisualWidth=2`, `:15,102-105`); hard-wrap at width, no horizontal scroll (`:108-113`). PASS.
- Viewport scroll-follow clamps top/bottom (`joinViewport :128-146`: `caretRow>=height → start=caretRow-height+1`, clamp to `len-height`, clamp `>=0`). PASS.
- NO_COLOR: only attribute/color swap, row/line structure identical (verified via renderer tests run with `theme.Load(..,true)`). PASS.
- Independently probed edge cases (temp test, removed, no source modified):
  caret on a wrapped continuation row stays in-window (no panic, correct row
  count); `\t` at a wrap boundary never overflows width (`["ab","  c"]` at w=3).
  Wrap + viewport math sound. PASS.

## (d) decide() + wiring + Home/persistence — MOSTLY PASS, 1 HIGH defect

PASS items:
- `decide()` `main.go:32-41` new `(bool,string)` sig; `ContinueOnError`+`io.Discard`; parse error → `return false,""` → TUI launches. Fall-through preserved for unknown/`-h`/`-v`/no-args (covered by `decide_test.go:12-39`). PASS.
- Bad `--text`: `main.go:65-72` sets `codeHint` via `codeHintFor` (`:45-56`), no crash, no `os.Exit` beyond `--version` (`:60-63`). PASS.
- Home no-text: `screen_home_view.go:81-91` shows `"pass --text <file> · in-app paste coming soon"` (matches contract line 44); `renderOptions :66-67` returns hint, NO length cycler for Code; `startCmd :166-169` returns `nil` → Enter no-op. With text → `startCmd :170-173` emits `StartTestMsg{CodeText}`. PASS.
- Tab/Enter reach engine mid-code-test: `model_key_handler.go` & `screen_result*.go` untouched (not in diff); `code_mode_test.go:58-102` proves Tab=RestartSame (startMs→0, stays ScreenTyping) and Enter advances engine log. PASS.
- Record: `model_history.go:21-23` `Mode="code"`, `Length=rune count` (display-only); `new_best.go:63-65` `IsNewBest` short-circuits `r.Mode=="code"` → false; `code_mode_test.go:125-156` asserts no ★. time/words/quote bucket logic unchanged. PASS.
- Renderer dispatch mode-gated `screen_typing_view.go:40-54`; word stream untouched for non-code. PASS.

### HIGH — Restart-same of a Code test from Result screen yields an empty test

`screen_result.go:90-95 restartSameCmd()`:
```go
mode, length, ql := m.mode, m.length, m.quoteLen
return func() tea.Msg {
    return StartTestMsg{Mode: mode, Length: length, QuoteLen: ql}  // CodeText == ""
}
```
`ResultModel` struct (`screen_result.go:18-22`) stores `mode,length,quoteLen` but
**not CodeText**; `NewResult` (`:32-37`) ignores `msg.CodeText`. On the Result
screen, tab/enter (`:71` `RestartSame`/`Start`) calls `restartSameCmd()` which
emits `StartTestMsg{Mode:ModeCode, CodeText:""}`. Root `model.go:84-86` then does
`ui.NewTypingCode("", …)` → `typing.New("", ModeCode, 0)` → empty target →
`Complete()` = `runesEqual(typed,target)` true at 0 chars → **test instantly
completes with an empty snippet**.

Impact: tab/enter "restart same" — the primary post-result action — is broken
specifically for Code mode. Worse, this **contradicts the implementation's own
documented intent**: `messages.go:17` and `screen_typing_actions.go:38` state
"CodeText carries the snippet … to allow ctrl+r restart with the same text" and
`screen_result.go:58` documents tab/enter as "restart SAME test". The data is
plumbed into `ResultMsg.CodeText` but dropped at `NewResult`.

Mitigation that limits severity to HIGH (not Critical): `ctrl+r` (`NewTest`,
`:75-77`) routes to Home via `AbortMsg`; Home retains `m.codeText`, so the
"new test" path still works and the snippet is recoverable. Only restart-same
is broken. No crash/panic (empty test → immediate Result screen again).

Not covered by tests: `code_mode_test.go:125-156` reaches `ScreenResult` but
never presses tab/enter; `screen_result_test.go` has zero code-mode coverage
(grep-verified). This is why CI is green despite the defect.

Recommended fix (not applied — review only): add `codeText string` to
`ResultModel`, set it in `NewResult` from `msg.CodeText`, and in
`restartSameCmd()` emit `StartTestMsg{Mode:mode, Length:length, QuoteLen:ql,
CodeText: m.codeText}`. Add a regression test driving Result→tab for ModeCode
asserting the restarted typing target equals the original snippet.

### MEDIUM — Restart-same length parameter for Code carries display rune-count

`buildRecord` (`model_history.go:21-23`) sets `Length = rune count` for code
(display-only, fine). `completeCmd` (`screen_typing_actions.go:42-44`) forwards
`length=m.length` (0 for code) into `ResultMsg.Length`, so `restartSameCmd`
would pass `Length=0` — harmless because `NewTypingCode` ignores length. No
action needed today, but once the HIGH fix lands ensure the restarted Code test
ignores `Length` (it does: `NewTypingCode` doesn't read it). Flagging so the fix
doesn't accidentally wire rune-count length into the engine. Verify in fix.

## (e) Mechanical — PASS

- Every prod file <200 LOC (max `screen_home.go` 190). PASS.
- `go build ./...` clean; `go vet` clean on codetext/typing/ui/app/config/storage/main. PASS.
- 3 exported-sig changes (`app.New`, `app.NewFromDisk`, `ui.NewHome`) all
  internal-only; 4 prod call sites updated (`main.go:74`,
  `model_settings.go:17,33`, `model.go:61`) + 4 test files; no external API.
  Traced callers: `buildRecord`/`handleResultMsg` (`model_history.go`),
  `onSettingsChange` (`model_settings.go` — preserves codeText/codeHint across
  settings change, `:33`), `startCmd` (`screen_home.go:162`),
  StartTestMsg/ResultMsg consumers (`model.go:82-100`). All consistent. PASS.

## Positive Observations

- Loader I/O boundary cleanly isolates `os`/`io` from pure core; sentinel-error
  + `errors.Is` hint mapping is idiomatic.
- Renderer duplication is deliberate and well-commented; golden-test isolation
  preserved (word stream byte-identical).
- `onSettingsChange` correctly preserves `codeText`/`codeHint` across a live
  settings/theme change — easy thing to drop, handled.
- Mode seam sync test mirrors the established theme-pack discipline.

## Recommended Actions (priority order)

1. **HIGH** Fix `ResultModel` to carry `CodeText`; wire into `restartSameCmd`;
   add Result→tab regression test for ModeCode. (Defect + missing coverage.)
2. **MEDIUM** While fixing #1, confirm restarted Code test ignores `Length`.
3. **LOW** Consider a comment on `knownModes` noting the slice is the manual
   half of the drift guard (or leave as-is — matches theme pattern).
4. **LOW** Optional: codetext could state in its doc that a BOM-only file is
   reported as `ErrEmpty` (current behavior is fine, just undocumented).

## Unresolved Questions

- Is restart-same for Code mode an intended feature for v1.2.0? The
  brainstorm-summary.md does not explicitly require it, but `messages.go:17`,
  `screen_typing_actions.go:38`, and `screen_result.go:58` all document it as
  supported. If restart-same for Code is intentionally out of scope, the fix is
  instead to disable tab/enter restart on the Result screen when
  `Mode==ModeCode` (and correct the misleading comments) — but silently
  producing an empty completed test is wrong either way. Recommend confirming
  intent with the lead before the fix direction is chosen.

---
Status: DONE_WITH_CONCERNS · 8/10 · Critical: 0 (High: 1, Medium: 1, Low: 2)
