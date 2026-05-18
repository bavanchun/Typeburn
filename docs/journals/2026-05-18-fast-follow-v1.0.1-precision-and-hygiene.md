# Fast-Follow v1.0.1 — Sub-WPM Precision & API Hygiene

**Date**: 2026-05-18 23:15
**Severity**: Medium (user-visible fix + hygiene cleanup)
**Component**: Storage (precision), Metrics (API)
**Status**: Resolved

## What Happened

Typeburn v1.0.1 shipped the same day as v1.0.0 (24h fast-follow). Two independent fixes: (1) new-best (★) detection now compares float `NetWPM` instead of rounded `int(WPM)`, so 75.4 and 75.0 are no longer tied; (2) removed the always-zero `MissedChars` field from `metrics.Result` and the result screen. Two fullstack agents executed in parallel with strict file ownership (zero shared files), each self-committed. Phase 3 integrated, ran full `-race` regression GREEN, and released (7 assets, checksums verified).

Commit range: `v1.0.0..v1.0.1` (4 code + 2 docs commits).
Release: https://github.com/bavanchun/Typeburn/releases/tag/v1.0.1

## The Brutal Truth

The roadmap's M2 specification was incomplete and would have shipped a silent regression. It stated "old JSON lacks field → unmarshals 0 (acceptable)" — sounding reasonable until you realize that a legacy 80-WPM record unmarshaled as `net_wpm: 0` would be beaten by any new 60.x run. The fix was right (persist float), but the legacy fallback rule was *not* in the roadmap — it emerged during scouting as a critical gap. The plan caught it and added explicit test cases (`legacy fallback: stored {WPM:80, NetWPM:0}; new {WPM:60,NetWPM:60.4} → false`), but this is a sharp reminder that inherited roadmap notes are hypotheses, not specs. Re-derive correctness from the data model every time.

Parallel execution worked cleanly because the plan did real upfront file-ownership partitioning. Two agents touched zero shared symbols (`storage.Record` plumbing vs. `metrics.Result` field removal), integrated first-try with no conflict, and `-race` GREEN. This contrasts with v1.0.0, where the locked plan still hid a vendor-library defect; here, the smaller, partitioned scope made parallelism low-risk and genuinely fast.

One honest boundary note: Phase 1 had to edit `internal/ui/screen_history_view.go` (the real `isBestRow` call-site), which wasn't in the declared ownership table — the table listed `history_table.go` but the comparison lives in the view. The partition was uncontested and the subagent flagged it transparently rather than silently crossing. Lesson: ownership tables should be derived from call-graph, not just the file where the helper is defined.

## Technical Details

**M2 — float precision fix (commit `30917ac`):**
- Added `Record.NetWPM float64` to `internal/storage/history_record.go`; kept `WPM int` for compact display and JSON back-compat.
- New-best detection now uses `effWPM()` fallback: if `NetWPM == 0` (legacy v1.0.0 record unmarshal), use `float64(WPM)` for comparison; else use `NetWPM`.
- Applied in two places: `storage.IsNewBest()` and `ui.bestWPMPerBucket()` (local copy of `eff()` to avoid cross-package export).
- Test case added: 75.4 vs 75.0 in same bucket → new-best wins (was tied before).

**m4 — MissedChars removal (commit `b98fad1`):**
- Deleted `metrics.Result.MissedChars int` field (hard-coded to 0 in `compute.go:78`, no computation target).
- Removed display in `screen_result_view.go` and test refs in `screen_result_test.go` and `test_helpers_test.go`.
- Pure deletion; zero plumbing cost (KISS/YAGNI applied as intended).

**Integration & release (Phase 3):**
- Full `-race` suite GREEN with both changes compiled together.
- `goreleaser check` clean; `make snapshot` → 6 archives + checksums.
- CHANGELOG updated with `[1.0.1]` section (Fixed + Removed). Roadmap M2/m4 status lines updated to "Shipped v1.0.1".
- Disposable dry-run on `v0.0.0-rc.test` pre-release verified 7 assets + non-empty notes before real tag.

## What We Tried

1. **Scout-based file ownership** (plan phase): `internal/storage/*` vs. `internal/metrics/*` vs. `internal/ui/screen_result*`. Checked for symbol sharing (none) and read the call graph carefully to surface the `screen_history_view.go` boundary (found during Phase 1, flagged transparently).

