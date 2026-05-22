# Planning: Defect Audit Remediation — Six Layered Fixes

**Date**: 2026-05-22 09:00
**Severity**: High
**Component**: notui/core/cli (performance, correctness, concurrency)
**Status**: Planning Complete — Ready for Implementation

## What Happened

Full `--deep` planning run to map remediation for 6 audited defects spanning perf, JSON marshaling, ANSI parsing, HTTP caching, terminal I/O, and package-level race conditions. The session produced a validated 6-phase plan with dependency constraints and a red-team review that surfaced 3 HIGH findings — all now addressed.

Plan location: `plans/20260522-0900-typeburn-defect-fixes/`

## The Brutal Truth

We had latent bugs across three tiers of the system. The painful part: some (like `stripANSI` hanging, `discardEscape` silently swallowing Ctrl-C) are silent failures that only surface under specific input sequences. Others (like `metrics.Compute` called on every keystroke in notui) are pure negligence — we knew it was O(n²) and never cached. The version command's double-emit is inexcusable: a simple encode error got masked by careless error handling. The fact that tests caught none of this is embarrassing; we need test coverage for edge cases like "ESC then Ctrl-C" and "prerelease HTTP timeout."

## Technical Details

**6 Defects Identified:**

1. **MEDIUM-1 (notui-perf)**: `notui/runner.go:50` recomputes `metrics.Compute(log)` on every keystroke — O(n²) over test duration. Live WPM is a stateless pure function; no reason to replay the entire log each frame.
   - Root: No `liveWPM` helper extracted; UI and notui both wrote inline compute-on-change.

2. **MEDIUM-2 (version-json-error)**: `cmd_version.go:104` does `_ = enc.Encode(out); return checkErr` — if `enc.Encode` succeeds but `checkErr` is non-nil, we emit valid JSON *and* error text to stdout.
   - Root: Inverted error handling; should be `return enc.Encode(out)`.

3. **LOW-1 (stripansi-csi)**: `stripANSI` only exits escape mode on `'m'`; non-SGR CSI sequences (e.g., `ESC[A` for cursor up) leave `inEsc=true`, corrupting all subsequent text width calculations.
   - Root: Single-state machine; CSI parsing incomplete.

4. **LOW-2 (update-cache)**: Prerelease window — every TUI launch pays 1.5s HTTP timeout because `check.go:32-40` returns synthetic result without persisting it.
   - Root: Early return in update check skipped cache save.

5. **LOW-3 (notui-reader)**: `discardEscape` buffer-gates reads; split read sequences leave stray `[` typed. Also: HIGH finding — if user presses ESC then Ctrl-C, `discardEscape` swallows the Ctrl-C, making session un-abortable.
   - Root: Buffered I/O assumption wrong; need blocking read + `UnreadByte` guard for Ctrl-C/Ctrl-D.

6. **LOW-4 (globals-race)**: Three mutable package-level vars (`cacheFilePath`, `fetchURL`, `checkFn`) are raw test seams; latent race if any test adds `t.Parallel()`.
   - Root: Lazy thread-safety; tests not parallelized yet, but no mutual exclusion.

## What We Tried

**Red-team review** (10 findings; 3 HIGH, 7 MEDIUM):

- **HIGH-1**: Ctrl-C swallowing via ESC preamble in `discardEscape`. **Applied**: Added `UnreadByte` guard; preserves both 0x03 and 0x04 on escape prefix.
- **HIGH-2**: Phase 6 dependency graph was incomplete — missed Phase 4 edge (new bare-var test site added; Phase 6 must migrate it). **Applied**: Updated DAG; P6 now depends on P2, P4.
- **HIGH-3**: `"context"` import in `cmd_version.go` is certain compile failure, not a risk row. **Applied**: Elevated to Phase 1, Step 0.

**Validation pass**:
- Confirmed `reader.go:29-30`: `0x03` → `EventAbort` (corrected test assertion from wrong `EventRune 0x03`).
- Confirmed `liveWPM` design: wrap in `typing_log_helpers.go`, no churn in `screen_typing.go:108`.
- Confirmed independent phases except P6 dependency on P2, P4.

## Root Cause Analysis

**Perf (MEDIUM-1)**: Premature caching abstention — nobody measured the replay cost; assumed "stateless" meant "cheap."

**JSON double-emit (MEDIUM-2)**: Cargo-cult error swallowing — `_ = enc.Encode(...)` meant "I don't care" rather than "I'll handle the result."

**ANSI parsing (LOW-1)**: Insufficient FSM — SGR-only parser worked for our test inputs, never tested cursor CSI.

**Update cache (LOW-2)**: Copy-paste from old code; early return in sync path skipped cache logic.

**Terminal I/O (LOW-3)**: Buffered I/O abstraction leakage — assumed `bufio.Reader` would buffer all escapes atomically.

**Race (LOW-4)**: Lazy seaming — test seams should have been typed methods from day one.

## Lessons Learned

1. **Measure live hot paths** — don't assume stateless = cheap. O(n²) sneaks in.
2. **Never silence encode errors with `_`** — return the error, handle it at the call site.
3. **FSM parsing must handle all CSI codes, not just SGR** — enumerate states, not just exit tokens.
4. **Prerelease caching matters** — 1.5s is real user friction; same code path as release must be cached.
5. **Buffered I/O needs explicit flush/sync boundaries** — don't assume atomic reads across control codes.
6. **Test seams must be typed (mutex-guarded accessors), not raw globals** — future `t.Parallel()` will expose races.

## Next Steps

**6 independent commit phases, individually revertable:**

1. **Phase 1 (cmd)**: Remove `"context"` import, fix JSON return.
2. **Phase 2 (check.go)**: Add `cacheSave(r)` before early return; add test site mutation guard.
3. **Phase 3 (stripansi)**: 3-state CSI machine.
4. **Phase 4 (notui/reader)**: Add `UnreadByte` guard + blocking-read introducer; add test site mutation guard.
5. **Phase 5 (notui/runner)**: Extract `liveWPM` helper, call directly.
6. **Phase 6 (check.go + reader.go)**: Migrate bare-var test sites to `sync.Mutex` + typed accessors.

**Constraint**: P6 depends on P2 ∧ P4. P1/P3/P5 fully independent.

**Owner**: Implementation team. Timeline: 6 commits, staggered merge to main (one PR per phase).

**Success**: All tests green, no latent races under `go test -race -count=1`, binary size unchanged, notui perf baseline re-measured post-Phase 5.
