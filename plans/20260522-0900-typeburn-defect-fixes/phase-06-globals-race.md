---
phase: 6
title: "globals-race"
status: pending
priority: P3
effort: 1h
dependencies: [2, 4]
---

# Phase 6: globals-race (LOW-4)

## Overview

Three mutable package-level vars exist solely as test seams and are read on the production path while being overwritten by tests with no synchronization:

- `internal/update/cache.go:27` — `var cacheFilePath = ""` (read in `getCachePath`, cache.go:30).
- `internal/update/client.go:16` — `var fetchURL = "https://api.github.com/..."` (read in `FetchLatest`, client.go:45).
- `internal/cli/cmd_version.go:14` — `var checkFn = update.Check` (read in `runVersion`, cmd_version.go:46).

Today this is safe because no test uses `t.Parallel()`. But it's a latent race: the moment any test in these packages adds parallelism (or runs concurrently with another that overwrites the var), `-race` will flag a data race and CI goes red. This phase wraps each var behind a `sync.Mutex` + typed accessor so reads/writes are synchronized regardless of future parallelism.

This is the highest-effort phase because it has the widest **test** blast radius: every existing override site must migrate to the setter/getter (verified list below).

## Requirements

### Functional
- Production reads go through `getCacheFilePath()`, `getFetchURL()`, `getCheckFn()`.
- Test overrides go through `setCacheFilePath(...)`, `setFetchURL(...)`, `setCheckFn(...)`.
- Behavior identical to current (default values unchanged).
- All existing tests pass; no behavior change visible to users.

### Non-functional
- `go test -race` clean now AND if a future test adds `t.Parallel()`.
- `sync` is stdlib — allowed. No new third-party dep.
- Each accessor pair is tiny; keep files < 200 LOC (cache.go is 131, client.go 71, cmd_version.go 147 — all fine after small additions).

## Architecture

### Pattern (per var, scoped mutex in each file/package)
```go
var (
    cacheFileMu   sync.Mutex
    cacheFilePath = ""
)
func getCacheFilePath() string { cacheFileMu.Lock(); defer cacheFileMu.Unlock(); return cacheFilePath }
func setCacheFilePath(p string) { cacheFileMu.Lock(); defer cacheFileMu.Unlock(); cacheFilePath = p }
```

Use a **distinct mutex name per var** (`cacheFileMu`, `fetchURLMu`, `checkFnMu`) to avoid coupling unrelated vars under one lock and to keep the diff local to each file. `checkFn` is a func value:
```go
var (
    checkFnMu sync.Mutex
    checkFn   = update.Check
)
func getCheckFn() func(context.Context, string, bool) (*update.Result, error) {
    checkFnMu.Lock(); defer checkFnMu.Unlock(); return checkFn
}
func setCheckFn(fn func(context.Context, string, bool) (*update.Result, error)) {
    checkFnMu.Lock(); defer checkFnMu.Unlock(); checkFn = fn
}
```
The `checkFn` signature matches `update.Check` (`func(ctx, ver string, force bool) (*update.Result, error)`) and the existing `stubCheck` return type (cmd_version_test.go:15).

### Data flow (unchanged values, synchronized access)
```
runVersion → getCheckFn()(ctx, ver, true)        // was: checkFn(...)
getCachePath → getCacheFilePath()                 // was: cacheFilePath
FetchLatest → http.NewRequestWithContext(..., getFetchURL(), ...)  // was: fetchURL
```

## Related Code Files

**Modify (production)**
- `internal/update/cache.go` — add `sync` import, `cacheFileMu`, accessors; `getCachePath` (line 30) reads `getCacheFilePath()`.
- `internal/update/client.go` — add `sync` import, `fetchURLMu`, accessors; `FetchLatest` (line 45) reads `getFetchURL()`.
- `internal/cli/cmd_version.go` — add `sync` + `context` imports (context already used? it imports via cobra cmd.Context() but may not import `context` directly — verify), `checkFnMu`, accessors; `runVersion` (line 46) reads `getCheckFn()`.

**Modify (tests) — full migration, enumerated**
- `internal/update/cache_test.go`:
  - `withTempCache` helper (lines 13-15): `orig := getCacheFilePath()`; `setCacheFilePath(filepath.Join(dir,...))`; return `func(){ setCacheFilePath(orig) }`.
  - `TestCacheSave_CreatesParentDir` (lines 151-154): two bare writes — line 151 reads `cacheFilePath` → `getCacheFilePath()`; line 153 assigns `cacheFilePath = filepath.Join(...)` → `setCacheFilePath(filepath.Join(...))`.
  - Direct read at line 165 (`os.Stat(cacheFilePath)`) → `os.Stat(getCacheFilePath())`.
