# Brainstorm Summary — Typeburn Defect Scan & Fix Scope (2026-05-29)

## Problem Statement
User asked: "check inside my app, tell me what needs to update to fix." Deep code-level defect hunt on Typeburn @ v2.1.2 (post doc-drift cleanup PR #28). Goal: concrete fixes, not style nits.

## Audit Baseline (verified, this session)
- `go test ./... -race` GREEN; `gofmt -l` empty; `go vet` clean; binary 8.7 MB.
- Coverage: runner/theme 100%, typing 96%, metrics 94%, codetext 92%, ui 87%; **lowest = cli/notui 52.3%**.
- 0 TODO/FIXME in code. Only 1 file >200 LOC (`app/model.go` = 204).
- Direct deps: NONE outdated. Pending updates all indirect (ultraviolet, x/text, testify…) — no action.

## Findings (ranked)

### 🔴 MAJOR — `--no-tui` Time mode never auto-ends (SHIP-BLOCKER, confirmed)
- **Loc:** `internal/cli/notui/runner.go:30-61`.
- **Root cause:** `runLoop` blocks on `ReadEvent(rd)` every iteration; `completed()` (Time: `nowMs-start >= Length*1000`) only evaluated AFTER a keystroke. No ticker (TUI uses 100ms `tea.Tick`).
- **Reachable via default path:** `buildRunRequest` sets `mode := settings.DefaultMode` (= Time); `typeburn run --no-tui` with no `--mode` → Time/30s → hits bug.
- **Failure:** user types then stops → test hangs until next keypress (or forever in pipe/script). Late keystroke also corrupts `bucketPerSecond`/`totalTyped` (inflated consistency/char counts).
- **Evidence:** no `runner_test.go` exists — the 52% gap; Time path untested.
- **User confirmed:** `--no-tui` Time mode IS an official flow → ship-blocker.

### 🟡 MINOR — `update.Check` forced path skips ReleaseURL validation
- **Loc:** `internal/update/check.go:50` + `internal/cli/cmd_version.go:157`.
- `cacheLoad` validates `ReleaseURL` vs `releaseURLPrefix` on read; `version --check-update` calls `Check(force=true)` (bypasses cache) then prints URL unvalidated. Low real risk (GitHub API for own repo) but asymmetric.

### 🟢 MINOR / hygiene (bundled per user scope = "all")
- `effWPM` zero-sentinel `storage/new_best.go:41` — harmless today (both 0); harden against future WPM/NetWPM divergence.
- `completion.go:55` misleading comment (logic correct, comment wrong).
- `app/model.go` 204 LOC → split to <200 (convention).

## Agreed Scope (user decision)
**Fix ALL:** MAJOR notui + MINOR update.Check + effWPM hardening + comment fix + app/model.go split. Time-mode = official → MAJOR mandatory.

## Recommended Solution — MAJOR fix (Approach A, chosen)
- Async read on goroutine → channel; `runLoop` `select`s over {event channel, `time.NewTicker`/`time.After(limit)`}.
- Time mode completes on clock even with zero input; `endMs` derived from timer fire (not post-expiry keystroke) → no log corruption.
- Add `runner_test.go` covering Time-mode auto-completion (closes coverage gap).
- Rejected Approach B (`SetReadDeadline` on raw terminal fd) — unreliable/flaky with bufio + raw mode.

## Risks / Considerations
- Goroutine lifecycle: reader goroutine must not leak on completion/abort/ctx-cancel (mirror existing `lifecycle.go` exit semantics; acceptable to leave blocked read at process exit as today).
- Determinism for tests: keep `now func() time.Time` seam; ticker/timeout must be injectable or use the existing time seam so `runner_test.go` is deterministic (no real sleeps).
- Backward compat: Words/Quote/Code paths must remain unchanged (they complete via `Engine.Complete`).

## Success Criteria
- `typeburn run --no-tui` (Time/30s) auto-ends at limit with zero trailing input.
- No late-keystroke inflation in result metrics.
- New `runner_test.go` deterministic, `-race` green.
- update.Check rejects non-prefixed ReleaseURL on all paths.
- `app/model.go` < 200 LOC; comment + effWPM cleaned.
- Full suite `-race` green; gofmt/vet clean.

## Next Step
Hand off to `/ck:plan --deep` (this file as context).

## Unresolved Questions
- Should the ticker resolution for notui Time mode match TUI (100ms) or coarser (1s, since notui re-renders status line per event)? Plan to decide — leaning 250ms-1s to reduce redraw churn on pipes.
