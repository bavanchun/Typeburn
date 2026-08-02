# Phase 7 — CLI and update hardening

Status: complete. Gate green (`gofmt -l .` empty, `go vet ./...`, `go test ./... -race -count=1`, `make lint`, `make build`, `make size-check`).
Cross-compile checked: `GOOS=windows GOARCH=amd64 go build ./... && go vet ./...`.

---

## B1 — plain path had no signal handling

**Repro** — subprocess running the real `typeburn update --yes` (real `update.Apply`, real O_EXCL lock, only the release check stubbed), asset download routed through an `httptest` CONNECT proxy that never answers. Parent waits for the lock file to appear, then signals.

Before (`internal/cli/cmd_update_signal_test.go` against unmodified code):

```
=== RUN   TestUpdate_PlainPathSurvivesSignals/terminated
    terminated: lock file leaked, every future update refuses to start
        output:
        updating dev → v9.9.9 ...
          checksums...
    terminated: partial artifacts left behind: [.typeburn-update.lock]
=== RUN   TestUpdate_PlainPathSurvivesSignals/interrupt
    interrupt: lock file leaked, every future update refuses to start
    interrupt: partial artifacts left behind: [.typeburn-update.lock]
--- FAIL: TestUpdate_PlainPathSurvivesSignals
```

After — child output captured from the same test:

```
interrupt child output:
  updating dev → v9.9.9 ...
    checksums...
  ERR update cancelled; nothing was installed
  EXIT 3

terminated child output:
  updating dev → v9.9.9 ...
    checksums...
  ERR update cancelled; nothing was installed
  EXIT 3
```

Lock gone, install dir back to just the binary, exit code = `ExitAbort` (3).

**Fix** — new `applyPlain` in `internal/cli/cmd_update_apply.go`: `signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)`, Apply moved onto a goroutine, `select` on results vs `ctx.Done()`, cancel path routed through the existing `stopApply` → `reportApplyResult`. Existing copy reused verbatim; `errStoppedTooLate` / `errDrainTimeout` ordering untouched.

`cmd_update_run.go` was over 200 LOC after the change, so apply machinery (`applyPlain`, `applyAnimated`, `stopApply`, sentinels) moved to `cmd_update_apply.go`. `cmd/typeburn/main.go` unchanged — the handler belongs where the cancellable work is, not in the fang entrypoint, and putting it there keeps both branches on one contract.

## B2 — recovery needed manual deletions

**Repro** (throwaway scaffold, deleting only the file each error named):

```
attempt 1: update: another update is already in progress (remove …/.typeburn-update.lock if stale)
attempt 2: update: create download file: open …/checksums.txt: file exists
attempt 3: update: create download file: open …/typeburn_9.9.9_linux_amd64.tar.gz: file exists
attempt 4: update: create extracted binary: open …/typeburn.new: file exists
attempt 5: SUCCESS
```

Four manual deletions, one of them hinted.

**After** — same seed with a lock naming a dead PID: one run, zero deletions (`TestApply_RecoversAbandonedUpdateInOneRun`). A legacy ownerless lock from the old format costs zero deletions too, once past the 5s write grace.

**Fix** — `internal/update/lock.go`:
- lock file now carries `pid <n>` / `taken <unix>`;
- `reclaimStaleLock` removes the lock only when the owner is not alive, or when it is older than `lockMaxAge` (15 min — a whole update is bounded by a 90s HTTP timeout, so anything older is wedged or a recycled PID);
- an ownerless lock (older format, or one caught between the O_EXCL create and the record write) is honoured within `lockWriteGrace` (5s) and reclaimable after;
- after the lock is held, `sweepUpdateArtifacts` clears `checksums.txt`, `typeburn{,.exe}.new`, `typeburn_*.{tar.gz,zip}` — matching only names the updater itself writes.

Liveness probe split per platform (`lock_process_unix.go` signal-0 with EPERM counting as alive so a root-held lock is not stolen; `lock_process_windows.go` `os.FindProcess`).

Not over-eager: `TestAcquireUpdateLock_RefusesWhileTheOwnerIsAlive` seeds a lock owned by a real spawned `/bin/sleep 60` and asserts both the refusal and that the lock file survives.

## B3 — decision: deleted

`restoreInterruptedUpdate` had no caller (`grep -rn restoreInterruptedUpdate --include='*.go' .` → definitions only). Deleted from both `replace_unix.go` (a no-op there anyway) and `replace_windows.go`.

Decisive reason beyond "no caller": the function recovers the state *target missing, target.old present*. A startup recovery pass can only run when typeburn is launched **from target** — so in the exact state it repairs it can never execute. Wiring it into `main()` would have added an `os.Executable()` + stat to every TUI launch to service a branch that is unreachable by construction.

