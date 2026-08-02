# Phase 4 — Input Bounds And Layout Overflows

Status: **completed**. Gate green (`gofmt -l` empty, `go vet ./...`, `go test ./... -race -count=1`, `make lint`, `make build`, `make size-check`).

Worktree: `/Users/vchun/Codes/My-projects/Typeburn/.claude/worktrees/agent-ac42ae5eece2e6df2`
Branch: `fix/input-bounds-and-layout-overflows`
Commits: `4bcce674` (bounds), `56b7e07b` (ui widths)

---

## Per-bound evidence

Every fix reproduced first by mutating the production code and re-running the new
test. Mutation outputs below are the actual failures observed.

### 1. `codetext.Load` read to EOF

- **Before** (`io.ReadAll(r)`): oversize stream fully consumed.
  `TestLoadReader_StopsReadingAtTheBound` → `got err codetext: read: floodReader: caller read past the ceiling` (consumed 8 MiB of an 8 MiB ceiling).
- **After**: `io.ReadAll(io.LimitReader(r, maxBytes+1))`, `maxBytes = 10000*utf8.UTFMax + 3` (BOM allowance) = 40003. Reading 40004 bytes ⇒ `ErrTooLarge`, never a truncation.
- Byte bound also placed at the top of `normalize`, so the in-memory `Normalize` path is bounded before `strings.ReplaceAll` copies anything. Parity with `loadReader` preserved (`TestNormalize_LoadReaderParity` still green, plus a new controls parity test).
- `/dev/zero`: `TestLoad_EndlessDeviceReturns` runs `Load` in a goroutine with a 10 s deadline; passes in ~0 s. Skips if `/dev/zero` is absent.
- The 400 MiB case is asserted as *bytes consumed*, not by writing a 400 MiB file — `floodReader` errors past a ceiling so a regression fails on the byte count instead of on the machine.

### 2. Control runes reached the renderer

- **Before**: bare CR, `0x08`, VT, FF, DEL, `\x1b[31m`, `\x1b]0;…\a`, U+0085 all returned `err == nil`. Reproduced: 8/8 subtests failed with `got err <nil>, want ErrControl`.
- **After**: new sentinel `ErrControl`; `firstControlRune` rejects any `unicode.IsControl` rune other than `\n`/`\t`, checked **after** CRLF→LF so a Windows file normalises rather than being refused.
- CRLF fixtures asserted green (`TestNormalize_KeepsTheTextControlsCodeModeNeeds`: CRLF file, CRLF + blank line, BOM+CRLF, tabs).
- Two error-message mappers gained an explicit branch (`internal/ui/screen_code_paste.go`, `internal/cli/runtime.go`) — see *Files touched beyond the phase list*.

### 3. Replay timestamps / bucket allocation

- **Before**: no timestamp validation. Reproduced by deleting the validation call — all four cases (`negative`, `backwards`, `span`, `end_ms`) failed with "expected a validation error"; `TestLoadReplayInput_RejectsASpanBeforeAllocatingIt` failed with "a 2^32 ms span was accepted".
- **After**: `validateReplayTimestamps` (non-negative, non-decreasing, span ≤ 24 h, `end_ms` within the same window) runs before `metrics.Compute`. The `1<<32` case is asserted by **rejection plus an allocation budget** (`runtime.MemStats` delta < 32 MiB), not by letting the allocation happen. Nothing near `1<<45` is constructed.
- Read also bounded: `readReplayFile` uses `io.LimitReader(f, 8 MiB+1)`; reproduced by removing the check (fell through to `malformed replay log`).

### 4. `per_second` defensive cap — **was NOT already present**

Checked `internal/metrics/per_second.go` at base commit `26702a87` (Phase 3 merged as `cb71bdd0`): `numBuckets := int(maxOffsetMs/1000) + 1` with no cap. Added here.

- **Before**: `TestBucketPerSecond_CapsTheBucketCount` → `got 4294968 buckets, want the cap of 86400`.
- **After**: `maxBuckets = 24*60*60`. Computed in `int64` first so a 32-bit build cannot wrap to a negative `make` length. Events past the cap fold into the last bucket via the existing index clamp — both keystrokes still counted (asserted).

