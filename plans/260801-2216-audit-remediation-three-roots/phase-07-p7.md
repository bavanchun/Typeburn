---
phase: 7
title: "CLI And Update Hardening"
status: pending
priority: P1
effort: "1d"
dependencies: []
---

# Phase 7: CLI And Update Hardening

## Overview

Make `typeburn update` survive Ctrl-C. This phase exists because red-team found
the plan had silently dropped an entire audit scope — and the audit's own verdict
was that these are *the findings that matter most to a real user*:

> Nobody is going to MITM a typing tester. A lot of people are going to press
> Ctrl-C during an update, and today that bricks the update feature until they
> manually delete three files they were told about one of.

## Requirements

- Functional
  - Ctrl-C during a plain-path update leaves no lock and no partial artifacts.
  - A stale lock from an abrupt death self-heals.
  - Recovery never requires the user to delete files by hand.
- Non-functional
  - No change to the verify → extract → replace ordering, which the audit
    confirmed correct and which must stay correct.

## Architecture

### B1 — the plain path has no signal handling at all

`cmd/typeburn/main.go:14` is
`fang.Execute(context.Background(), root, fang.WithoutVersion())` — **no signal
options**, so `fang.go:167` never installs `signal.NotifyContext` and
`cmd.Context()` is a bare `Background()`. `runApply`'s plain branch
(`cmd_update_run.go:51-52`) then calls `Apply` **synchronously**. SIGINT/SIGTERM
hit Go's default disposition and kill the process mid-`Apply` with zero defers run.

Verified directly: `grep -rn "signal\.Notify" --include='*.go' .` returns exactly
one hit — `internal/cli/notui/lifecycle.go:27`. The repo already knows the
pattern; the update command is the inconsistent one.

The animated path *is* hardened (bubbletea installs handlers → `InterruptMsg` →
`!settled` → `stopApply` drains). The plain path covers pipes, redirects, CI,
non-tty, and any terminal narrower than 56 columns.

Subprocess repro against a hanging httptest server:
```
interrupt:  lock file leaked, every future update refuses to start
terminated: lock file leaked, every future update refuses to start
```

Fix: wrap the plain branch in `signal.NotifyContext(cmd.Context(), os.Interrupt,
syscall.SIGTERM)` and route the result through the existing
`stopApply`/`reportApplyResult` machinery, so both paths share one contract.

### B2 — recovery needs three manual deletions, one hint

Four artifacts each independently and permanently block every future update via
`O_EXCL`, and none is cleaned up by a later run:

| stale file | error | actionable? |
|---|---|---|
| `.typeburn-update.lock` | `another update is already in progress (remove … if stale)` | yes |
| `checksums.txt` | `create download file: … file exists` | path only |
| `typeburn_<v>_<os>_<arch>.tar.gz` | `create download file: … file exists` | path only |
| `typeburn.new` | `create extracted binary: … file exists` | path only |

Measured, removing only the file each error names:
```
attempt 1: another update is already in progress (remove …/.typeburn-update.lock if stale)
attempt 2: create download file: open …/checksums.txt: file exists
attempt 3: create download file: open …/typeburn_9.9.9_linux_amd64.tar.gz: file exists
attempt 4: SUCCESS
```

After deleting the lock as instructed the user hits a raw `file exists` that reads
like a bug. A non-technical user concludes self-update is broken.

Fix: write PID + start time into the lock file. On `IsExist`, if the PID is dead
or the lock is older than a threshold, reclaim it **and sweep the three sibling
artifacts** before retrying. This closes B2 and the deferred SIGKILL/power-loss
case in one change.

### B3 — dead recovery code with a lying comment

`replace_windows.go:39` / `replace_unix.go:38`: `restoreInterruptedUpdate` has
**no caller anywhere** (`grep -rn restoreInterruptedUpdate --include='*.go' .`
returns only the definitions). Yet `replace_windows.go:13` says the leftover
"may stay locked while the process runs — `restoreInterruptedUpdate` cleans it up
on the next launch." It does not run on the next launch; it does not run ever.
`go vet` cannot catch an unused unexported function.

Either call it early in `main()` (and extend it to sweep a leftover `.old` when
the target is present), or delete it and correct the comment. **Do not leave a
comment asserting recovery that does not happen** — a future maintainer will
trust it.

### B4 — the tag-to-path invariant is accidental

`download.go:53-56` interpolates the raw release tag into a filename with no
sanitisation, and `filepath.Join(destDir, name)` at `:127,139` uses it.

```
tag="v../../../../../../../../tmp/typeburn-probe-evil"
  -> join escapes destDir
```

