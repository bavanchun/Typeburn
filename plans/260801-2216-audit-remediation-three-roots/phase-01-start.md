---
phase: 1
title: "Correctness Harness And Glyph Fix"
status: done
priority: P1
effort: "6h"
dependencies: []
---

## Outcome

Landed as specified, with four things worth recording because they change what
later phases inherit:

1. **The measured baseline is 84 ui-level entries and 68 root-level ones.**
   Measured fresh, never copied from the audit. `typing/time30` is 47 lines at
   80×24 (width fits — the plan's illustrative `Width: 80` was wrong);
   `history/120` is 143 cells at every width; `home` is 73 cells at exactly 72.

2. **A defect the audit did not have: the persistence toast is unbounded.**
   Only the composed root frame can show it, which is why nothing found it
   before. 78 cells at widths 60/61/72. Assigned to Phase 4.

3. **`NewTyping` seeds its word generator from the clock**, so a frame built
   with it changes line count between runs. The root-level cases use
   `NewTypingCode` with a fixed target instead; the ui-level cases use
   `newTypingWithSeed`. Any later phase adding a typing case must do the same or
   the ratchet goes flaky.

4. **Two maps, not one.** Per-screen size debt is in the ui package; the root
   map holds only what the composition can show — the Result height as the user
   meets it, the same height through a transition, and the toast. Each lives in
   its own file so the debt can shrink without either file passing 200 LOC.

`TestBigDigits_RendersItsOwnDigits` reads the same table it checks and passed
throughout the bug's life. It is kept for the decompose-and-join logic and its
comment says so. Distinctness plus the recorded strip are what actually cover
the artwork.

### Corrected after review

Review caught the harness-fidelity gate proving nothing — the exact failure this
phase exists to eliminate, so it is recorded rather than quietly fixed:

- **The gate compared a live re-render to itself.** It passed with the recorded
  golden replaced by the word `CORRUPTED`, and passed again with the harness
  loading the wrong theme. It now compares the **raw bytes on disk**, escapes
  included; stripping ANSI is what erased the theme difference, so the stripped
  comparison alone could never have caught it. Both mutations now fail.
- **`nowFn` was a no-op.** `TypingModel.View` calls `time.Now()` directly and
  never reads the field. Frames are stable because `startMs == 0`. The
  assignments are gone and the comment states the real reason, so a later case
  that applies keystrokes is not misled into thinking the clock is frozen.
- **`transition/late` measured nothing.** Past the crossfade midpoint its
  stripped output is byte-identical to `result`; its 14 entries were duplicates
  implying coverage that did not exist. Dropped.
- **`knownImplausible` had no stale-key sweep** despite a comment claiming
  parity with the layout ratchets. Added.
- **The app package's `stripANSI` was a scan-to-`'m'`**, which on any non-SGR CSI
  swallows content to the next `m` downstream and would make an overflowing
  frame measure as fitting. Replaced with the state machine and moved to
  `internal/app/strip_ansi_test.go`.
- **Theme parity strengthened** from equal widths to byte equality, and extended
  to NO_COLOR. Equal widths would accept a theme swapping `█` for `▓`.
- **The artwork strip got its own `-update-strip` flag.** A slot swap (3↔7) is
  rectangular, distinct and the same width, so it fails *only* this golden — a
  routine `-update` after a layout change must not be able to re-bless it.

Not addressed, deliberately: the two harnesses duplicate ~90 lines of ratchet
machinery. Go test helpers cannot cross a package boundary, and neither copy
belongs in the shipped binary. `internal/ui/result_layout_test.go` is 230 LOC —
it was already over the ceiling before this phase; Phase 9 owns that sweep.

# Phase 1: Correctness Harness And Glyph Fix

## Overview

Build the missing assertion layer — tests that compare output to *reality*
rather than to another part of the same system — and use it immediately to fix
the glyph table, the cheapest possible proof that the approach works.

## Requirements

- Functional
  - A reusable harness renders any screen at any terminal size, ANSI-stripped.
  - A table test asserts `lineCount <= h` and `displayWidth <= w` for every
    screen × size, with currently-failing combinations enumerated in an
    allowlist rather than silently skipped.
  - `digitGlyphs` is rectangular, and no two digits are byte-equal.
  - `BigDigits(n)` renders the digits of `n` and nothing else.
- Non-functional
  - CI stays green. No behaviour change beyond the glyph data.
  - The harness must be usable unchanged by phases 2, 3, 4.

## Architecture

### Why this phase exists

`go test ./...`, `go vet`, and `gofmt -l` are all green with every audit finding
present. The reason is uniform: **existing tests assert that a component agrees
with itself.** `result_layout_test.go:140,161` compares panel width to
`lay.PanelW`. `screen_result_test.go:440` does the same. The 8 golden files in
`internal/ui/testdata/` recorded `BigDigits(0)` rendering a "2" as the baseline —
golden tests confirm *unchanged*, never *correct*.

### The harness

```go
// renderScreen returns the frame a screen produces at the given terminal size,
// with ANSI stripped. The single place tests obtain rendered output, so a new
// screen is covered by every invariant the moment it is added to screenCases.
func renderScreen(t *testing.T, s screenCase, w, h int) []string
```

Cover all six screens plus the app root and the quit prompt.

**Sizes are scoped to what the product actually supports:** widths
`{60, 61, 72, 80, 88, 120, 200}` × heights `{20, 24, 30, 50}`. `model_view.go:34`
renders `DegradedNotice` below 60×20, so asserting fit at 40×15 asserts against a
configuration the app deliberately refuses — and those cells are unfixable by
design, which would permanently block Phase 9's "map is empty" criterion. 61 and
72 stay: they are the off-by-one boundaries the audit found (`footer.go:25`
switches label form at exactly 72, and the Home footer measures 73 cells there).

Include a **CJK/emoji fixture** in `screenCases`. Without it the harness cannot
catch the display-width defect (`code_stream_renderer.go:124` counts runes where
it means cells) even though it measures with `lipgloss.Width` — the one case
where the harness would silently prove less than it appears to.

### The shrinking allowlist

`main` is protected and CI must stay green, so a failing test cannot land. Encode
the debt instead:

```go
// overflow records how a screen exceeds its terminal, so a fix that trades one
// dimension for the other cannot pass unnoticed.
type overflow struct{ Lines, Width int } // measured values, 0 = fits

// knownOverflow lists screen×size combinations that do not yet fit. The test
// asserts BOTH directions: a listed case must still overflow with exactly these
// measurements, and an unlisted case must fit. A fix that changes the numbers
// fails until the entry is updated or deleted.
var knownOverflow = map[string]overflow{
    "typing/time30@80x24": {Lines: 47, Width: 80},
    // ... enumerated from fresh measurement
}
```

Two corrections over the naive version, both from red-team:

**Record the measured dimensions, not a boolean.** A boolean entry stays
"correctly listed" when a phase fixes the width and introduces a height overflow
at the same key — green CI, invisible regression. `history120@60x20` is
width-only today (`lines=20 maxw=143`); a downsampled sparkline that adds a wrap
row would slip through.

**No phase numbers in the values.** `.claude/rules/review-audit-self-decision.md`
forbids plan IDs, phase numbers, and finding codes in code artifacts. The mapping
from entry to owning phase lives in `plan.md`.

Note what the height assertion can and cannot do: Home, Settings, History and
CodePaste self-place via `lipgloss.Place(w, h, ...)`, so `len(lines) == h`
always. Only width carries signal for those four. Result and Typing are the
screens where the line-count assertion does real work. Say so in the test's doc
comment so nobody later reads more into a green run than it earns.

### The glyph defect

`digitGlyphs[0]` is byte-identical to `digitGlyphs[2]` (`ascii_big_digits.go:12-93`)
— the zero slot holds the "2" artwork. Verified two ways: byte-equality, and
visual comparison against the real ANSI Shadow zero (which has a `██╔═████╗`
second row this glyph lacks).

```
BigDigits(0) -> "2"   BigDigits(60) -> "62"   BigDigits(100) -> "122"
```

Reaches the user at `screen_result_hero.go:28` — the largest element on the
Result screen shows the wrong number.

Second defect, same table: digits 3, 6, 9 have a 9-rune row 4 where the other
five rows are 8, so the hero block is not rectangular:

```
BigDigits(60) row widths: 24, 24, 24, 25, 25, 25
```

The comment at `:9-11` claims "6 characters wide … for uniform column joining" —
wrong on both counts (8 runes, 4 for `1`, and not uniform). Rewrite it to state
the actual contract, which the new test then enforces.

Rectangular glyphs are a **prerequisite for Phase 4** — A2's right-hand rail is
positioned by column arithmetic and would shift one column on digits 3/6/9.

### Metric plausibility helper

Phase 3 needs it; define the shape here so both phases agree:

```go
// assertPlausible fails when a Result reports a value no human could produce.
// Not a formula check — a sanity floor that no amount of coverage supplied.
func assertPlausible(t *testing.T, r metrics.Result)
```

Phase 1 lands the helper plus its own allowlist entry for the AFK case; Phase 3
removes the entry.

## Related Code Files

- Create: `internal/ui/render_harness_test.go` (the harness + `screenCases`)
- Create: `internal/ui/frame_fits_test.go` (table test + `knownOverflow`)
- Create: `internal/app/frame_fits_test.go` (app-root frames incl. transitions)
- Create: `internal/metrics/plausibility_test.go` (helper + bounds)
- Modify: `internal/ui/ascii_big_digits.go` (glyph data + corrected comment)
- Modify: `internal/ui/ascii_big_digits_test.go` (invariants)
- Modify: `internal/ui/result_layout_test.go` — **fixture WPM only** (see below)
- Regenerate: `internal/ui/testdata/result_baseline_*.txt` (8 files — **read the diff, do not auto-accept**)

### The golden fixture is blind to the defect it baked in

`result_layout_test.go:38` is `NetWPM: 74, RawWPM: 74`. **74 contains no `0` and
none of the ragged digits `3`/`6`/`9`.** So the glyph fix produces an *empty*
golden diff, the criterion "golden diff contains digit changes only" passes
vacuously, and a reviewer may read the empty diff as "the fix didn't land".

Change the fixture to a value exercising both defects — e.g. `NetWPM: 106`
(`1`, `0`, `6`). That makes the 8 goldens finally able to see the bug they
recorded as correct for three releases.

Also note `settledPanel` calls `SetSize(termW, 40)` and renders `renderPanel()`
only — height 40, no footer. **No golden exercises 80×24 or the footer at all.**
The goldens are a weaker guard than they look; the frame-fits harness is the real
one. Do not treat a green golden run as coverage of the height problem.

## Implementation Steps

1. Write `renderScreen` + `screenCases`. Verify it reproduces a known-good frame
   byte-for-byte before trusting it.
2. Write the frame-fits table test with an empty `knownOverflow`; run it and
   record exactly which combinations fail. Those measurements populate the map —
   do not populate it from the audit report, measure it fresh.
3. Add the NO_COLOR/mono parity assertion to the same table (strip ANSI, compare
   line counts and per-line widths across themes). It passes today; the point is
   that it keeps passing.
4. Add glyph invariants: rectangular per digit, no two digits byte-equal,
   `BigDigits` output width equals the sum of its digit widths plus separators.
   Confirm they FAIL against the current table.
5. Replace `digitGlyphs[0]` with the correct ANSI Shadow zero; pad rows of 3, 6,
   9 to a uniform 8. Re-run — invariants must now pass.
6. Regenerate the 8 golden files. **Read the diff**: the only changes should be
   digits. Any other delta means step 5 broke something.
7. Add `assertPlausible` with bounds and an allowlist entry for the AFK case.

## Success Criteria

- [ ] `renderScreen` covers all six screens + app root + quit prompt + a CJK/emoji fixture
- [ ] Frame-fits table runs 7 widths × 4 heights (supported range only) × every screen
- [ ] `knownOverflow` populated from fresh measurement, keyed on measured dimensions
- [ ] Every entry has an owning phase recorded **in `plan.md`**; none in code
- [ ] Test fails if a listed case starts passing, and if its measurements change
- [ ] Test doc comment states that height is vacuous for the four self-placing screens
- [ ] Glyph invariants fail against old table, pass against new
- [ ] `BigDigits(0)`=`0`, `(60)`=`60`, `(100)`=`100`, `(90)`=`90` — asserted
- [ ] Golden fixture changed to a value containing `0` and a ragged digit
- [ ] Golden diff non-empty and contains digit changes only
- [ ] `go test ./... -race -count=1` green

## Rollback

Self-contained: revert the commit. The glyph data change and the new test files
have no runtime dependants. Goldens revert with the same commit.

## Risk Assessment

**Risk:** the harness renders something subtly different from the real program,
so every later assertion is measured against a fiction.
**Mitigation:** step 1 gates on reproducing a known-good frame byte-for-byte
before any invariant is written on top of it.

**Risk:** regenerating 8 goldens hides an unintended change in the noise.
**Mitigation:** the glyph fix touches only digit rows; the diff is reviewed, and
step 6 states explicitly what the diff must and must not contain.

**Risk:** `knownOverflow` becomes a permanent exemption list.
**Mitigation:** both-direction assertion makes an unremoved entry a build
failure, and Phase 6's criterion is that the map is empty and deleted.
