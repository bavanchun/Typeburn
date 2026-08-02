---
phase: 2
title: "Typing Viewport, Wrapping, Display Width"
status: done
priority: P1
effort: "1d"
dependencies: [1]
---

# Phase 2: Typing Viewport, Wrapping, Display Width

# Overview

Give the word stream the viewport that Code mode already has, and stop breaking
words in half at line ends. This is the worst user-facing defect in the product:
the default mode on the default terminal loses half its frame.

## Requirements

- Functional
  - Typing renders within its height budget in every mode, at every supported size.
  - The caret is always on screen, at every point in a run of any length.
  - No word is split across a line break.
  - Code mode behaviour is unchanged.
- Non-functional
  - The removed `knownOverflow` entries stay removed.
  - No new dependency; `joinViewport` is reused, not reimplemented.

## Architecture

### The defect

`screen_typing_view.go:46-51` clamps Code mode to `m.h - fixedOverhead` and
routes through `joinViewport` (`code_stream_renderer.go:152`). The `else` branch
at `:53` calls `renderWordStreamAnim` **with no height argument at all**, and
`RenderWordStream` (`word_stream_renderer.go:93-137`) emits every wrapped row of
the entire target.

Measured through the real `app.Model.View()`:

| mode | 80×24 | 120×24 | 80×30 |
|---|---|---|---|
| time30 | 47 lines → 23 clipped | 36 → 12 clipped | 46 → 16 clipped |
| time15 | 47 → 23 clipped | 36 → 12 clipped | 47 → 17 clipped |
| words10 | 24 → fits | 24 → fits | 30 → fits |

`config.Defaults()` is `ModeTime, 30` (`settings.go:43-44`). Words/Quote fit only
because their buffers are naturally short — luck, not adaptation.

Bubble Tea v2 in altscreen draws into a `w×h` cell buffer, so overflow is
**silently dropped**, not scrolled. The footer is unreachable, and past ~950
characters at 60×20 the caret itself leaves the screen — the user types blind.

**The fix is small because the hard part is already written and tested.**
`joinViewport(rows, caretRow, height)` exists, has tests, and does exactly what
is needed. It is simply not called from the non-Code branch.

### Word-boundary wrapping

`word_stream_renderer.go:116-118` documents the second defect outright:

> There is no scan-back to the last space, so a word wider than the line is
> split between runes here.

But the wrap condition `lineWidth+cellW > width && lineWidth > 0` breaks at the
exact cell boundary, so **every** wrap point splits whatever word straddles it,
not only over-long ones:

```
row 2: ... night book shi
row 3: p they boat love ...
```

For a typing test this is severe — the reader's eye tracks whole words, and a
split word destroys the read-ahead that typing speed depends on. Monkeytype,
`tt`, and `toipe` all wrap on word boundaries.

Fix: one-token lookahead in `wrapTokens`. Accumulate to the next space token; if
the pending word does not fit the remaining width, flush first. The token list
already knows word boundaries. A word genuinely wider than the line still
hard-splits — that path must be kept and tested.

### Display width, not rune count (D4)

`word_stream_renderer.go:112` and `code_stream_renderer.go:124` both hardcode
`cellW := 1`. Wide runes therefore consume one accounting cell and two terminal
columns:

```
120 CJK runes, term 80x24 -> rendered line cells=141 runes=72
60 emoji,      term 80x24 -> rendered line cells=120 runes=60
```

The wrapper breaks at 72 runes = 144 cells on an 80-column terminal, so ~61
columns of text the user must type are off screen, and the frame's widest line
destroys centring for the header and footer.

**This belongs in this phase, not elsewhere:** `wrapTokens` is being rewritten
here anyway, and it is the exact site. `word_stream_renderer.go` carries a
`// CJK double-width is not handled — deferred` note; `code_stream_renderer.go`
carries none and is precisely where arbitrary pasted text arrives via `--text`.

Fix: `cellW = uniseg.StringWidth(tok)` (already an indirect dependency) or
`ansi.StringWidth`. Zero-width runes get `cellW = 0` — which also handles the
control characters that survive `codetext.Normalize` (A10, bounded in Phase 4):
today each consumes a layout cell but renders nothing, so rows come out short and
a caret landing on one is invisible.

This phase therefore owns `code_stream_renderer.go` outright, not "verify only".

### Tick-loop arming guard

`screen_typing.go:158-164` returns `tickCmd()` on both `RestartSame` and
`NewTest`, and `handleTick` re-arms unconditionally — no `armed` guard, though
`frameLoopArmed` exists two fields away, so the pattern was known and not
applied. Three `tab` presses leave 4 live loops, each emitting `ResultMsg`.
Persistence does not double (extras are dropped on `ScreenResult`, verified in
the audit), so this is battery/CPU, not corruption. Same file as the viewport
work; fix it here.