`replace_windows.go:13`'s comment claimed that function cleans up the leftover `.old` "on the next launch". Replaced with what actually happens: the leftover is cleared by `os.Remove(old)` at the top of the next `replaceBinary`, and nothing clears it before then — plus an explicit note on why no startup recovery exists, so it is not re-added by reflex.

## B4 — tag validated

One line in `check.go` after `FetchLatest`: `validSemverRe.MatchString(rel.TagName)`. Framed as a maintainability guard, not a breach — an attacker who can forge the API response can already serve a matching-checksum binary, so traversal buys nothing. What it buys *us* is that the "a tag is always `v1.2.3`" invariant is stated rather than inferred from `parseSemver`'s `strconv.Atoi` rejecting everything else.

Covered both ways: `TestCheck_RejectsATagThatIsNotAVersion` (separators, `latest`, empty) and `TestCheck_AcceptsPublishedTagShapes` (`v2.9.0`, `2.9.0`, `v2.9.0-rc.1`, `v2.8.1-3-gabc1234`) so the guard cannot silently start rejecting real releases.

## Also fixed

- `exitcodes.go`: `errors.As` instead of the bare type assertion. `ExitAbort` (3) and `ExitManagedInstall` (5) are script-facing and collapsed to 1 through any `%w`.
- `archive.go`: `capReader` bounds the whole decompressed tar stream, not just the wanted member — `tr.Next()` decompresses everything it walks past. Reports `errDecompressCap` rather than an opaque truncated-stream error. Test builds a <1 MB archive that expands past 60 MiB.
- `download_test.go`: the checksum-mismatch test now asserts the temp dir is **empty**, pinning verify-before-extract.

---

## Falsifiability

Each production change was mutated and the owning test re-run. 12/12 mutations caught, 0 missed:

| mutation | test that caught it |
|---|---|
| plain path back to no signal handler | `TestUpdate_PlainPathSurvivesSignals` |
| drop `sweepUpdateArtifacts` call | `TestApply_RecoversAbandonedUpdateInOneRun` |
| `reclaimStaleLock` always true | `TestAcquireUpdateLock_RefusesWhileTheOwnerIsAlive` |
| `reclaimStaleLock` always false | `TestApply_RecoversAbandonedUpdateInOneRun` |
| lock record drops the timestamp | `TestAcquireUpdateLock_RecordsItsOwner` |
| drop the `lockMaxAge` check | `TestAcquireUpdateLock_ReclaimsWhenTheLockOutlivesAnyUpdate` |
| drop the `lockWriteGrace` check | `TestAcquireUpdateLock_HonoursAnOwnerlessLockWithinTheWriteGrace` |
| widen sweep glob to `typeburn*.tar.gz` | `TestSweepUpdateArtifacts_TouchesOnlyUpdaterFiles` |
| remove the tag validation | `TestCheck_RejectsATagThatIsNotAVersion` |
| `ExitCode` back to a type assertion | `TestExitCode_SurvivesWrapping` |
| archive cap back to per-member | `TestExtractBinary_BoundsPaddingWalkedPastTheWantedMember` |
| keep the download after a failed verify | `TestDownloadVerified_ChecksumMismatch` |

Signal handling is proven with real OS processes and real signals; `-race` and in-process tests cannot see that class of failure.

## CHANGELOG-worthy (user-visible)

- `typeburn update` now handles Ctrl-C / SIGTERM on non-terminal output (pipes, redirects, CI, narrow terminals): the download is stopped, temp files and the lock are cleaned up, and it reports `update cancelled; nothing was installed` with exit code 3. Previously it died silently and left a lock that blocked every later update.
- An update abandoned by a crash, power loss, or SIGKILL now self-heals: the next run reclaims the stale lock and clears the leftover temp files. No manual file deletion.
- Exit codes 3 (abort) and 5 (managed install) are now preserved through wrapped errors instead of degrading to 1.

## Follow-up not done (outside this phase's file ownership)

`docs/codebase-summary.md:37` still cites `restoreInterruptedUpdate` as Windows "crash recovery". That symbol is gone. The line should read that the Windows path does a rename-aside swap with rollback, and clears the previous `.old` on the next update. Left untouched to respect file ownership — needs whoever owns the docs to apply it before merge, otherwise the doc asserts recovery that does not exist (the same defect B3 was about).

## Unresolved questions

1. `lockMaxAge` = 15 min and `lockWriteGrace` = 5s are my values, derived from the 90s HTTP client timeout. Confirm the trade-off is acceptable: a genuinely wedged updater blocks retries for up to 15 minutes.
2. `sweepUpdateArtifacts` deletes `checksums.txt` from the install directory. `downloadVerified` already unconditionally deletes that same path at the end of every update, so this is not new exposure — but if someone keeps an unrelated `checksums.txt` next to the binary, both old and new code eat it. Worth renaming to `typeburn-checksums.txt`? Out of scope here.
3. `TestUpdateHelperProcess` shows as a SKIP in normal runs (standard subprocess-helper pattern). Confirm that is acceptable noise in CI output.