2. **Legacy back-compat rule** (plan design correction): initial roadmap said "accept 0 on old records"; scouting revealed silent regression risk. Plan locked the `eff()` fallback rule with explicit test. Applied in both `IsNewBest` and UI per-bucket calc to keep old and new records on the same scale.

3. **Parallel execution** (Phase 1 + 2 concurrent): separate commits, no merge conflicts, no cross-file edits. Phase 3 verified integration.

4. **Irreversibility discipline** (reused from v1.0.0): disposable dry-run on `v0.0.0-rc.test` before real `v1.0.1` tag; cleanup-deleted the test tag; annotated tag on proven SHA; separate push (never `--follow-tags`).

## Root Cause Analysis

Two failures of specification:

1. **Roadmap under-specification of legacy handling.** The note "unmarshals 0, acceptable" was a hypothesis, not a derived rule. Until you model the comparison (`new 60.x vs. legacy 80 → 0`), you don't see the regression. Plan-time scouting (comparing actual unmarshal behavior to business logic) caught it, but it should have been locked upfront.

2. **File-ownership table derived from file names, not call-graph.** The ownership table listed ownership by file (`history_table.go`, `screen_result_view.go`), but the actual M2 fix required edits to both (history *table* calc AND history *view* comparison site). Reading the call-site code during planning would have surfaced this. File names are surface-level; ownership should be defined by the symbols each phase touches.

Both are planning discipline issues, not execution failures. Parallel execution itself was sound.

## Lessons Learned

1. **Roadmap notes are hypotheses, not specs.** "Accept legacy 0" sounds right until you trace the data flow. Re-derive correctness rules from the actual model (unmarshal behavior + comparison semantics) every time. Lock legacy-compat rules explicitly with test cases.

2. **Parallel safety requires call-graph precision, not file-name precision.** File ownership tables are useful for merge-conflict avoidance but miss cross-file call sites. Define ownership by the symbols (structs, fields, functions) each phase modifies, not by which file the primary struct lives in.

3. **Sub-WPM fairness is a precision issue, not a rounding issue.** Storing only the rounded int was architecturally unsound — the information is lost, and tie-breaking becomes arbitrary. Storing the float adds 8 bytes per record (negligible) and eliminates the ambiguity. For future precision fixes, persist the full value, not the bucketed one.

4. **Parallel execution + file ownership is low-risk when partition is real.** This fast-follow shipped faster than v1.0.0 and with fewer integration surprises, directly because the two fixes touched disjoint code paths (no symbol sharing, no cross-file dependencies within a phase). The v1.0.0 plan was locked but still hid a vendor-library defect — smaller, partitioned scope wins.

5. **Disposable gates for irreversible operations are not optional.** The dry-run caught the GoReleaser `changelog.disable` bug in v1.0.0; here it verified asset count and notes. For append-only operations (sumdb, git tags), dry-run before real is a non-negotiable discipline, not a nice-to-have.

## Next Steps

1. **Monitor** `go install github.com/bavanchun/Typeburn@v1.0.1` proxy (≤~1h lag expected; low priority).

2. **Roadmap review** (follow-up session): audit other fast-follow items for similar under-specification. If M1/M3/M5 have implicit data-model assumptions, surface them now with test cases. Example: "accept defaults" is a hypothesis until you model it.

3. **Plan template enhancement** (documentation): add a checklist item "Derive legacy-compat rules from actual unmarshal + business logic; lock with explicit tests; don't inherit roadmap hypotheses."

4. **File-ownership template**: include call-graph notes (e.g., "Phase 1 edits `history_table.go` AND `screen_history_view.go` because the `isBestRow` site is in the view").

5. **Release runbook** (from v1.0.0, still actionable): bump GitHub Actions SHA pins before 2026-06-02 (Node20 deprecation deadline). Task: `.github/workflows/release.yml` checkout→v5, setup-go→v6, goreleaser-action→v7.

---

**Status**: DONE
**Summary**: v1.0.1 shipped fast-follow (2 parallel fixes) on the same day as v1.0.0. Parallel execution was clean because file ownership was real (zero shared symbols). Legacy back-compat rule was a planning discovery (not in roadmap); locked with explicit test. Full -race regression GREEN; release verified with disposable dry-run.
**Concerns**: Roadmap M2 specification lacked the legacy fallback rule (hypothesis, not spec). File-ownership table was name-based, not call-graph-based; Phase 1 had to edit a file not explicitly listed (transparent flag, no damage, but template should be tighter).
