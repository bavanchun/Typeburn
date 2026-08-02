# Phase 2 — Typing Viewport, Wrapping, Display Width

Worktree: `/Users/vchun/Codes/My-projects/Typeburn/.claude/worktrees/agent-afc1018e9e3d67f49`
Branch: `worktree-agent-afc1018e9e3d67f49`
Commits: `7b1f8a10`, `11d7a0cc`, `90be502d`, `8bd99431`

Status: DONE

## What changed, per file

| File | Change |
|---|---|
| `internal/ui/word_stream_renderer.go` | `wrapTokens` moved out; `RenderWordStream` now joins the returned rows. 90 LOC. |
| `internal/ui/word_stream_wrap.go` (new) | `cellWidth` (display cells via `lipgloss.Width`, ASCII fast path) + `wordWrapper` + `wrapTokens` → `([]string, caretRow)`. Word-boundary lookahead, hard split kept for over-long words, space folding at breaks. 158 LOC. |
| `internal/ui/word_stream_anim.go` | Takes `height`; routes rows through the existing `joinViewport`. `height ≤ 0` = every row (measuring callers, goldens). |
| `internal/ui/screen_typing_view.go` | `codeStreamHeight(h)` = `h-4`; `wordStreamHeight(h)` = `clamp(3, (h-4)/2, 7)`. Both branches now get a budget. |
| `internal/ui/code_stream_renderer.go` | `cellW := 1` → `cellW := cellWidth(r)`. `joinViewport` untouched. |
| `internal/ui/screen_typing.go` | `tickLoopEnded` field (zero = a chain is running, because the root's `InitCmd` starts one at construction); restart/new-test go through `armTickLoop`. Split to 177 LOC. |
| `internal/ui/screen_typing_tick.go` (new) | `handleTick` + `armTickLoop` — moved to hold `screen_typing.go` under 200 LOC. |
| `internal/ui/screen_typing_input.go` | **Out of the listed file set — see Deviations.** First-keystroke tick start now goes through `armTickLoop`. |
| `internal/ui/frame_fits_known_overflow_test.go` | 34 typing entries deleted. |
| Tests | new `word_stream_wrap_test.go`, `screen_typing_viewport_test.go`, `screen_typing_tick_test.go`; extended `code_stream_renderer_test.go`; `caret_anim_test.go` / `nocolor_layout_invariant_test.go` / `anim_bench_test.go` pass the new `height` arg as 0. |

## Measured line counts (through `TypingModel.View`, seed 42)

| mode | 80×24 | 120×24 | 80×30 |
|---|---|---|---|
| time30 before | 47 | 36 | 47 |
| time30 after | **11** | **11** | **11** |
| time15 before | 47 | 36 | 47 |
| time15 after | **11** | **11** | **11** |
| words10 before | 5 | 5 | 5 |
| words10 after | 5 | 5 | 5 |

`words10` already fitted (10 words ≈ 55 runes = one row). The phase file's table
listed 24/24/30 for it — that was measured through the *composed root* frame,
which pads to `h`; this table is the sub-model frame the ratchet measures.

`time30` target is 3061 runes; the whole thing used to be emitted every frame.

## `knownOverflow` entries deleted (34)

- `typing/time30@` {60,61,72,80,88,120}×{20,24,30} + `60x50,61x50,72x50` + `200x20` — 22 entries, all `Lines` overflow (23–59 lines).
- `typing/cjk@` {60,61,72}×{20,24,30,50} — 12 entries, all `Width: 76` on a ≤72-column terminal.

Nothing was added. `TestFrameFits` is green in both directions, and asserts stale
keys, so the deletions are proven by the harness rather than by inspection.

## Falsification — mutations run, each reverted after

Every one was applied to the production code, tested, and reverted (`git checkout`).

| # | Mutation | Result |
|---|---|---|
| M1 | `renderWordStreamAnim` joins every row (no viewport) | KILLED — `TestFrameFits`, `TestTypingView_*` |
| M2 | `cellWidth` always returns 1 | KILLED — `TestRenderCodeStream_WrapsOnCellWidth`, `TestWrapTokens_MeasuresCellsNotRunes` |
| M2b | same, run against the ratchet alone | KILLED — `TestFrameFits` (the CJK entries) |
| M3 | zero-width runes charged one cell | KILLED — both `*_ZeroWidthRunesTakeNoCell` |
| M4 | word lookahead removed (`flushWord` never pre-breaks) | KILLED — `TestWrapTokens_BreaksOnlyBetweenWords` |
| M5 | `putCell` never breaks | KILLED — `TestWrapTokens_OverlongWordSplits` |
| M6 | folded space dropped even when it carries the caret | KILLED — `TestWrapTokens_CaretOnFoldedSpaceStaysVisible` |
| M7 | space never folded (starts the next row) | KILLED — `TestWrapTokens_ExactFillEmitsNoBlankRow` |
| M8 | `armTickLoop` always returns `tickCmd` | KILLED — `TestTyping_RestartsDoNotStackTickLoops` |
| M9 | caret row never recorded (always -1) | KILLED — `TestTypingView_CaretVisibleAtEveryKeystroke`, `TestWrapTokens_CaretRowMatches…` |
| M10 | code stream back to `cellW := 1` | KILLED — `TestRenderCodeStream_WrapsOnCellWidth`, `TestFrameFits` |
| M11 | `(h-4)/2` → `h*4` | **SURVIVED** — the `clamp` absorbed it; replaced by M11b |
| M11b | `wordStreamHeight` returns `h*4` before the clamp | KILLED — `TestFrameFits`, `TestTypingView_FitsAndKeepsItsFooter` |
| M12 | caret row hardcoded to 0 | KILLED — `TestTypingView_CaretVisibleAtEveryKeystroke` |
| M13 | viewport pinned to the top instead of following the caret | KILLED |

Note M11: the first attempt was a bad mutation, not a weak test — the clamp made
it a no-op. Recorded because "I ran a mutation and it survived" has to mean
something.

## Byte-identity evidence

- **Step 1 (refactor) is byte-identical.** `RenderWordStream` output for 12
  hand-picked targets × 4 widths plus 3 generated targets × 4 widths, captured
  from `HEAD` before the change and diffed after: empty diff. Committed
  separately (`7b1f8a10`) so `11d7a0cc`'s diff is only the behaviour change.
- **Code mode is byte-identical for ASCII.** 505 SHA-256 digests of
  `RenderCodeStream` over 8 widths × 9 heights × 7 caret positions, plus one
  frame exercising all six `CharState`s, captured before the change and
  re-verified at `HEAD`: empty diff. Wide-rune frames change by design.

## Test coverage added

- Caret on screen at **every** keystroke index 0…len(target) at 60×20, 80×24,
  120×32, through the real `View()`, asserted by the exact `RoleCursorBg` token
  (with blink off the caret cell takes no animation override, so the token is in
  the frame iff its row is windowed in). Fixture is 930 runes — the audit's
  ~950-at-60×20 case.
- Window is a contiguous run of the wrapped rows, holds the caret's row, and
  moves at most one row per keystroke, walked forward 500 keystrokes and
  backspaced all the way back out.
- Window is a pure function of the run so far: backspacing to keystroke *i*
  renders the same frame typing to *i* did (no scroll hysteresis).
- No row ends mid-word at 7 widths; over-long word still splits and loses no
  runes; exact fill emits no blank row; caret-bearing space survives the fold.
- Cells not runes: CJK/emoji rows, in both renderers; zero-width and control
  runes cost no column, in both renderers.
- 120-CJK and 60-emoji Code targets fit `m.w` at three sizes.
- Footer is the last line at three sizes, in Time 30 and Words.
- Three `tab` presses + one `ctrl+r` + the first keystroke leave exactly one
  live tick chain, and that chain is still re-arming.

## Deviations and decisions

1. **`internal/ui/screen_typing_input.go` was edited** (2 lines) although it is
   not in the phase's file list. `applyText` started a second tick chain on the
   first keystroke, unconditionally — the same defect the phase names, one file
   over. Guarding only `screen_typing.go` would have shipped a fix that still
   left two loops as soon as the user typed. No concurrent phase owns that file
   (phase 3 owns `typing_log_helpers.go` and `mode_header.go` in `internal/ui`),
   so there is no collision. Flagging it because it is outside the letter of the
   ownership list.
2. **Space folding, not a leading space.** The phase says a wrap-point space
   should not be dropped. Keeping it starts the next row with a blank column, and
   at exact fill it produces a row containing only a space — a whole line of a
   3–7 line window. Resolved by state: an untyped or correct space draws nothing,
   so it is folded into the break; a space that draws (caret, mistyped, extra)
   hangs at the end of its own row. Nothing the user can see is lost.
3. **The viewport is stateless** (`joinViewport` on the caret row), so
   backspacing back over a scroll boundary moves the window back by one row. The
   phase's mitigation wording ("never scroll back on backspace unless the caret
   leaves the top") implies remembered scroll state; a `View` on a value receiver
   has nowhere to keep it, and hysteresis would make the same content render at
   two different offsets. Tested instead: ≤1 row of movement per keystroke in
   either direction, and path-independence.
4. **`screen_typing.go` was split** (`screen_typing_tick.go`) — the guard pushed
   it to 211 LOC.
5. **`ui` package test time under `-race` went 5s → ~24s.** The
   every-keystroke × 3-sizes sweep is O(n²) in target length through
   `lipgloss.Render`. Trimmed where it did not weaken an assertion (secondary
   sweeps use a 500/250-keystroke prefix); the required sweep is intact.

## Gate

```
gofmt -l .                  empty
go vet ./...                clean
go test ./... -race -count=1  all packages ok
make lint                   ok
make build                  ok
make size-check             ok (exit 0)
```

Every non-test file touched is under 200 LOC (largest: `word_stream_wrap.go`,
158). `internal/ui/screen_home.go` is 208 — pre-existing, phase 9 owns it.

## Unresolved questions

1. **The phase's stated viewport height and its stated examples disagree.**
   `clamp(3, (h-4)/2, 7)` gives **7** rows at 80×24, not the 3 the prose claims
   ("80×24 → 3 lines; 120×50 → 7"); `(h-4)/2` only reaches 3 at h ≤ 10, which is
   below the supported floor of 20. The formula was implemented as written,
   because it is the normative line in both the plan and the phase file. If 3
   lines at 80×24 was the actual intent, the formula wants to be something like
   `clamp(3, (h-20)/4, 7)`, and this is a one-line change plus a re-measure.
2. Should the Code branch also cap its height, or keep taking `h-4`? It fits
   today and the phase said not to perturb Code mode, so it was left alone — but
   it means Code and Words now use visibly different amounts of the screen.
3. The `words10` row of the phase's own defect table (24/24/30) does not
   reproduce against `TypingModel.View` (5/5/5). Likely measured through the
   composed root frame. Worth confirming before that table is cited again.