### 5. `Words(n)` cap = 10 000 (user decision, implemented as specified)

- `words.MaxWords = 10000` exported; `Words` clamps at the allocation site.
- `--words` **errors** rather than clamping: `--words must be at most 10000`, `ExitUsage`. Clamping there would silently start a 10 000-word test for someone who typed two billion.
- The clamp is asserted at `MaxWords+1`, deliberately not at 2e9 — a regression must fail the test, not OOM CI. The 2e9 path is covered at the flag layer, where nothing is allocated.

### 6. History sparkline / centring

- **Before**: one cell per record. `TestHistoryView_FitsAndStaysCentred` at 120 records / w=60 → `line 2 is 143 cells`.
- **After**: `trendLine` computes `termW − width("trend  ") − width("  last N tests")` and `downsampleSpark` averages contiguous groups into that many cells. Shape preserved (a strictly rising series stays strictly rising after compression — asserted). `termW <= 0` (unsized model) draws every record.
- Helpers moved to a new `internal/ui/history_sparkline.go` to keep `screen_history_view.go` well under the LOC ceiling.

### 7. History rules / meta line / row gutter

- Rule: `historyRuleW(termW) = clamp(historyRowW, termW−2, 62)`. Reproduced by pinning it back to 62 → `terminal 60: rule is 62 cells`.
- Meta line: now reports the window end, not the cursor. Reproduced → `showing 1–1 of 120` where 14 rows were on screen; after: `showing 1–14 of 120`, clamped to the record count.
- **Extra, not in the phase list:** unselected rows used a 3-cell gutter while the header and selected rows use 2, so a row's columns shifted one cell right whenever the cursor was elsewhere. Found by `TestHistoryRow_MatchesTheDeclaredWidth`. Fixed to 2 (`historyIndentW`), which is what the header already uses.

### 8. Footer

- **Before**: tier switch at `termW >= 72` with no measurement. Reproduced by relaxing the measurement → **only** `home footer at width 72 renders 73 cells` across the whole 60→90 sweep, and `home@72x20 overflows 72x20: {Width:73}`.
- **After**: at/above 72 the full form is built and measured; it falls back to glyphs when `lipgloss.Width(full) > termW`.
- **Design decision kept, deliberately:** the `< 72` narrow tier is *retained*. `internal/ui/width_tier.go` documents `TierNarrow` (60–71) as glyphs-only by design, and the phase's own risk note says "the glyph form is already the design for narrower terminals". Dropping the tier and going purely by measurement would have rendered full labels at 60–71 — a rendering change beyond this phase's remit — and would have contradicted `TestFooter_NarrowCollapsesToGlyphs`, which asserts the narrow form at w=65. That existing test is untouched and still green.

### 9. Settings row width

- **Before**: `gap := 44 - len(label) - len(display)`. `display` is wrapped in `‹ ›` (3 bytes / 1 cell per guillemet), overcounting by 4. Reproduced → rows render **40** cells against the 44-cell rule (the plan said 42; measured 40, because the byte overcount is 4 *and* the 2-cell row indent was never in the formula).
- **After**: `settingsGap` measures in cells and subtracts the 2-cell indent, so a row ends exactly where the rule does. `settingsBlockW = 44`, `settingsRowIndentW = 2` named in `settings_rows.go`.
- A byte-vs-cell unit test pins the measurement directly with a wide-rune value, so the drift cannot return through a CJK theme/value.

### 10. Persistence notice

