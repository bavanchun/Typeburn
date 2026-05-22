---
title: "typeburn-defect-fixes-v2.1"
description: "Fix 6 audited defects: notui O(n²) live WPM, version JSON double-error, stripANSI CSI handling, update prerelease cache, notui split-read escape parsing, and unsynchronized package globals."
status: completed
priority: P2
effort: 4h
branch: feat/pro-cli-v2
tags: [bugfix, performance, cli, notui, update-check, race-safety]
created: 2026-05-22
createdBy: "ck:plan"
source: skill
blockedBy: []
blocks: []
---

# typeburn-defect-fixes-v2.1

## Overview

Six independent defect fixes surfaced by audit + deep research. One MEDIUM perf bug, one MEDIUM correctness bug in machine-readable output, four LOW correctness/robustness items. No new dependencies. No behavior changes visible to interactive TUI users except the notui live-WPM speedup. Each phase is self-contained and independently revertable.

All work gated by the CI trinity: `go test ./... -race -count=1`, `go vet ./...`, empty `gofmt -l .` (see `make lint`, `make test-race`).

## Severity & Value

| ID | Severity | User-visible impact |
|----|----------|---------------------|
| MEDIUM-1 | MEDIUM | O(n²) CPU in `--no-tui` mode; lag on long Time-mode tests |
| MEDIUM-2 | MEDIUM | `version --json --check-update` double-emits error → breaks machine parsers |
| LOW-1 | LOW | Non-SGR CSI in result panel corrupts border-title width |
| LOW-2 | LOW | Prerelease window: every TUI launch pays the network timeout budget |
| LOW-3 | LOW | Split-read ESC leaves stray `[` typed in `--no-tui` |
| LOW-4 | LOW | Latent data race if any update/cli test gains `t.Parallel()` |

## Phases

| Phase | Name | Files (owner) | Status | Depends |
|-------|------|---------------|--------|---------|
| 1 | [notui-perf](./phase-01-notui-perf.md) | `internal/metrics/*`, `internal/ui/typing_log_helpers.go`, `internal/cli/notui/runner.go` | Completed | — |
| 2 | [version-json-error](./phase-02-version-json-error.md) | `internal/cli/cmd_version.go`, `internal/cli/cmd_version_test.go` | Completed | — |
| 3 | [stripansi-csi](./phase-03-stripansi-csi.md) | `internal/ui/result_render_helpers.go`, `internal/ui/result_render_helpers_test.go` | Completed | — |
| 4 | [update-cache](./phase-04-update-cache.md) | `internal/update/check.go`, `internal/update/check_test.go` | Completed | — |
| 5 | [notui-reader](./phase-05-notui-reader.md) | `internal/cli/notui/reader.go`, `internal/cli/notui/reader_test.go` | Completed | — |
| 6 | [globals-race](./phase-06-globals-race.md) | `internal/update/cache.go`, `internal/update/client.go`, `internal/cli/cmd_version.go` + their tests | Completed | 2, 4 |

## Dependency Graph & Parallelism

Phases 1, 2, 3, 4, 5 are fully independent and can land in any order. Phase 6 must follow both Phase 2 and Phase 4 (file-ownership overlap + new bare-var test sites added by each).

**File-ownership notes:**
- Phase 2 and Phase 6 both touch `internal/cli/cmd_version.go` (disjoint regions — `renderVersionCheckJSON` vs `checkFn` accessor). Do not run in parallel.
- Phase 4 adds a new test in `internal/update/check_test.go` with a bare `fetchURL` override. Phase 6 migrates that site in its global sweep. Phase 4 must land first so Phase 6 has the complete set of sites to migrate.

```
P1 ──┐
P3 ──┤  (fully independent, any order / parallel)
P5 ──┘
P2 ──┐
     ├──> P6   (P6 depends on both P2 and P4)
P4 ──┘
```

## Cross-cutting Constraints

- **Dependency layering is sacred.** Phase 1 exists specifically because notui cannot import `internal/ui`. The fix moves the formula *down* into `internal/metrics` (pure logic, importable by both ui and notui). Do not introduce a ui→notui or notui→ui edge.
- **No new dependencies** (stdlib + `charm.land/*` + `cobra` + `golang.org/x/*` only). Phase 6 uses `sync` (stdlib) — fine.
- **File size < 200 LOC.** Phase 1 may add a new `internal/metrics/live_wpm.go` rather than growing `compute.go` (currently 163 LOC).
- **No plan refs in code/test names** (per repo rule): name tests by scenario (`TestLiveWPM_BelowGuard`), not by finding code.

## Global Rollback

Each phase is a focused diff revertable via `git revert <phase-commit>` with zero cascade — no phase depends on another's symbols except the documented P2→P6 file-sharing (still independently revertable since they touch disjoint lines).

## Red Team Review

Reviewed 2026-05-22. 10 findings accepted and applied (3 HIGH, 7 MEDIUM):