### Viewport height — decided: scales with the terminal

```go
// The window grows with the terminal instead of pinning to Monkeytype's 3
// lines, so a tall terminal is used rather than padded. Clamped at 7 because
// beyond that the eye stops tracking the caret's line.
rows := clamp(3, (h-4)/6, 7)
```

80×24 → 3 lines; 120×50 → 7. Scroll by one line when the caret enters the last
visible line.

**Divisor corrected during execution.** This section originally read `(h-4)/2`,
which contradicts both worked examples directly above it: at h=24 it yields 10,
and the upper clamp turns that into 7, not 3. `(h-4)/2` only drops to 3 at
h ≤ 10 — far below the h=20 degraded-mode floor — so the window would have been
**7 rows at every supported height**, never adapting, and the stated trade-off
below would have been bought without ever being paid for.

`/6` is the divisor that satisfies the two examples as written: h=24 → 3,
h=50 → 7. Confirmed with the user during execution. The mapping is asserted at
intermediate heights, since a test that only checks the clamp bounds passes
against the saturating version too.

Accepted trade-off: **the reading rhythm differs between machines** — the same
user gets 3 lines on a laptop and 7 on an external monitor. That is the cost of
using the space, and it was chosen deliberately over a fixed 3.

Consequence for testing: the caret-visibility test must run at every height in
the harness matrix, not just one, because the window size is now height-dependent.
A test that passes at 24 rows proves nothing about 50.

## Related Code Files

- Modify: `internal/ui/screen_typing_view.go` (pass the height budget to both branches)
- Modify: `internal/ui/word_stream_renderer.go` (`wrapTokens` lookahead; return rows)
- Modify: `internal/ui/word_stream_anim.go` (thread the budget through)
- Modify: `internal/ui/screen_typing_test.go`, `internal/ui/word_stream_renderer_test.go`
- Modify: `internal/ui/code_stream_renderer.go` (display width; `joinViewport` itself reused unchanged)
- Modify: `internal/ui/screen_typing.go` (tick-arming guard)
- Modify: `internal/ui/frame_fits_test.go` (delete the typing `knownOverflow` entries)

## Implementation Steps

1. Change `wrapTokens` to return `[]string` rows instead of a joined string.
   Confirm the joined form is byte-identical to today's output for a case that
   fits — this isolates the refactor from the behaviour change.
2. Add the word-boundary lookahead. Test: no line ends mid-word; a word wider
   than the line still splits; a line exactly filled by a word does not emit a
   trailing blank.
3. Thread `streamHeight` into `renderWordStreamAnim` and route the rows through
   `joinViewport` with the caret's row index.
4. Test the caret is visible at every keystroke index from 0 to `len(target)`,
   at 60×20, 80×24, 120×32 — the audit's failing case was ~950 chars at 60×20.
5. Delete the typing entries from `knownOverflow`; the both-direction assertion
   from Phase 1 now proves the fix.
6. Confirm Code mode output is byte-identical to before (it already had a
   viewport; this phase must not perturb it).

## Success Criteria

- [x] Typing fits `w×h` at every harness size, every mode — `knownOverflow` typing entries deleted
- [x] Caret on screen at every keystroke index, verified at 3 sizes
- [x] No line ends mid-word; over-long word still splits; both tested
- [x] Code mode frames byte-identical to pre-change **for ASCII input** (wide-rune frames change by design)
- [x] Footer visible at 80×24 in Time mode
- [x] A 120-CJK-rune and a 60-emoji Code target both fit `m.w`; the CJK fixture entry leaves `knownOverflow`
- [x] Zero-width/control runes consume no layout cell
- [x] 3× `tab` leaves exactly one live tick loop
- [x] `go test ./... -race -count=1` green; files under 200 LOC

## Rollback

Revert the commit. No persisted-data or public-contract change; the only cross-phase
coupling is the deleted `knownOverflow` entries, which return with the revert.

## Risk Assessment

**Risk:** scroll-follow jitters — the window jumps when the caret crosses a
boundary, or oscillates on backspace.
**Mitigation:** scroll only when the caret leaves the visible band, and never
scroll back on backspace unless the caret leaves the top. Test a
type-then-backspace-across-the-boundary sequence explicitly.

**Risk:** the wrapping change alters where words land, invalidating existing
typing tests in a way that hides a real regression.
**Mitigation:** step 1 lands the refactor with byte-identical output, so step 2's
diff contains only the wrapping change.

**Risk:** anim path and static path diverge, so the reveal animation and the
settled frame disagree on row count.
**Mitigation:** both go through the same `wrapTokens` → `joinViewport` path; the
frame-fits harness covers the animated screen too.
