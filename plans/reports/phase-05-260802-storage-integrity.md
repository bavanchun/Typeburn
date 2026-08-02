# Phase 5 — Storage Integrity

Status: DONE. Gate green (gofmt / vet / `go test ./... -race -count=1` / GOOS=windows build / make lint / make build / make size-check).

## Measured persisted counts (2 OS processes × 60 appends, 2 ms cadence, one XDG_DATA_HOME)

| Code state | Persisted / 120 | Notes |
|---|---|---|
| Before any change | **96, 87, 89** (3 runs) | Plan measured 114 on its machine; this machine loses more. Also hard failures: `rename ... history.json.tmp: no such file or directory` — writers delete each other's shared temp file. |
| Unique temp names only | **60, 60, 60** | Write errors gone; read-modify-write race now fully exposed, last writer wins (all surviving records from writer 2). Matches the plan's "improve but not fix". |
| Unique temp + advisory lock | **120, 120, 120** — then 8/8 green under `-race` | Final. |

Harness: `internal/storage/concurrent_test.go`. Children are re-executions of the test binary (`-test.run=^TestHistoryAppendChild$`) gated by env; they block on a **one-byte stdin start signal** so all writers contend from the same instant (no parent sleep). Assertion is on the final count and per-writer counts only, never on interleaving. Parent counts survivors *before* reporting child exit status, so the measurement is never hidden by a failing writer.

Why `-race` never covered this: it instruments shared memory inside one process; two `typeburn` processes share none.

## Design decisions

**Lock.** `history.lock` beside `history.json`, held across load→append→write. Locking the data file itself is wrong: `rename` swaps the inode, so a holder of the old inode would not exclude a process that opened the new one. Non-blocking primitive + bounded poll (5 ms) up to `lockTimeout = 2s`, always at least one attempt.

**flock-unavailable / timeout degradation.** Every acquisition failure — timeout, unsupported filesystem (NFS, container overlay), unsupported GOOS — is wrapped in one sentinel `errLockUnavailable`, so there is a single degradation branch: **write anyway, return `Notice{Kind: NoticeUnsynchronised}`**. Never blocks, never fails, never silent. "Never lose a result" outranks "never race".

Platform files: `lock_unix.go` (`syscall.Flock`, LOCK_EX|LOCK_NB; EWOULDBLOCK = busy, anything else = degrade), `lock_windows.go` (`windows.LockFileEx` with LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY — real locking on Windows, not a stub), `lock_unsupported.go` (everything else: reports "cannot lock", degrades). Both kernels release the lock on process death, so a crash cannot wedge the lock. `golang.org/x/sys` moved from indirect to direct in go.mod (allowed family; no version change, no go.sum change).

**fsync vs docstring.** Chose to fsync — `syncDir` after `os.Rename`, and the package docstring now states exactly that. The dir fsync is **best effort**: its error is deliberately not returned, because the data is already in the renamed target at that point and turning a durability hint into a "save failed" notice would be a false alarm (Windows rejects fsync on a directory handle and journals the rename anyway). Docstring says this rather than claiming unconditional crash-safety.

**Quarantine.** `loadForAppend` (write path only; `LoadHistory` stays read-only and non-mutating) renames an existing-but-unparseable file to `history.json.corrupt-<YYYYMMDD-HHMMSS>`, uniquified with `-1`, `-2`… so an earlier quarantine is never overwritten. Nothing is ever deleted. Added case not in the spec: a **zero-byte/whitespace file is not corruption** (crash between create and write) — treated as no history, no scary notice, no clutter file.

If the quarantine rename itself fails, the code continues and lets `atomicWrite` report its own error: a failed rename means the directory is unwritable, so the write would fail regardless — no special case needed.

**Also done.** `sort.Slice` → `sort.SliceStable`; `os.CreateTemp(dir, "<base>-*.tmp")` (also kills the stale-`.tmp` lockout).

## storage → app API added (HANDOFF — orchestrator must wire)

`internal/storage` stays UI-free (no bubbletea/lipgloss). New exported surface:

```go
type NoticeKind int
const (NoticeNone NoticeKind = iota; NoticeQuarantined; NoticeUnsynchronised)

type Notice struct {
    Kind    NoticeKind
    Message string // display-ready single line
    Path    string // quarantine file, set only for NoticeQuarantined
}
func (n Notice) IsZero() bool

func AppendHistoryWithNotice(r Record) ([]Record, Notice, error)
```