| Finding | Phase | Severity | Summary | Applied |
|---------|-------|----------|---------|---------|
| 1 | 5 | HIGH | `discardEscape` swallows Ctrl-C/Ctrl-D — session un-abortable after ESC | ✓ |
| 2 | 6 | HIGH | Phase 4 adds bare `fetchURL` test site; Phase 6 `dependencies` must include 4 | ✓ |
| 3 | 6 | HIGH | `"context"` import in `cmd_version.go` is a certain compile failure, not a risk | ✓ |
| 4 | 5 | MEDIUM | `TestReadEvent_StandaloneESC` + `TestReadEvent_EscThenCtrlC` missing | ✓ |
| 5 | 3 | MEDIUM | OSC comment wrong — `ESC]` exits via `stEsc` else-branch, not 0x40–0x7E range | ✓ |
| 6 | 3 | MEDIUM | Missing test case `{"trailing_incomplete_csi", "x\x1b[", "x"}` | ✓ |
| 7 | 4 | MEDIUM | Test code had `defer srv.Close()` + explicit `srv.Close()` — double-close ambiguity | ✓ |
| 8 | 6 | MEDIUM | `cache_test.go:153` bare write not explicitly enumerated in migration list | ✓ |
| 9 | 1 | MEDIUM | Phase-01 cited `screen_result_view.go`; correct file is `screen_result.go:7` | ✓ |
| 10 | 1 | MEDIUM | Phase-01 risk table mis-stated the protection mechanism (`len(log)==0` guard, not loop entry) | ✓ |

## Validation Log

### Session 1 — 2026-05-22
**Trigger:** `--deep` mode post-red-team validation (automated gate)
**Questions asked:** 4

#### Verification Results
- Claims checked: 15
- Verified: 14 | Failed: 1 | Unverified: 0
- Tier: Full (6 phases)
- Failures: phase-05 `TestReadEvent_EscThenCtrlC` expected `EventRune 0x03` but `ReadEvent` maps `0x03` → `EventAbort` (`reader.go:29-30`); corrected to `EventAbort` check before interview.

#### Questions & Answers

1. **[Architecture]** Keep unexported `liveWPM` wrapper in `typing_log_helpers.go` or remove it and update `screen_typing.go:108` directly?
   - Options: Keep wrapper | Remove wrapper
   - **Answer:** Keep wrapper
   - **Rationale:** Zero churn in `screen_typing.go:108` and `screen_typing_test.go:235`; safest minimal diff.

2. **[Scope]** PR strategy for 6 independent phases?
   - Options: One PR per phase | One PR all 6 | Two PRs (P1+P2 then P3-P6)
   - **Answer:** One PR per phase
   - **Rationale:** Each phase independently reviewable and revertable; aligns with plan's rollback design.

3. **[Risk]** Include Phase 6 (globals-race, P3) in this batch?
   - Options: Include | Defer
   - **Answer:** Include
   - **Rationale:** Simple accessor pattern, prevents future CI race; low effort, high prevention value.

4. **[Architecture]** Preserve `0x04` (Ctrl-D) alongside `0x03` (Ctrl-C) in `discardEscape`?
   - Options: Both 0x03 + 0x04 | Only 0x03
   - **Answer:** Both (already in plan)
   - **Rationale:** Symmetric; both are abort/EOF signals that must not be silently dropped.

#### Confirmed Decisions
- liveWPM wrapper kept in `typing_log_helpers.go` — no churn in `screen_typing.go`
- One PR per phase — 6 commits, independently revertable
- Phase 6 included — accessor pattern is the durable fix for the latent race
- Both 0x03 and 0x04 preserved via `UnreadByte` in `discardEscape`

#### Action Items
- [x] Fix `TestReadEvent_EscThenCtrlC` — check `EventAbort`, not `EventRune 0x03` (applied pre-interview)

#### Impact on Phases
- Phase 1: No change — wrapper approach already specified
- Phase 5: Test corrected (`EventAbort` instead of `EventRune 0x03`); no architectural change
- Phases 2, 3, 4, 6: No changes

### Whole-Plan Consistency Sweep
All 6 phase files cross-checked after red-team edits + validation fix:
- Phase 1: `screen_result.go:7` citation correct (metrics import); `len(log)==0` guard confirmed as the protection mechanism
- Phase 2: `cmd_version.go:104-105` verified (`_ = enc.Encode(out); return checkErr` exists at those lines)
- Phase 3: OSC behavioral description corrected (exits via `stEsc` else-branch, not 0x40–0x7E range); `trailing_incomplete_csi` test case added
- Phase 4: `defer srv.Close()` removed from test code; single explicit close at the right point
- Phase 5: Ctrl-C/Ctrl-D guard added; `TestReadEvent_EscThenCtrlC` corrected to `EventAbort`; `reader.go:62` `discardEscape` verified at that line
- Phase 6: `dependencies: [2, 4]` correct; context import elevated to required step 1; `cache_test.go:153` bare write enumerated
- plan.md: Dependency graph updated to reflect P4 → P6 edge; Red Team Review table complete
- Zero unresolved contradictions.

## Definition of Done (whole plan)

- [ ] `make test-race` GREEN
- [ ] `go vet ./...` clean
- [ ] `gofmt -l .` empty
- [ ] `make size-check` passes (no binary-size regression)
- [ ] No new third-party dependency in `go.mod`
- [ ] `docs/codebase-summary.md` updated if any package's public surface changed (Phase 1 adds `metrics.LiveWPM`)