- **Before**: unbounded. Reproduced → `width 60: notice is 78 cells`, and at the root frame `result/persist-notice@60x30 overflows 60x30: {Width:78} (unlisted)`.
- **After**: `PersistenceNotice(msg, termW, th)`. Drops the dismiss hint first (dismissal is discoverable by pressing anything; the reason is not), then truncates the reason in **display cells** with an ellipsis. `termW == 0` means "size unknown" and is left untrimmed.
- Call site updated in `internal/app/model_view.go` (Phase 3's file, sequenced not concurrent — group B merged before group C).

---

## 60→90 width sweep

Two new sweeps, one cell at a time, both green:

- `internal/ui/frame_fits_width_sweep_test.go` — all 14 `screenCases()` × widths 60..90 at h=24. Width only; height debt stays in `knownOverflow`.
- `internal/app/notice_width_sweep_test.go` — the composed root frame with a notice present, widths 60..90.

Sweep result against the *pre-fix* footer: exactly one width flagged (**w=72**, home footer, 73 cells) — matches the plan's measurement, no off-by-one elsewhere. Pre-fix History rule flagged w=60 and w=61 only.

---

## `knownOverflow` entries

### `internal/ui/frame_fits_known_overflow_test.go` — 36 deleted, 14 remain

Deleted (all now fit, verified in both directions by `TestFrameFits`):

- `history/120@{60,61,72,80,88,120}x{20,24,30,50}` — 24 entries, `{Width: 143}` (sparkline + rule).
- `home@72x{20,24,30,50}`, `home/code-loaded@72x{…}`, `home/code-error@72x{…}` — 12 entries, `{Width: 73}` (footer).

Remaining — **all `result@…` `{Lines: 29}`**, owned by the concurrent Result redesign phase. Not mine; no width entry survives in this map.

### `internal/app/frame_fits_known_overflow_test.go` — 8 deleted, 6 corrected, 42 remain

- Deleted: `home@72x{20,24,30,50}` (footer) and `result/persist-notice@{60,61,72}x{30,50}` (notice was the only failing dimension there).
- Corrected in place, not deleted: `result/persist-notice@{60,61,72}x{20,24}` from `{Lines: 29, Width: 78}` → `{Lines: 29, Width: 0}`. The width is fixed; the Result panel's height is not, and it is not mine.
- Remaining: `result@…`, `result/persist-notice@…`, `transition/early@…`, every one of them `{Lines: 29, Width: 0}`. **No `Width` entry survives in either map.**

---

## Mutations run (falsifiability)

| # | Mutation | Test that failed |
|---|---|---|
| M-a | `LimitReader` → `ReadAll`, byte check → `if false`, control check → `&& false` | `StopsReadingAtTheBound`, 8/8 `RejectsControlRunes` subtests |
| M-b | delete `validateReplayTimestamps` call | 4/4 `RejectsImpossibleTimestamps`, `RejectsASpanBeforeAllocatingIt` |
| M-c | delete the `maxReplayBytes` length check | `RejectsAnOversizeFile` |
| M-d | delete the `nb > maxBuckets` clamp | `CapsTheBucketCount` (4 294 968 buckets) |
| M-e | footer: `Width(full) <= termW` → `<= termW+10` | `RenderFooter_FitsEveryWidth` (w=72), `TestFrameFits` (`home@72x20`) |
| M-f | settings gap: `lipgloss.Width` → `len` | `SettingsRows_EndWhereTheRuleEnds` (40 cells vs 44) |
| M-g | `historyRuleW`: `termW-2` → `historyRuleMaxW` | `HistoryRuleW_NeverExceedsTheTerminal`, `HistoryView_FitsAndStaysCentred`, `TestFrameFits` |
| M-h | `downsampleSpark`: return `vals` unchanged | `HistoryView_FitsAndStaysCentred` (143 cells), `TestFrameFits` |
| M-i | `PersistenceNotice`: ignore `termW` | `PersistenceNotice_FitsTheTerminal`, `TestAppFrameFits` (`persist-notice@60x30` unlisted) |
| M-j | meta line: `to := end` → `to := top + 1` | `HistoryMeta_ReportsTheWindowNotTheCursor`, `HistoryMeta_ClampsToTheRecordCount` |

All mutations reverted; working tree matches the two commits.

---

## Files touched beyond the phase's list

Three, all additive, none owned by a concurrent phase (checked against the plan's ownership table — group C is phases 4, 5, 7; 5 owns `internal/storage/{history_store,atomic_write}.go`, 7 owns `cmd_update*.go`, `exitcodes.go`, `main.go`, `internal/update/*`):

1. `internal/cli/cmd_run_validate.go` — the `--words` upper bound. The success criterion is "`--words` beyond the cap **errors cleanly**", which the generator's clamp alone cannot deliver.
2. `internal/ui/screen_code_paste.go` and `internal/cli/runtime.go` — one `errors.Is` branch each for `ErrControl`. Both already had a `default:`, so this is not required for correctness, but the fallback text ("could not read the pasted text") misdescribes a control-rune rejection.
3. `internal/ui/history_table.go` row gutter — in an owned file; described in §7 above.

Not touched, as instructed: `screen_result*.go`, `result_layout.go`, `result_graph*.go`, `result_comparison_rail.go`, `result_context.go`, `stat_card.go`, `celebration.go`, `testdata/result_baseline_*.txt`, `internal/app/model_history.go`.

---

## CHANGELOG-worthy (user-visible)

- Code mode refuses text containing control characters (ANSI escapes, bare CR, backspace, DEL) instead of starting a test that cannot be completed or exited. Files with CRLF line endings are unaffected.
- Over-large `--text` files and endless streams (`/dev/zero`, an open pipe) now fail fast with a clear message instead of consuming memory or hanging.
- `--words` above 10 000 is rejected with a message instead of exhausting memory.
- `typeburn replay` rejects logs whose timestamps are negative, out of order, or span more than a day.
- History: the trend sparkline is scaled to the terminal, the table rules fit a 60-column terminal, the screen stays centred with a full 200-record history, table columns line up with the header, and the meta line reports the visible window ("showing 1–14 of 120") rather than the cursor.
- Home at exactly 72 columns no longer pushes the screen off centre; the footer falls back to key glyphs when the labelled form does not fit.
- Settings rows are padded to the width of the rules above and below them.
- The persistence/withheld-run notice fits narrow terminals; it drops its dismiss hint, then abbreviates the reason, rather than widening the screen.

## Files changed

Modified: `internal/codetext/codetext.go`, `internal/metrics/per_second.go`, `internal/cli/cmd_replay.go`, `internal/cli/cmd_run_validate.go`, `internal/cli/runtime.go`, `internal/words/generator.go`, `internal/words/for_mode.go`, `internal/ui/footer.go`, `internal/ui/history_table.go`, `internal/ui/screen_history_view.go`, `internal/ui/screen_settings_view.go`, `internal/ui/settings_rows.go`, `internal/ui/persistence-notice.go`, `internal/ui/screen_code_paste.go`, `internal/ui/persistence-notice_test.go`, `internal/ui/frame_fits_known_overflow_test.go`, `internal/app/model_view.go`, `internal/app/frame_fits_known_overflow_test.go`

Added: `internal/ui/history_sparkline.go`, `internal/codetext/bounds_test.go`, `internal/metrics/per_second_bounds_test.go`, `internal/cli/cmd_replay_bounds_test.go`, `internal/cli/cmd_run_words_bound_test.go`, `internal/words/bounds_test.go`, `internal/ui/footer_width_test.go`, `internal/ui/history_layout_test.go`, `internal/ui/settings_layout_test.go`, `internal/ui/frame_fits_width_sweep_test.go`, `internal/app/notice_width_sweep_test.go`

Every non-test Go file is under 200 LOC. The largest is `internal/ui/screen_home.go` at 208, which predates this phase (last touched in `8307ee6c`) and belongs to the docs/polish phase.

---

## Unresolved questions

1. **`--duration` is still unbounded.** It does not allocate (Time mode always uses the fixed 600-word buffer), so it is not the same defect class, but `--duration 2000000000` starts a test that cannot end. Out of this phase's stated scope; flagging it rather than widening scope unasked.
2. **Replay span limit is 24 h, chosen here.** The plan said "a stated max" without naming one. If a longer replay is ever legitimate, this is the constant to move (`maxReplaySpanMs`).
3. **`ForMode(ModeCode, …)` returns `""`.** Explicit beats random prose, but an empty target is still a value a future caller could use without noticing. Turning `ForMode` into a `(string, error)` would be the honest signature; it changes a public contract and was not in scope.
4. **The `/dev/zero` test leaks a goroutine if the bound is ever removed.** The deadline makes the test fail rather than hang, but a regressed build would leave that goroutine reading until the test binary exits. Accepted: the flood test fails first and loudly. A FIFO-based fixture would remove the risk at the cost of a unix-only build tag.
5. **Footer narrow tier retained (see §8).** If the intent was in fact "measure at every width and use labels whenever they fit", the change is two lines and one existing test assertion — but it would render full labels at 60–71, which contradicts `width_tier.go`'s documented `TierNarrow`.
