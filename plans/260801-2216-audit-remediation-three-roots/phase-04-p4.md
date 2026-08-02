---
phase: 4
title: "Input Bounds And Layout Overflows"
status: pending
priority: P1
effort: "1d"
dependencies: [1]
---

# Phase 4: Input Bounds And Layout Overflows

## Overview

Put a bound at every unbounded input, and fix the fixed-width layout constants
that overflow narrow terminals. Grouped because they are the same defect shape —
a size taken from data rather than from the container — and because they land in
files no other phase owns.

## Requirements

- Functional
  - Every untrusted input is bounded before allocation.
  - No screen has a hardcoded width exceeding its terminal.
  - Control characters cannot make a Code target uncompletable.
- Non-functional
  - Bounds go at the boundary; `bucketPerSecond` additionally gets a defensive
    cap because it is the shared callee of an untrusted path.

## Architecture

### Unbounded inputs

| Boundary | Defect | Fix |
|---|---|---|
| `codetext.go:64-69` | `io.ReadAll` to EOF; the 10 000-rune cap is only checked at `:92`. A 400 MiB file is rejected **after allocating 1274 MiB**. `Load("/dev/zero")` **never returns** — same for `--text -` on a pipe that never closes. | `io.ReadAll(io.LimitReader(r, maxRunes*utf8.UTFMax+1))`; reject when the result reaches the cap, so oversize stays an error rather than a silent truncation |
| `per_second.go:31-43` | `numBuckets = maxOffsetMs/1000 + 1`, unvalidated. 2 keystrokes at `0` and `1<<32` → 197 MiB; at `1<<45` the test binary was **OOM-killed by the kernel**. Realistic non-adversarial trigger: a log mixing `0` with epoch-ms timestamps → ~63 GiB. | validate in `loadReplayInput` (monotonic, non-negative, span ≤ a stated max) **and** cap `numBuckets` defensively |
| `generator.go:56-65` | `Words(n)` unbounded; `--words 2000000000` is an OOM. `cmd_run_validate.go:85-89` checks only `> 0`. | cap in `Words`, where the invariant lives (proposed 10 000 — open question 3) |
| `screen_history_view.go:38-45` | sparkline emits one cell per record, cap 200 → **143 cells on an 80-column terminal**; the `last 120 tests` label is cut, and because `lipgloss.Place` sizes from the widest line the **whole History screen loses centring** | downsample to `m.w - len("trend  ") - len("  last N tests")` |
| `persistence-notice.go` | Found by Phase 1's root-frame harness, not by the audit. The toast is centred with `lipgloss.PlaceHorizontal(m.w, …)` but never bounded, so a long reason spills past a narrow terminal: measured **78 cells at widths 60/61/72**. `model_view.go:62-69` overwrites the frame's last row with it, so the spill widens the whole frame. | take a width and truncate/wrap inside `PersistenceNotice`; pass `m.w` at the call site |

The persistence-notice call site is `internal/app/model_view.go`, which Phase 3
owns. Phase 3 is in group B and this phase is in group C, so the edit is
sequenced, not concurrent.

**Correction on the sparkline fix:** the original plan said to reuse
`bucketSamples` from `result_graph_axes.go`. That file belongs to Phase 6, and
the signature does not fit anyway — `bucketSamples(perSec []metrics.PerSecond,
secPerCell int)`, not `[]float64`. Write a small local helper in
`screen_history_view.go`; do not reach into Phase 6's file.

### Control characters reach the renderer (A10)

`codetext.go:76-98` rejects only invalid UTF-8 and NUL. All of these are accepted
with `err == nil`: bare CR, `0x08`, vertical tab, DEL, ANSI CSI `\x1b[31m`, ANSI
OSC `\x1b]0;…\a`.

Functional consequence: `Engine.Complete` for `ModeCode` requires an exact rune
match, and `screen_typing_input.go:33` only feeds runes arriving as `k.Text`.
`\x1b` is bound to Back and `\x08` to Backspace — **those runes can never be
produced**, so any snippet containing one is an unwinnable test the user cannot
finish or escape from.

Reached via `--text <file>`, `--text -` (a piped download), and clipboard paste.
`result_render_helpers.go:130` proves the project already knows ESC needs
stripping; it just is not applied at the boundary.

Fix: in `normalize`, reject or strip every control rune except `\n` and `\t` —
the two the package already documents as intentional. Phase 2 independently gives
zero-width runes `cellW = 0`; that is defence in depth, not a substitute.

