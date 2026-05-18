---
phase: 1
title: "M2 sub-WPM new-best precision"
status: completed
priority: P1
effort: "1.5h"
dependencies: []
---

# Phase 1: M2 sub-WPM new-best precision

## Overview

New-best (★) detection compares `Record.WPM` (`int(math.Round(NetWPM))`), so
75.4 and 75.0 both become 75 and a strictly faster run is not flagged. Persist
the float `NetWPM` and compare on an effective-value that stays correct for
v1.0.0 records that predate the new field. Runs in parallel with Phase 2.

## Requirements

- Functional:
  - `storage.Record` gains `NetWPM float64 \`json:"net_wpm"\``; `WPM int` kept
    for compact display + JSON back-compat.
  - `buildRecord` (model_history.go) sets `NetWPM: msg.Result.NetWPM` (keep the
    existing `WPM: int(math.Round(msg.Result.NetWPM))`).
  - `IsNewBest` and the history-table per-bucket best compare **effective
    NetWPM**, where `eff(r) = r.NetWPM` if non-zero else `float64(r.WPM)`.
    Strictly-greater wins; first-in-bucket wins. New-record value uses the same
    `eff()` (its NetWPM is always populated post-fix, so eff == NetWPM).
- Non-functional: pure functions stay UI-free; files < 200 LOC; no new deps;
  history JSON remains forward/backward readable (extra/missing key tolerated).

## Architecture

```
ResultMsg.Result.NetWPM ─► buildRecord ─► Record{WPM:int, NetWPM:float64}
                                              │
        AppendHistory (unchanged) ◄───────────┤
        IsNewBest(hist, rec): max eff(h) in bucket; rec wins if eff(rec) > max
        ui.bestWPMPerBucket: map[key]float64 of max eff(h); row ★ if eff(row)==max
```

`eff()` is the single back-compat seam: legacy records (`net_wpm` absent →
`0.0`) fall back to their rounded int so a new 60.x run cannot beat a stored
legacy 80. Define it once per package (storage + ui) — tiny, DRY within package
boundary (cross-package sharing would force a new exported helper; not worth it).

## Related Code Files

- Modify: `internal/storage/history_record.go` (add field + doc)
- Modify: `internal/storage/new_best.go` (`eff()` + float compare in `IsNewBest`)
- Modify: `internal/app/model_history.go` (`buildRecord` sets `NetWPM`)
- Modify: `internal/ui/history_table.go` (`bestWPMPerBucket` → `map[string]float64`
  using `eff()`; update the `isBestRow` comparison; **display still `%d` r.WPM**)
- Modify: `internal/storage/history_store_test.go` (add cases)
- Verify/regenerate if needed: `internal/ui/screen_history_test.go` teatest goldens

## Implementation Steps

1. Add `NetWPM float64 \`json:"net_wpm"\`` to `storage.Record` with a doc line
   ("effective value for new-best; legacy records fall back to WPM").
2. `new_best.go`: add unexported `func effWPM(r Record) float64 { if r.NetWPM == 0 { return float64(r.WPM) }; return r.NetWPM }`. Change `IsNewBest` `best` to
   `float64` (sentinel `-1`), compare `effWPM(h)` / `effWPM(r)`.
3. `model_history.go`: add `NetWPM: msg.Result.NetWPM,` to the `Record` literal.
4. `history_table.go`: `bestWPMPerBucket` returns `map[string]float64` using the
   same eff rule (local helper; do not import storage's unexported one). Update
   the caller that derives `isBestRow` to compare `effRow == bucketBest`
   (float equality is fine — same stored value, not recomputed). Keep
   `fmt.Sprintf("%d", r.WPM)` for the visible column.
5. Tests in `history_store_test.go`:
   - sub-WPM: bucket has NetWPM 75.0; new 75.4 → `IsNewBest == true`.
   - tie: new 75.0 vs stored 75.0 → `false`.
   - legacy fallback: stored `{WPM:80, NetWPM:0}`; new `{WPM:60,NetWPM:60.4}`
     → `false` (legacy 80 wins).
   - first-in-bucket → `true`.
6. `go test ./internal/storage/ ./internal/app/ ./internal/ui/ -race -count=1`.
   If `screen_history_test.go` goldens shifted, inspect the diff; only
   regenerate if the change is the intended ★ placement (distinct WPMs should
   keep goldens stable — investigate any unexpected diff before regenerating).
7. `go build ./... && go vet ./... && test -z "$(gofmt -l .)"`.
8. **Commit (this phase's deliverable):**
   `git add internal/storage/history_record.go internal/storage/new_best.go internal/app/model_history.go internal/ui/history_table.go internal/storage/history_store_test.go [internal/ui/screen_history_test.go if regenerated]`
   then `git commit -m "fix: compare float net WPM for new-best so sub-WPM gains earn the star"`
   (no AI refs; do not push — Phase 3 owns push/release).

## Success Criteria

- [ ] `Record.NetWPM` persisted; `WPM` unchanged for display/JSON
- [ ] `IsNewBest` & history-table ★ use effective NetWPM with legacy fallback
- [ ] 75.0→75.4 flagged new best; 75.0 tie not; legacy-80 not beaten by new-60.x
- [ ] storage/app/ui tests GREEN with `-race`; goldens intact or intentionally regenerated
- [ ] build/vet/gofmt clean; files < 200 LOC
- [ ] exactly one `fix:` commit, only the owned files, not pushed

## Risk Assessment

- Legacy `net_wpm:0` false-win → mitigated by `effWPM` fallback + explicit test.
- Golden drift → step 6 inspects before regenerating (no blind `-update`).
- Float equality for `isBestRow` is safe: comparing the *same persisted* value,
  not a recomputation, so no epsilon needed (document this in a code comment —
  explain the invariant, not the plan).
