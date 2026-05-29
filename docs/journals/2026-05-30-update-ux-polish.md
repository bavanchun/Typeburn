# Update UX Polish: Discovery Path, Progress Feedback, Release Notes

**Date**: 2026-05-30 15:45
**Severity**: Medium (UX improvement, shipping with v2.3.0)
**Component**: CLI / UI (Result Screen + Update Command)
**Status**: Resolved (merged PR #36, commit 377fdbe)

## What Happened

Shipped three UX refinements to `typeburn update`: (1) fixed a stale footer hint on the result screen that misdirected users to the check-only `version --check-update` instead of the actual `update` command; (2) added live progress feedback during the ~9MB download/verify/swap cycle (was silently frozen); (3) surfaced release notes URL in both `update` and `update --check` output. All CI gates green; docs updated (CHANGELOG, README, codebase-summary, cli-reference).

## The Brutal Truth

The hint defect exposed a critical gap in the app's own discovery path. The `typeburn update` feature shipped in v2.3.0, but v2.2.0 users who saw the footer hint and typed `typeburn update` silently got the TUI instead — because Typeburn treats unknown root args as "launch the app" by design. The command existed, the hint existed, but they never connected. A user had to reverse-engineer the command from release notes. The real fix is one string, but the *feeling* is frustration that we shipped a feature nobody could discover from inside the app.

## Technical Details

**Phase 1 — Hint Fix:**
- File: `internal/ui/screen_result_view.go` + `screen_result_test.go`
- Change: repointed footer hint from `typeburn version --check-update` to `typeburn update`
- Rationale: `version --check-update` reports available updates but doesn't actually update; `update` is the action command

**Phase 2 — Progress Feedback:**
- New file: `internal/update/progress.go` with `Stage` enum (downloading, verifying, installing)
- Nil-safe `report(stage)` helper to emit stage labels
- Threaded trailing `func(Stage)` callback param through `Apply() → downloadVerified()` call chain
- CLI prints stage lines like `  downloading...` in real-time; reporter kept strictly outside verify/replace logic to preserve checksum-only trust model
- Signature change rippled to `applyFn` type, `setApplyFn`, `recordingApply` test stub, all call sites in apply_test and download_test
- Trade-off: single callback param (KISS) over Options struct; enumerated all callers first to confirm ripple cost was acceptable

**Phase 3 — Release Notes:**
- Reused existing `Result.ReleaseURL` field (already populated from GitHub API)
- New `printReleaseNotes(url)` helper that guards on non-empty URL
- Applied in both `cmd_update.go` (pre-confirm) and `cmd_version.go` (check output)
- Wording mirrors existing `version --check-update` pattern

## What We Tried

1. **Initial scope:** hint only. Realized silent download freeze was usability debt.
2. **Progress design options:** (a) global progress writer (too loose), (b) Options struct (heavier), (c) trailing callback (settled here; minimal coupling, easy to test with nil-safe noop)
3. **Release notes placement:** considered adding to TUI footer hint, rejected (width-capped, already crowded)
4. **Test coverage:** added `TestReportStage` for progress emission; added regression assert for hint string in result view test

## Root Cause Analysis

**Discovery Gap:** Treating unknown root args as TUI launch (v1 alias behavior, intentional design) was correct for the REPL but created a silent trap for commands that didn't exist in an old binary. The hint was the bridge, but it pointed at the wrong target (`--check-update` is a status check, not an action). Root cause: hint was written before the command was built, never double-checked against shipped behavior.

**Silent Download:** Progress was deprioritized during the initial `update` implementation (Phase 1–3 planning was complete, code review gates met). This is acceptable shipping debt for v2.3.0 (the command works), but users experience a frozen terminal and might ctrl-c thinking it's hung.

**Release Notes Omission:** Result.ReleaseURL existed in the data but wasn't surfaced to the user. Low-hanging fruit that should have been in the original PR.

## Lessons Learned

1. **Discovery paths are themselves a feature.** Shipping a command doesn't ship the knowledge of it. In-app hints must be validated against the actual command signature, not just the intent. Treat hint strings like API contracts — if it says "run X", verify X exists and does what the hint claims.

2. **Progress feedback prevents panic.** A 9MB download with no output feels broken even when it's working. Structured stage reporting (downloading → verifying → installing) costs one callback param but eliminates the "is it stuck?" anxiety. Worth it.

3. **Callback over config struct when the chain is short.** The ripple to 4 call sites was manageable; a new field in a broad Options struct would have been harder to test and audit. Keep signature changes shallow when possible.

4. **Trust model isolation is non-negotiable but invisible.** Keeping the progress reporter outside verify/replace logic means we're not introducing a code path that bypasses checksums. Easy to miss in review if you're not thinking about it; hard to fix after merge. Callout in code review saved us.

## Next Steps

- **Merged and shipped.** PR #36 squash-merged to main; commit 377fdbe in main.
- **Follow-up (M1, optional):** `version --check-update` currently prints bare `Release notes:` on empty URL. Consider applying the same guard (`printReleaseNotes` pattern) for parity. Out of scope for this PR but noted for v2.4 backlog.
- **Docs:** README updated with updated hint quote; codebase-summary and cli-reference synced; CHANGELOG entry for [Unreleased].
- **Testing:** gofmt clean, go vet clean, go test -race PASS, size-check PASS. All Go files < 200 LOC.

## Unresolved Questions

- (a) Should `version --check-update` apply the same empty-URL guard as the new `printReleaseNotes` for full parity, or is the bare line acceptable for that command?
- (b) Should a managed-install message (e.g., "installed via Homebrew, run `brew upgrade typeburn`") also surface release notes, or is the command hint enough?
