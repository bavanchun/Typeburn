---
phase: 5
title: "Storage Integrity"
status: done
priority: P1
effort: "1d"
dependencies: [1]
---

# Phase 5: Storage Integrity

## Overview

Stop history losing records to concurrent writers and stop one corrupt byte
destroying 200 of them. Separated from the other correctness work because
cross-process locking is a distinct discipline with its own failure modes and its
own test harness.

## Requirements

- Functional
  - Two concurrent instances lose no records.
  - A corrupt file is quarantined, never overwritten.
  - A stale temp file cannot permanently block saves.
- Non-functional
  - A lock failure must never lose a result — degrade, notify, continue.
  - The regression test spawns **real OS processes**; `-race` cannot see this class.

## Architecture

### Concurrent writers lose records

`history_store.go:57-90` is `LoadHistory` → append → sort → `atomicWrite` with no
lock, and `atomic_write.go:15` uses a fixed `path + ".tmp"` shared by every
process. So two writers open, truncate and write the same temp file, then race to
rename it.

Measured, two real OS processes against one `XDG_DATA_HOME`, 60 appends each at
2 ms cadence:

```
total persisted=114 (expected 120); per-instance: map[1:57 2:57]
```

In-process (8 goroutines × 20): **2 of 160** survive.

`atomicWrite` is atomic *per write*; the read-modify-write around it is not.

Fix, two parts:
- `os.CreateTemp(dir, "history-*.json")` for a unique temp name. This alone also
  fixes the stale-`.tmp` lockout (`atomic_write.go:15-20`), where a leftover
  unwritable temp file makes **every** future save fail.
- An advisory lock held across load→append→write.

**Why `-race` never caught this and never will:** it instruments shared memory
within one process. Two `typeburn` processes share no memory. Compounding it,
`history_store_test.go` has 20 test functions and **zero** `go func`,
`sync.WaitGroup`, or `t.Parallel` — every test is single-process on a fresh
`t.TempDir()`. `grep -rn "Flock\|LOCK_EX" internal/` returns nothing; there is no
lock to test. Yet `docs/codebase-summary.md:460` advertises "Race detection …
GREEN" as the concurrency assurance. It assures nothing about this failure mode.

### The lock must never cost a result

A lock that can hang is worse than the race it prevents. Requirements:
- Bounded acquisition with a timeout.
- On timeout or on a filesystem without working `flock` (NFS, some containers),
  **degrade to today's behaviour and surface a notice** — never block a save
  forever, never drop the result silently.
- The notice reuses the seam Phase 3 makes load-bearing.

### One corrupt byte destroys everything

`history_store.go:43-46` returns `nil` on unmarshal failure; `:57-59` immediately
appends to that `nil` and overwrites the file.

```
200 valid records, truncate 3 bytes:
  -> after one append: 1 record on disk; backups found: []
```

Silent, irreversible, no error surfaced. Any transient corruption — including one
caused by the race above — becomes permanent total loss.

Fix: rename to `history.json.corrupt-<ts>` before the next write and surface a
notice through the existing seam. The user's data becomes recoverable by hand
instead of gone.

### Also here

- `sort.Slice` → `sort.SliceStable` (`history_store.go:62-64`): records sharing a
  timestamp currently reorder on every append.
- `atomic_write.go:39` never fsyncs the parent directory after `os.Rename`, so the
  rename is not durable across power loss — while the package docstring claims
  crash-safety. Either fsync the dir or correct the docstring; do not leave the
  claim standing.

## Related Code Files

- Modify: `internal/storage/history_store.go`, `internal/storage/atomic_write.go`
- Create: `internal/storage/lock.go` (advisory lock with timeout + degradation)
- Create: `internal/storage/concurrent_test.go` (subprocess harness)
- Note: `internal/storage/new_best.go` belongs to **Phase 3**, not this phase

## Implementation Steps

1. Build the subprocess harness first — a test binary that appends N records on
   demand, driven by the parent. Confirm it reproduces `114/120` against current
   code before changing anything.
2. `os.CreateTemp` unique temp names. Re-run: this alone should improve but not
   fix the count.
3. Advisory lock with bounded acquisition. Re-run: expect 120/120.
4. Degradation path: simulate lock failure, assert the record is still written
   and a notice is surfaced.
5. Corrupt-file quarantine + notice; assert the original bytes are recoverable
   from the `.corrupt-<ts>` file.
6. Stale-temp test: leave an unwritable `history.json.tmp`, assert saves still work.
7. `sort.SliceStable`; fsync the parent dir or correct the docstring.

## Success Criteria

- [ ] Subprocess harness reproduces the loss against current code (`114/120`)
- [ ] Two subprocesses × 60 appends → **120 records persisted**
- [ ] Lock failure → record still written, notice surfaced, no hang
- [ ] Corrupt file quarantined to `history.json.corrupt-<ts>`; original bytes recoverable
- [ ] A stale unwritable `history.json.tmp` no longer blocks saves
- [ ] Records with equal timestamps keep a stable order across appends
- [ ] `atomic_write.go`'s durability claim matches its behaviour
- [ ] `go test ./... -race -count=1` green, run with `-count=1` to avoid cached results masking flakes

## Risk Assessment

**Risk:** the advisory lock deadlocks, or `flock` is unavailable, and users can no
longer save results at all — strictly worse than the bug being fixed.
**Mitigation:** bounded acquisition + explicit degradation path, tested in step 4.
"Never lose a result" outranks "never race".

**Risk:** the subprocess test is flaky in CI (process startup timing, `-race`
overhead on shared runners).
**Mitigation:** drive the children by explicit signal rather than sleeps; assert
on the final count, not on interleaving. Run `-count=1`. If it proves flaky,
raise the append count rather than adding retries — a flaky concurrency test that
is "fixed" by retrying is a test that no longer tests anything.

**Risk:** quarantine files accumulate.
**Mitigation:** timestamped names, and the notice tells the user where the file
went. Do not auto-delete a user's only copy of their data.

## Rollback

Revert the commit. The on-disk format is unchanged — only the temp-file naming
and the lock file are new, and both are transient. Quarantine files created
before a rollback remain readable JSON.