- `internal/update/client_test.go` (lines 113-115): `orig := getFetchURL(); setFetchURL(url); defer setFetchURL(orig)`.
- `internal/update/check_test.go` (lines 34, 60, 80, 134 — 4 sites, plus the Phase 4 `TestCheck_PrereleaseCached` site): migrate each `origURL := fetchURL; fetchURL = srv.URL; defer func(){ fetchURL = origURL }()` → `origURL := getFetchURL(); setFetchURL(srv.URL); defer setFetchURL(origURL)`.
- `internal/cli/cmd_version_test.go` (5 sites: 52/60, 76/82, 94/96, 108/110, 123/132 — plus the Phase 2 `TestVersionCheckUpdate_JSONError` site): migrate each `orig := checkFn; checkFn = ...; defer func(){ checkFn = orig }()` → `orig := getCheckFn(); setCheckFn(...); defer setCheckFn(orig)`.

**Delete** — none.

## Implementation Steps

1. **`internal/cli/cmd_version.go` — add missing imports first (required, not optional):** `cmd_version.go` does not currently import `"context"` directly (it uses `cmd.Context()` which is unqualified). The `getCheckFn` accessor signature requires `context.Context` as a type, so `"context"` MUST be added before the accessor compiles. Add both `"context"` and `"sync"` to the import block at the top of the file. This is a certain compile failure if omitted — do it first, before any var/accessor additions.

2. **`internal/update/cache.go`:** add `"sync"` import; replace `var cacheFilePath = ""` (line 27) with the `cacheFileMu` + var block + `getCacheFilePath`/`setCacheFilePath`; change `getCachePath` body (line 30) `if cacheFilePath != ""` → `if p := getCacheFilePath(); p != ""` (and return `p`).

3. **`internal/update/client.go`:** add `"sync"`; replace `var fetchURL = "..."` (line 16) with `fetchURLMu` + var + accessors; in `FetchLatest` (line 45) use `getFetchURL()`.

4. **`internal/cli/cmd_version.go` — add accessors:** Replace `var checkFn = update.Check` (line 14) with `checkFnMu` + var + accessors; in `runVersion` (line 46) `result, err := getCheckFn()(cmd.Context(), info.Version, true)`.

5. **Migrate all test override sites** per the enumerated list above. After migration, no test references the bare vars `cacheFilePath`, `fetchURL`, `checkFn` directly (grep to confirm zero remaining bare references outside the accessor/var definitions).

6. **Verification grep:** after edits, run
   ```
   grep -rn "cacheFilePath\|fetchURL\|checkFn" internal/ --include='*.go'
   ```
   Expect matches ONLY in: the var declarations, the accessor bodies, and accessor *names*. No bare assignment/read elsewhere.

7. **Sequencing:** Phase 2 adds `TestVersionCheckUpdate_JSONError` with a bare `checkFn` seam; Phase 4 adds `TestCheck_PrereleaseCached` with a bare `fetchURL` seam. **Phase 6 must run after both** (`dependencies: [2, 4]`) to migrate all override sites in one pass. Migrate the Phase 4 `TestCheck_PrereleaseCached` fetchURL site in addition to the sites enumerated in step 5 above.

8. `gofmt -w`, `go vet ./...`, then the race gate: `go test ./internal/update/ ./internal/cli/ -race -count=1`, then full `make test-race`.

9. **(Optional) Prove the race fix:** temporarily add `t.Parallel()` to two `internal/update` tests that both call `withTempCache`/`setFetchURL`, run `-race`, confirm GREEN, then remove the `t.Parallel()` (do not commit speculative parallelism — YAGNI; the accessors are the durable fix). This is a one-off local sanity check, not a committed test.

## Success Criteria

- [ ] `cacheFilePath`, `fetchURL`, `checkFn` each guarded by a dedicated `sync.Mutex` with get/set accessors.
- [ ] All production reads use getters; all test overrides use setters.
- [ ] Verification grep shows no bare var access outside declarations/accessors.
- [ ] Default values unchanged; all existing tests pass unmodified in behavior.
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.
- [ ] `cmd_version.go` imports `"context"` and `"sync"` (added in step 1 before any other edit).
- [ ] No new third-party dep (`sync` is stdlib).
- [ ] cache.go / client.go / cmd_version.go each < 200 LOC.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Missed a test override site → compile error or stale var | Medium | Low (caught at compile) | Enumerated full list (5 checkFn + 5 fetchURL incl. Phase 4 site + 3 cacheFilePath sites); step-6 grep confirms zero bare refs. |
| Concurrent edit conflict with Phase 2/4 on shared files | Low | Medium | `dependencies: [2, 4]`; run sequentially. P2 touches `renderVersionCheckJSON`; P4 touches `check.go`; P6 touches `checkFn`/`fetchURL`/`cacheFilePath` regions — all disjoint. |
| Accessor adds lock contention on hot path | None | — | Update check is once-per-launch; `runVersion` once per invocation. No hot path. |
| Deadlock from nested locks | None | — | Each accessor takes one lock and returns; no nesting, no cross-lock calls. |

### Rollback
`git revert` the phase commit. Reverts to bare vars + bare test overrides. Self-contained; no production behavior change to undo. Must revert as one unit (production accessors + all test migrations are coupled).

### File-ownership note
Shares `internal/cli/cmd_version.go` with **Phase 2** (disjoint regions) and `internal/update/check_test.go` with **Phase 4** (Phase 4's new test uses bare `fetchURL` — this phase migrates it). **Runs after Phase 2 and Phase 4.** No other file overlaps.
