---
phase: 4
title: "update-cache"
status: pending
priority: P2
effort: 30m
dependencies: []
---

# Phase 4: update-cache (LOW-2)

## Overview

`update.Check` (`internal/update/check.go:14-53`) caches the result of a successful upgrade check via `cacheSave` (`check.go:51`) — so the 24h TTL (`cache.go:17`) suppresses repeat network hits. **Except** the prerelease/draft branch (`check.go:32-40`): it builds a synthetic "no upgrade" `Result` and returns it **without** caching.

Consequence: while the latest GitHub release is a prerelease/draft, every TUI launch with `update_check=on` re-fetches `releases/latest`, paying the full HTTP timeout budget (`client.go:31`: 1.5s total, 800ms dial/TLS/header). The cache exists precisely to bound this; the prerelease path silently bypasses it.

**Fix:** call `cacheSave` on the synthetic prerelease result before returning, mirroring the happy path. The 24h TTL then applies, capping it to one hit per day during a prerelease window.

## Requirements

### Functional
- Prerelease/draft branch persists its synthetic `Result` via `cacheSave` before returning (best-effort, error ignored — consistent with `check.go:51`).
- Cached prerelease result must pass `cacheLoad`'s validation (`cache.go:43-76`): `SchemaVersion == cacheSchemaVersion`, `Latest` matches `validSemverRe` or is empty, `CheckedAt` recent.
  - The synthetic result sets `Latest: currentVer` (`check.go:36`). `currentVer` for a real install is a semver tag (e.g. `v2.0.0`) → passes `validSemverRe`. `ReleaseURL` is empty → passes the prefix check. `CheckedAt: time.Now().UTC()` (already set at line 38) → fresh.
- `force=true` still bypasses the cache on read (`check.go:20-24`), so `--check-update` always re-fetches; only the *cached* path benefits. After a forced prerelease check, the cache is now populated for subsequent non-forced TUI launches.

### Non-functional
- Network calls during a prerelease window drop from once-per-launch to once-per-24h.
- No new deps. No signature change.

## Architecture

### Current (broken)
```
FetchLatest → rel
if rel prerelease/draft:
    return synthetic Result          // NOT cached → re-fetch every launch
upgrade := Compare(...)
r := Result{...}
cacheSave(r)                          // happy path caches
return r
```

### Target
```
if rel prerelease/draft:
    r := synthetic Result{CheckedAt: now}
    cacheSave(r)                      // best-effort; TTL now suppresses repeats
    return r, nil
```

## Related Code Files

**Modify**
- `internal/update/check.go` — prerelease/draft branch (lines 32-40): assign to a local, `cacheSave`, return.

**Modify (tests)**
- `internal/update/check_test.go` — extend existing `TestCheck_PrereleaseIgnored` (test:78) OR add `TestCheck_PrereleaseCached` asserting the cache file is written.

**Delete** — none.

## Implementation Steps

1. Rewrite the prerelease branch (`check.go:32-40`):
   ```go
   // Treat draft/prerelease as "no stable upgrade available".
   if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
       r := &Result{
           SchemaVersion:    cacheSchemaVersion,
           Current:          currentVer,
           Latest:           currentVer,
           UpgradeAvailable: false,
           CheckedAt:        time.Now().UTC(),
       }
       _ = cacheSave(r) // best-effort; TTL suppresses repeat network hits during prerelease windows
       return r, nil
   }
   ```

2. Add `TestCheck_PrereleaseCached` to `check_test.go` (use the existing `withTempCache(t)` helper from cache_test.go:13 and `stubServer`/`fetchURL` override pattern verified at test:78-94):
   ```go
   func TestCheck_PrereleaseCached(t *testing.T) {
       defer withTempCache(t)()
       srv := stubServer(t, Release{TagName: "v2.1.0-rc.1", Prerelease: true})
       origURL := fetchURL
       fetchURL = srv.URL
       defer func() { fetchURL = origURL }()

       // force=true → fetch + (now) cache
       r1, err := Check(context.Background(), "v2.0.0", true)
       if err != nil {
           t.Fatalf("unexpected error: %v", err)
       }
       if r1.UpgradeAvailable {
           t.Error("prerelease must not be an upgrade")
       }

       // Cache must now be fresh: a non-forced call returns the cached result
       // WITHOUT hitting the server. Close the server to prove no network use.
       srv.Close()
       r2, err := Check(context.Background(), "v2.0.0", false)
       if err != nil {
           t.Fatalf("cached call should not error after prerelease cache write: %v", err)
       }
       if r2 == nil || r2.UpgradeAvailable {
           t.Errorf("expected cached no-upgrade result, got %#v", r2)
       }
   }
   ```
   This asserts the cache write by proving the second (non-forced) call succeeds with the server down — only possible if the prerelease result was cached. (Defensive: if `withTempCache` isolation matters, ensure no real `StateDir` is touched — `withTempCache` sets `cacheFilePath` to a temp path, verified at cache_test.go:13-15.)

   `srv.Close()` is called explicitly after `r1` is verified (no `defer` — avoids double-close ambiguity). The server is down before `Check` is called a second time, proving the second call returns from cache, not from the network.

3. `gofmt -w`, `go vet ./...`, `go test ./internal/update/ -race -count=1`, then `make test-race`.

## Success Criteria

- [ ] Prerelease/draft branch calls `cacheSave` before returning.
- [ ] `TestCheck_PrereleaseCached` proves a non-forced follow-up call succeeds with the server closed (cache hit).
- [ ] Existing `TestCheck_PrereleaseIgnored` still passes.
- [ ] Cached synthetic result passes `cacheLoad` validation (semver `Latest`, empty `ReleaseURL`).
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Synthetic `Latest=currentVer` fails `validSemverRe` for non-semver builds | Low | Low | `Check` early-returns `(nil,nil)` for dev/pseudo versions (`check.go:15-18`) before reaching this branch, so `currentVer` here is always a real semver tag. |
| Caching a prerelease masks a later stable release for up to 24h | Low | Low | Matches existing happy-path TTL semantics (24h staleness is the accepted design); `force=true`/`--check-update` bypasses. Acceptable per existing cache design. |
| `withTempCache` not isolating real cache | Low | Medium | Helper sets `cacheFilePath` to `t.TempDir()` (verified cache_test.go:13-15); always `defer withTempCache(t)()`. |

### Rollback
`git revert`; prerelease path returns to no-cache. Isolated to `check.go`. (Phase 6 also touches `cache.go`/`client.go` globals but not `check.go` — no overlap.)
