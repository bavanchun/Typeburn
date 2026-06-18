---
phase: 3
title: "Caret animation"
status: pending
priority: P2
effort: "6h"
dependencies: [2]
---

# Phase 3: Caret animation

## Overview
Make the typing caret feel alive: **blink** (slow cycle), **new-cell fade**
(freshly-typed cell eases accent→normal), and **leave-trail** (the cell just
vacated dim-decays). This is the one feature on the perf-sensitive typing hot
path; the v2 cell-diff renderer keeps it cheap, but it is validated by benchmark in P7.

## Requirements
- Functional: combined blink + new-cell fade + trail on `Current`/recently-typed cells.
- Non-functional: WPM/completion path untouched; NO_COLOR → attribute-only (reverse
  blink, faint fade/trail, no color math); zero layout change; cheap idle.

## Architecture
**State (on `TypingModel`):** `lastKeyMs int64` (set in `applyText`), `frameLoopArmed bool`
(idle→active edge guard), and `nowFn func() int64` (clock seam, defaults to
`time.Now().UnixMilli`; test-overridable — mirrors the existing `seed` injection at
`screen_typing.go:36,60`). No new heap per keystroke. `caretBlinkOn` is **derived**, not stored.

**Blink:** rides the existing 100ms timer tick via time-threshold — `caretBlinkOn =
(nowMs/blinkHalfMs)%2==0` with `blinkHalfMs=265` (530ms cycle, research-confirmed Windows
standard). No fast frames needed for blink alone. **Pre-start blink:** the 100ms `tickCmd`
does not run before the first keystroke today (`Init` nil; `NewTyping` starts no tick), so the
caret would sit static while the user decides to start. Bootstrap a `tickCmd()` on entry to
ScreenTyping (in the `StartTestMsg` handler at `model.go:82`) so the pre-start caret blinks.

**Fade + trail need fast frames:** active only for `fadeMs≈150` after `lastKeyMs`.
`HasActiveAnim(nowMs)` returns `nowMs-lastKeyMs < fadeMs` → frame driver runs 33ms frames
during the post-keystroke window, then self-stops back to 100ms blink cadence when typing pauses.

**Bootstrap on the idle→active edge ONLY (red-team P1):** `applyText` must NOT return
`frameTickCmd()` on every keystroke — overlapping 33ms `tea.Tick` timers would multiply and
never cleanly self-stop. Return the frame bootstrap only when `frameLoopArmed` is false, then
set it true; clear `frameLoopArmed` when `HasActiveAnim` goes false (checked in the FrameTick
handler before deciding to re-arm). Net: exactly one live frame loop regardless of typing speed.

**Rendering integration (`word_stream_renderer.go` + `code_stream_renderer.go`):**
- Extend `RenderWordStream`/`RenderCodeStream` to accept a small `caretAnim` struct
  (`nowMs`, `lastKeyMs`, `blinkOn`, `cursorIdx`) — keep signatures cohesive (one struct, not 4 args).
- **Current cell (cursor):** when `blinkOn` false, render as the non-cursor underlying state
  (cursor "off"); when true, `RoleCursorBg`. Under NO_COLOR, blink toggles `Reverse(true)` on/off
  (already the attr mapping at `theme.go:105`).
- **New-cell fade:** the most-recently-typed correct cell (index `cursorIdx-1`) interpolates
  foreground `RoleAccent → RoleTextMuted` over `fadeMs` via `anim.LerpColor` + `EaseOutQuad`.
  NO_COLOR fallback: render bold for first ~half of fade, then normal (attribute step, not color).
- **Trail (separable sub-step):** the cell at `cursorIdx-2` (just-vacated) renders one notch
  dimmer (`RoleTextFaint`) decaying to `RoleTextMuted` over `fadeMs`. NO_COLOR: `Faint(true)`
  for the window, then normal. Implement blink+fade first, then trail as a clearly-isolated
  addition so the P7 golden review can drop it without touching the rest (it is the weakest UX
  bet per both researchers — but it was a confirmed user choice, so cutting it requires asking).

**Mandatory prefix-token cache (red-team P2 — NOT deferred):** the v2 cell-diff renderer saves
terminal *output bytes* but NOT the per-frame CPU/GC cost of building styled tokens upstream
(`word_stream_renderer.go:42-84` allocates one `Render` string per rune, every frame → ~18k
allocs/s for a 100-word test at 33fps). So the renderer caches the **static prefix** of styled
tokens (everything except the ≤2 animated cells) and only re-`Render`s those ≤2 cells per frame;
the cache invalidates on keystroke (state change) and on resize. This is a P3 design requirement,
not a P7 contingency. Only the ≤2 animated cells get per-frame styles — bounded work, not O(n).