**Not reachable today** — every tag surviving `runUpdate`'s
`UpgradeAvailable` gate is `v?D.D.D` or `v?D.D.D-N-g<hex>`, so no separators are
possible. But that safety is a *side effect* of `parseSemver`'s `strconv.Atoi`
and `IsPrerelease`'s heuristic. `cache.go:23` already has `validSemverRe` and
calls it an "injection guard" — applied only on the cache-read path, never to the
live `FetchLatest` result that `update` actually uses.

Fix: apply `validSemverRe` to `rel.TagName` in `check.go` right after
`FetchLatest`. One line, and it makes the invariant explicit instead of
incidental so a future prerelease-support change cannot silently unlock it.

**Threat-model honesty:** an attacker who can forge the API response can already
serve a malicious binary with a matching checksum, so traversal buys them
nothing. This is a maintainability landmine, not a live breach — fix it as such,
do not inflate it.

### Also here (same files, cheap)

- `exitcodes.go:52` uses a bare type assertion, not `errors.As`, so any `%w` wrap
  collapses the documented exit codes 3 and 5 to 1. Latent — nothing wraps today
  — but `ExitManagedInstall` and `ExitAbort` are script-facing contracts.
- `archive.go:83-85` skips unwanted members via `tr.Next()`, which decompresses
  them; `decompressCap` bounds only the wanted member. A 1 MB archive expanded
  1 GiB in testing. DoS-only and post-integrity (the attacker already owns the
  release), so: wrap the gzip reader in one `io.LimitReader` across the whole
  stream.

## Related Code Files

- Modify: `cmd/typeburn/main.go`, `internal/cli/cmd_update_run.go`, `internal/cli/exitcodes.go`
- Modify: `internal/update/lock.go`, `download.go`, `check.go`, `archive.go`
- Modify or delete: `internal/update/replace_unix.go`, `replace_windows.go`
- Modify: `internal/cli/cmd_update_run_test.go`, `internal/update/*_test.go`

## Implementation Steps

1. Reproduce B1 with a subprocess + hanging httptest server; confirm the lock leaks.
2. `signal.NotifyContext` on the plain branch, routed through `stopApply`.
   Re-run: no lock, and the outcome is reported honestly (the existing
   `errStoppedTooLate` / `errDrainTimeout` ordering in `reportApplyResult` is
   load-bearing — do not reorder it).
3. PID+timestamp lock with reclaim + sibling sweep. Test: seed all four stale
   artifacts, assert one run recovers with no manual deletion.
4. Decide B3 (wire up or delete); make the comment match reality either way.
5. `validSemverRe` on `rel.TagName` in `check.go`.
6. `errors.As` in `ExitCode`; whole-stream `LimitReader` on the gzip reader.

## Success Criteria

- [ ] SIGINT and SIGTERM during a plain-path update leave no lock and no partial files
- [ ] A seeded stale lock + three sibling artifacts recover in **one** run, no manual deletion
- [ ] A live lock held by a running process is still respected (reclaim must not be over-eager)
- [ ] `restoreInterruptedUpdate` is called or gone; its comment matches behaviour
- [ ] `rel.TagName` validated after `FetchLatest`; a tag with a separator is rejected
- [ ] Wrapped `ExitError` still yields its documented exit code
- [ ] Padded-archive decompression bounded
- [ ] verify → extract → replace ordering unchanged; checksum-mismatch still leaves the temp dir empty
- [ ] `go test ./internal/update/... ./internal/cli/... -race -count=1` green

## Risk Assessment

**Risk:** the stale-lock reclaim is over-eager and two genuinely concurrent
updaters both proceed, racing to replace the binary.
**Mitigation:** reclaim only when the PID is dead *or* the lock exceeds a
generous age; test with a live holder to prove it is respected. Prefer refusing a
live update over corrupting a binary.

**Risk:** signal handling changes what the user sees on Ctrl-C.
**Mitigation:** it should — today they see nothing and get a bricked updater. The
existing `reportApplyResult` contract already has honest copy for each outcome,
including "stopped too late — already installed"; reuse it, do not invent new copy.

**Risk:** deleting `restoreInterruptedUpdate` removes real Windows recovery.
**Mitigation:** it has no caller, so it removes nothing that runs. Verify with
`grep` before deleting, and note the leftover `.old` already self-heals on the
next successful update (`replace_windows.go:20`).

## Rollback

Fully self-contained — no shared files with any other phase, no persisted format
change. Revert the commit. The lock-file format changes, but a lock written by
the new code and read by the old is simply treated as "in progress", which is the
safe direction.
