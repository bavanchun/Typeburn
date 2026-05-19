# Phase 5 Report — Wiring + Home + History
Date: 2026-05-19

## Files Modified

| File | Change |
|---|---|
| `main.go` | `decide()` → 2-return signature; `--text` flag; `codetext.Load`; `codeHintFor` |
| `internal/app/model.go` | `Model` gains `codeText/codeHint`; `New()` +2 params; StartTestMsg handler routes ModeCode to `NewTypingCode` |
| `internal/app/model_settings.go` | `NewFromDisk()` +2 params; `onSettingsChange` passes code fields to `NewHome` |
| `internal/app/model_history.go` | `buildRecord` sets `Length=len([]rune(CodeText))` for code; `ResultMsg.CodeText` forwarded |
| `internal/ui/messages.go` | `ResultMsg` gains `CodeText string` |
| `internal/ui/screen_home.go` | `HomeModel` gains `codeText/codeHint`; `NewHome` +2 params; `modeOrder`/`modeLabels` include ModeCode; `optionCount`/`OptLeft`/`OptRight` guard Code; `startCmd` handles Code (nil when no text) |
| `internal/ui/screen_home_view.go` | `renderOptions` switches on Code → `renderCodeHint()` |
| `internal/ui/screen_typing.go` | Removed `NewTypingCode`/exported accessors (moved to split file) |
| `internal/ui/screen_typing_code.go` | NEW — `NewTypingCode`, `ExportedStartMs`, `ExportedLog` |
| `internal/ui/screen_typing_actions.go` | `completeCmd` forwards `CodeText`; `newTest` uses `NewTypingCode` for ModeCode |
| `internal/ui/screen_typing_view.go` | Selects `RenderCodeStream` for ModeCode with computed `streamHeight` |
| `internal/storage/new_best.go` | `IsNewBest` early-returns false for `mode=="code"` |
| `decide_test.go` | Updated 5 existing tests to `pv, _ := decide(...)`; added 3 new flag tests |
| `internal/storage/history_store_test.go` | Added `TestIsNewBest_CodeMode` |
| `internal/ui/screen_home_code_test.go` | NEW — 9 tests for Code row (hint, no-op enter, startCmd, OptLeft/Right, tab cycle) |
| `internal/app/code_mode_test.go` | NEW — 6 tests (routing, target fidelity, Tab/Enter mid-test, view chars, not-new-best) |
| `internal/ui/screen_home_test.go` | Updated cycle tests for 4-mode order; fixed ShiftTab wrap; Code skipped in range check |
| `internal/app/model_test.go` | `New()` call updated (+2 empty args) |
| `internal/app/smoke_test.go` | `New()` calls updated |
| `internal/app/persistence_notice_test.go` | `New()` calls updated |
| `internal/app/phase09_polish_test.go` | `New()` / `NewHome()` calls updated |
| `internal/ui/phase09_polish_test.go` | `NewHome()` call updated |

## Tests Added

### `decide_test.go`
- `TestDecide_TextFlag_SetsPath` — `--text myfile.go` → textPath="myfile.go", printVersion=false
- `TestDecide_TextFlag_Stdin` — `--text -` → textPath="-"
- `TestDecide_NoTextFlag_EmptyPath` — absent → textPath=""

### `internal/storage/history_store_test.go`
- `TestIsNewBest_CodeMode` — code record never new-best (nil hist, empty hist, non-empty hist)

### `internal/ui/screen_home_code_test.go`
- `TestHome_CodeInModeOrder` — ModeCode present in modeOrder
- `TestHome_CodeLabelInLabels` — "Code" in modeLabels
- `TestHome_TabCycleIncludesCode` — tab eventually reaches ModeCode
- `TestHome_CodeNoText_StartCmdNil` — Enter returns nil when no text
- `TestHome_CodeNoText_SpaceNoop` — Space also no-op
- `TestHome_CodeWithText_StartCmdEmitsMsg` — Enter emits StartTestMsg{Mode:ModeCode, CodeText:...}
- `TestHome_CodeOptLeftRightNoPanic` — no panic, no mode change
- `TestHome_CodeRow_HintNoText` — view contains "pass --text"
- `TestHome_CodeRow_ReadyWithText` — view contains "ready"
- `TestHome_CodeRow_ErrorHint` — custom error hint shown

### `internal/app/code_mode_test.go`
- `TestCodeMode_StartTestMsgRoutesToTyping`
- `TestCodeMode_TypingTargetMatchesCodeText`
- `TestCodeMode_TabDuringTest_DeliveredToEngine` (regression: Tab → RestartSame, not nav)
- `TestCodeMode_EnterDuringTest_DeliveredToEngine` (regression: Enter → engine, not nav)
- `TestCodeMode_ViewContainsCodeChars`
- `TestCodeMode_ResultNotNewBest`

## make test-race Result

```
ok  github.com/bavanchun/Typeburn              1.658s
ok  github.com/bavanchun/Typeburn/internal/app  2.544s
ok  ...all 11 packages                           PASS
```
All packages pass, race-clean.

## Contract Changes (unavoidable)

| Contract | Change | Reason |
|---|---|---|
| `decide()` return | `bool` → `(bool, string)` | Need textPath out of pure function without I/O |
| `app.New()` signature | +2 params `(codeText, codeHint string)` | Thread code state from main without global |
| `app.NewFromDisk()` signature | +2 params | Same; only caller is main.go |
| `ui.NewHome()` signature | +2 params | Thread code fields to HomeModel |
| `ui.StartTestMsg` | +`CodeText string` field | Carry snippet through msg without a separate channel |
| `ui.ResultMsg` | +`CodeText string` field | Forward for rune-count length + restart support |

All call sites updated atomically in this commit. No existing behaviour changed for non-code modes.

## Status: DONE

Phase 5 complete. All 6 success criteria met: `--text` flag wired end-to-end, Home shows Code in cycle with correct hint/no-op behaviour, Tab/Enter delivered to engine during code test (regression tests green), code records persisted and never starred, other modes' goldens unchanged, race-clean, all files ≤200 LOC, one commit.