### Hardcoded widths that overflow

| Site | Measured | Fix |
|---|---|---|
| `history_table.go:26`, `screen_history_view.go:69` | `strings.Repeat("─", 62)` — `history7 60x20 maxw=62`, overflowing the advertised 60-col minimum and killing centring | `min(62, m.w-2)` |
| `footer.go:25` | switches to full labels at `termW >= 72` but never measures the result; the 6-hint Home footer is 73 cells — `home 72x{20,24,30,50} maxw=73`. A 60→90 sweep flags **only** w=72 | build the full form, then fall back to the glyph form if `lipgloss.Width(full) > termW` |
| `screen_settings_view.go:86,97` | `gap := 44 - len(row.label) - len(display)` uses **byte** length; every value is wrapped in `‹ ›` (3 bytes / 1 cell each), so `len` overcounts by 4 and rows render 42 cells against a 44-cell rule | `lipgloss.Width` |

These three are why Phase 1's `knownOverflow` could not otherwise reach empty —
red-team's central objection. This phase owns all of them.

### Also here

- `history_table.go:81-89`: `to := sel + 1` reports the **cursor**, not the
  window — `showing 1–1 of 120` with 14 rows on screen.
- `words/for_mode.go:27`: `default:` swallows `ModeCode` and returns a random word
  buffer. `runner.NewCodeSession` bypasses it today, so it is a trap for the next
  caller, not a live bug. Make `ModeCode` explicit.

## Related Code Files

- Modify: `internal/codetext/codetext.go`
- Modify: `internal/cli/cmd_replay.go`, `internal/metrics/per_second.go` *(defensive cap only — coordinate with Phase 3, which owns this file for the consistency fix; if the phases overlap in time, the cap moves to Phase 3)*
- Modify: `internal/words/generator.go`, `internal/words/for_mode.go`
- Modify: `internal/ui/screen_history_view.go`, `internal/ui/history_table.go`
- Modify: `internal/ui/footer.go`, `internal/ui/screen_settings_view.go`, `internal/ui/settings_rows.go`
- Modify: `internal/ui/frame_fits_test.go` (delete the history/home/settings entries)

## Implementation Steps

1. Reproduce each bound failure first. `/dev/zero` needs a test timeout, not a
   test that hangs the suite.
2. `codetext`: `LimitReader` + control-rune rejection. Verify `Load` and
   `Normalize` still share one core (CLAUDE.md's claim, verified true today).
3. Replay validation + defensive bucket cap.
4. `Words` cap.
5. Sparkline downsampler (local helper, not Phase 6's).
6. The three hardcoded widths; sweep 60→90 to confirm no new off-by-one.
7. History meta line; `for_mode.go` explicit `ModeCode`.
8. Delete the corresponding `knownOverflow` entries.

## Success Criteria

- [ ] `codetext.Load("/dev/zero")` errors promptly (test has a timeout)
- [ ] 400 MiB file rejected without allocating it
- [ ] Control runes other than `\n`/`\t` rejected; a snippet with `\x1b` cannot be created
- [ ] Replay JSON with a `1<<32` span rejected before allocation
- [ ] `Words(n)` bounded; `--words` beyond the cap errors cleanly
- [ ] History sparkline fits `m.w` at 200 records; History stays centred
- [ ] `history7@60x20`, `home@72`, `settings` entries deleted from `knownOverflow`
- [ ] 60→90 width sweep clean for every screen
- [ ] History meta line reports the window
- [ ] `go test ./... -race -count=1` green

## Risk Assessment

**Risk:** the control-rune rejection breaks a legitimate snippet (e.g. a file
with CRLF line endings).
**Mitigation:** normalize CRLF → LF rather than rejecting it; only *bare* CR and
the non-`\n`/`\t` controls are rejected. Test with a CRLF fixture.

**Risk:** `per_second.go` is shared with Phase 3.
**Mitigation:** stated above — if the phases are concurrent, the defensive cap
moves into Phase 3 and this phase keeps only `cmd_replay.go`'s validation.

**Risk:** the footer fallback makes 72-column terminals lose the labelled hints,
which reads as a downgrade.
**Mitigation:** it is the correct behaviour (the labels do not fit); the glyph
form is already the design for narrower terminals.

## Rollback

Revert the commit. Bounds are additive and no persisted format changes, so the
revert is clean. The deleted `knownOverflow` entries return with it.