**Blink-setting interaction:** `settings.BlinkCursor` already exists (`screen_typing.go:31`).
If `blink=false`, force `blinkOn=true` always (steady block) but **keep** fade+trail (they are
independent of the blink toggle). Document this so the toggle's contract is clear.

## Related Code Files
- Create: `internal/ui/caret_anim.go` — `caretAnim` struct + `resolveCaretStyles(...)` helper
  (returns the ≤2 animated cell styles given nowMs/lastKeyMs/blinkOn + theme; NO_COLOR-aware).
- Create: `internal/ui/caret_anim_test.go` — fade endpoints, trail decay, NO_COLOR attr fallback, blink phase.
- Modify: `internal/ui/screen_typing.go` — add `lastKeyMs`, `frameLoopArmed`, `nowFn`; set
  `lastKeyMs=nowFn()` in `applyText`; bootstrap `frameTickCmd()` only on the idle→active edge;
  `HasActiveAnim`; `FrameTickMsg` case storing `nowMs`; **reset `lastKeyMs=0` + `frameLoopArmed=false`
  in `restartSame`/`newTest`** (`screen_typing.go:128-134`) so a stale fade never renders on a
  fresh test's first frame.
- Modify: `internal/ui/word_stream_renderer.go` — thread `caretAnim`, apply animated cell styles.
- Modify: `internal/ui/code_stream_renderer.go` — same, for Code mode (deliberate duplication kept).
- Modify: `internal/ui/screen_typing_view.go` — pass `caretAnim` into the renderer.

## Implementation Steps
1. `caret_anim.go`: define struct + `resolveCaretStyles`; pure given theme + times; branch on
   `th.Color(RoleAccent)==nil` for NO_COLOR attr path.
2. `screen_typing.go`: add `nowFn` (default `time.Now().UnixMilli`); record `lastKeyMs=nowFn()`
   in `applyText`; implement `HasActiveAnim`; bootstrap `frameTickCmd()` via `tea.Batch` **only**
   on the idle→active edge (guard with `frameLoopArmed`); add `FrameTickMsg` case; reset caret
   state in `restartSame`/`newTest`.
3. Thread the struct through `screen_typing_view.go` → `RenderWordStream`/`RenderCodeStream`,
   including the static prefix-token cache (invalidate on keystroke/resize).
4. In the renderers, after building cached base tokens, overwrite the ≤2 animated indices with
   `resolveCaretStyles` output. Never change `ch`/width.
5. Tests: inject `nowFn` + pinned `nowMs`; assert NO_COLOR output differs from static only in
   attribute SGR (same runes, same line count, same per-line rune width); fade/trail color
   endpoints; blink phase; prefix cache reused across frames (alloc count bounded).

## Success Criteria
- [ ] Caret blinks at 530ms; fresh cell visibly fades; vacated cell trails — in a color theme.
- [ ] Under `NO_COLOR=1`, caret uses reverse/faint/bold only; **layout-identical** (same runes, line count, rune width) to static.
- [ ] Typing a 100-word test: WPM/accuracy/consistency identical to pre-change (no hot-path semantic drift).
- [ ] Frame loop self-stops within ~150ms of the last keystroke (paused typing → no 33ms ticks).
- [ ] Sustained fast typing keeps exactly ONE frame loop live (no multiplying `tea.Tick` timers).
- [ ] Caret blinks on the typing screen BEFORE the first keystroke.
- [ ] `restartSame`/`newTest` clear caret state — no stale fade on the fresh test's first frame.
- [ ] Caret goldens are deterministic via injected `nowFn` (no wall-clock read in the tested path).
- [ ] `make test-race`, `go vet`, `gofmt -l` clean; new files < 200 LOC.

## Risk Assessment
- **Trail legibility (UX researcher flag):** dim trail may read as noise at high speed. Mitigation:
  keep trail subtle (one faint notch, ≤150ms); if golden review finds it distracting, drop trail and
  keep blink+fade (escalate to user — it was a confirmed choice, do not silently cut).
- **teatest golden churn:** animated frames are time-dependent. Mitigation: tests pin `nowMs`
  explicitly; golden captures use a fixed `lastKeyMs` so output is deterministic.
- **Hot-path cost:** full word-stream rebuild at 33fps. Mitigated by v2 cell-diff; **benchmarked in P7**;
  if regressed, cache the static prefix of tokens.