`AppendHistory(r) ([]Record, error)` is unchanged and still works — it delegates and drops the notice — so nothing in `internal/app` breaks before wiring. Precedence when both conditions hit: quarantine wins (more actionable).

**Wiring the orchestrator must do after the concurrent `internal/app` phase merges** — `internal/app/model_history.go:50`, inside `handleResultMsg`:

```go
_, notice, err := storage.AppendHistoryWithNotice(rec)
if err != nil {
    m.persistErr = "Couldn't save result to disk"
} else if !notice.IsZero() {
    m.persistErr = notice.Message
}
```

`m.persistErr` already feeds `ui.PersistenceNotice` — no new UI component needed. Nothing else to wire.

Second, smaller handoff: `handleResultMsg` calls `storage.LoadHistory()` before appending, so a corrupt history file makes `IsNewBest` see an empty history and claim a spurious new best. Out of this phase's scope (new_best.go is Phase 3, model_history.go is another phase); flagging it.

Doc handoff: `docs/codebase-summary.md:460` advertises "Race detection … GREEN" as the concurrency assurance. It never covered cross-process writes; someone owning docs should correct that claim.

## Falsifiability — mutations run, each made the named test fail

| Mutation | Test that failed |
|---|---|
| `sort.SliceStable` → `sort.Slice` | `TestAppendHistory_EqualTimestampsKeepInsertionOrder` (`reordered stored records at index 12: was 12, now 18`) |
| quarantine removed (`return nil, Notice{}`) | `TestAppendHistory_UnreadableFileIsQuarantined` |
| `os.CreateTemp` → fixed `path+".tmp"` | `TestAppendHistory_StaleTempFileDoesNotBlockSaves` (`permission denied`) |
| `defer lock.release()` → `lock.release()` (lock dropped before the read-modify-write) | `TestConcurrentAppends_NoRecordsLost` (**60/120**) |
| lock failure returns the error instead of degrading | `TestAppendHistory_LockUnavailableStillSaves` |
| empty-file guard disabled | `TestAppendHistory_EmptyFileIsNotQuarantined` |
| `quarantinePath` never uniquified | `TestQuarantinePath_KeepsEarlierQuarantines` |

First attempt at the stable-sort test was **not** falsifiable: Go's pdqsort insertion-sorts n≤12 and short-circuits already-sorted input, so `sort.Slice` looked stable. Probed offline for a construction where it genuinely reorders — the append must arrive **out of order** (record whose timestamp is not the newest, e.g. clock stepped back), with ≥12 records per equal-timestamp group. Test rebuilt on that, then confirmed failing under the mutation. No retries anywhere; the concurrency test needed none (8/8 under `-race`).

Lock-degradation test forces a **real** contention (a second open file description holds the lock; flock/LockFileEx are per-open-file, so this contends from the same process) and calls the unexported `appendHistory(rec, 50ms)` — no mock, no global mutation. It asserts: record on disk, `err == nil`, `NoticeUnsynchronised`, and elapsed bounded (no hang).

## Files

Modified: `internal/storage/history_store.go` (91→154), `internal/storage/atomic_write.go` (45→66), `go.mod` (x/sys indirect→direct).
Created: `lock.go` (69), `lock_unix.go` (33), `lock_windows.go` (34), `lock_unsupported.go` (23), `history_notice.go` (84), `concurrent_test.go` (140), `history_quarantine_test.go` (130), `history_integrity_test.go` (76), `lock_test.go` (104). All < 200 LOC. `new_best.go` untouched.

## CHANGELOG (user-visible)

- Running two Typeburn instances no longer loses results — history writes are serialised across processes; if the lock cannot be taken the result is still saved and a notice says it was saved unsynchronised.
- A damaged `history.json` is no longer silently replaced: it is kept as `history.json.corrupt-<timestamp>` and the app says where it went.
- A leftover `history.json.tmp` from a crashed run no longer blocks every future save.
- History entries sharing a timestamp keep a stable order.

## Unresolved questions

1. `NoticeUnsynchronised` fires on every save on a filesystem without working locks (some NFS/container setups) — repeated per-result toast. Suppress after the first occurrence per session? Needs a product call; the app-side seam can do it.
2. Quarantine files accumulate with no reaper by design ("do not auto-delete a user's only copy"). Is a History-screen hint or a `typeburn history repair` command wanted later?
3. `SaveSettings` shares `atomicWrite` but takes no lock — concurrent settings writers still last-writer-win. Out of this phase's scope; worth a decision.
