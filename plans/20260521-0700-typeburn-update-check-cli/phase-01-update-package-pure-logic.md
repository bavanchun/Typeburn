---
phase: 1
title: "Update Package Pure-Logic"
status: pending
priority: P1
effort: "1d"
dependencies: []
---

# Phase 1: Update Package Pure-Logic (`internal/update/`)

## Overview

Build a stdlib-only, UI-free package that handles: GitHub Releases API fetch,
24h JSON cache (atomic), stdlib semver comparison, and an orchestrator that
ties them together with cache + dev-skip + pre-release filter. Zero
`bubbletea`/`lipgloss` imports. Mirrors the layering rule applied to
`internal/typing`, `internal/metrics`, etc.

## Requirements

**Functional**
- `Check(ctx context.Context, currentVer string, force bool) (*Result, error)` —
  honors 24h cache unless `force=true`; skips entirely when `currentVer` is
  `""`/`"dev"`/pseudo-version (`(devel)` or `v0.0.0-…` from `debug.ReadBuildInfo`).
- `FetchLatest(ctx context.Context, currentVer string) (Release, error)` —
  GET `https://api.github.com/repos/bavanchun/Typeburn/releases/latest`,
  `User-Agent: Typeburn/<currentVer> (+https://github.com/bavanchun/Typeburn)`,
  `Accept: application/vnd.github+json`, body capped at 64KB via `io.LimitReader`.
- `Compare(a, b string) int` — pure, returns `-1/0/+1`; strips leading `v`;
  treats `vX.Y.Z-N-gSHA` (git-describe) as `vX.Y.Z`; returns 0 (treat as equal)
  for malformed input.
- `IsPrerelease(tag string) bool` — `true` if tag contains `-rc/-beta/-alpha/-pre`
  (case-insensitive) or starts with `v0.0.0-`.
- Cache load/save at `$XDG_STATE_HOME/typeburn/update-check.json`
  (fallback `$HOME/.local/state/typeburn/`), atomic write, 24h TTL.

**Non-functional**
- Stdlib only: `net/http`, `encoding/json`, `context`, `time`, `io`,
  `strings`, `strconv`, `os`, `path/filepath`, `errors`. **No new go.mod entries.**
- Each source file < 200 LOC.
- Zero `bubbletea`/`lipgloss` imports (CI-lintable via `go list -deps`).
- HTTP: 1.5s total timeout (layered defense — `http.Client.Timeout` AND
  `context.WithTimeout`, per researcher-01 §4).
- Silent-degrade on any error: network, 403, 429, 404, parse, malformed JSON.

## Key Insights (from research)

- GitHub requires `User-Agent`; 403 without it (researcher-01 §3).
- Default `http.DefaultTransport` does **not** honor `HTTPS_PROXY` —
  custom `Transport{Proxy: http.ProxyFromEnvironment}` needed (researcher-01 §6).
- `internal/storage/atomic_write.go` exposes a private `atomicWrite(path, []byte) error`
  — we cannot import it directly. Two options: (a) duplicate the ~30-LOC helper
  in `internal/update/cache.go` (KISS), or (b) export it from `internal/storage`
  (DRY but touches a stable package). **Decision: option (b) — promote to
  `storage.AtomicWrite` (one-line rename + visibility change)** so future
  packages reuse it. Justification: KISS still holds (no duplication), and
  the API surface change is trivial + tested by existing users.
- `internal/config/xdg-paths.go` has `ConfigDir()` and `DataDir()` but **no
  `StateDir()` helper** (researcher-02 Part A). Add `StateDir()` in the same
  file using the same `resolveDir` pattern: `$XDG_STATE_HOME` →
  `$HOME/.local/state`.
- `internal/version.Resolve()` returns three strings; `Version` is what we
  compare. Pseudo-version detection: starts with `v0.0.0-` OR equals
  `(devel)` OR equals `dev` OR is empty.

## Architecture

```
internal/update/
├── client.go        # FetchLatest(ctx, ver) (Release, error) — HTTP + UA + LimitReader
├── compare.go       # Compare(a, b string) int — stdlib semver
├── prerelease.go    # IsPrerelease(tag string) bool — tiny, splits out for testing
├── cache.go         # Load() / Save(Result) — uses storage.AtomicWrite + StateDir()
├── check.go         # Check(ctx, ver, force) — orchestrator: dev-skip, cache, fetch, persist
├── result.go        # Public types: Release, Result (with json tags for output)
├── client_test.go   # httptest.Server — 6+ cases (200 happy, 403, 404, 429, slow, malformed)
├── compare_test.go  # table-driven — 12+ cases (equal, M/m/p bumps, prerelease, malformed, git-describe)
├── prerelease_test.go
├── cache_test.go    # t.TempDir() — write/read/expiry/corrupt-file
└── check_test.go    # integration: stubbed client + tempdir cache
```

Public types in `result.go`:
```go
type Release struct {
    TagName     string    `json:"tag_name"`
    HTMLURL     string    `json:"html_url"`
    Name        string    `json:"name"`
    Draft       bool      `json:"draft"`
    Prerelease  bool      `json:"prerelease"`
    PublishedAt time.Time `json:"published_at"`
}

type Result struct {
    Current          string    `json:"current"`
    Latest           string    `json:"latest"`
    UpgradeAvailable bool      `json:"upgrade_available"`
    ReleaseURL       string    `json:"release_url"`
    CheckedAt        time.Time `json:"checked_at"`
}
```

## Related Code Files

- **Create:** `internal/update/client.go`, `compare.go`, `prerelease.go`,
  `cache.go`, `check.go`, `result.go` + 5 `*_test.go` files.
- **Modify:** `internal/storage/atomic_write.go` — rename `atomicWrite` →
  `AtomicWrite` (export); update internal callers in `settings_store.go` and
  `history_store.go`. **Verify with `go build ./...` after rename.**
- **Modify:** `internal/config/xdg-paths.go` — add `StateDir() (string, error)`
  beside existing `ConfigDir`/`DataDir`.

## Implementation Steps

1. **Hardened atomic write helper** (private to `internal/update/cache.go`).
   <!-- SUPERSEDED by red-team-finding-7 — do NOT promote storage.atomicWrite. Duplicate ~30 LOC into update/cache.go and harden with O_EXCL|O_NOFOLLOW + unique tmp suffix. See Red Team Review Updates. -->
   Implement a private `atomicWrite(path string, data []byte) error` in
   `internal/update/cache.go` that opens with `O_WRONLY|O_CREATE|O_EXCL|O_NOFOLLOW`,
   uses `fmt.Sprintf("%s.%d.tmp", path, os.Getpid())` for the temp suffix, then
   fsync + rename. Leave `internal/storage/atomic_write.go` untouched.
2. **Add `StateDir()`** to `internal/config/xdg-paths.go`. Test parity with
   existing `ConfigDir`/`DataDir` test pattern.
3. **`internal/update/result.go`** — types only, no logic.
4. **`internal/update/compare.go`** + `compare_test.go` (table-driven, TDD-friendly).
   Cases: `("v1.0.0","v1.0.0")=0`, `("v2.0.0","v2.1.0")=-1`, `("v2.10.0","v2.9.0")=+1`,
   `("v2.0.0-1-g123","v2.0.0")=0`, `("","v2.0.0")=0` (malformed → equal),
   `("v0.0.0-rc.test","v2.0.0")=-1` if reached (but `IsPrerelease` should filter
   earlier).
5. **`internal/update/prerelease.go`** + test. Cases: `v2.0.0`=false,
   `v2.0.0-rc.1`=true, `v0.0.0-rc.test`=true, `V2.0.0-Alpha`=true (case-insensitive),
   `v2.0.0-7-gabc123`=false (git-describe is NOT a prerelease).
6. **`internal/update/client.go`** — `FetchLatest`. Layered timeout:
   ```go
   client := &http.Client{
       Timeout:   1500 * time.Millisecond,
       Transport: &http.Transport{Proxy: http.ProxyFromEnvironment},
   }
   ctx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
   defer cancel()
   req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
   req.Header.Set("User-Agent", "Typeburn/"+ver+" (+https://github.com/bavanchun/Typeburn)")
   req.Header.Set("Accept", "application/vnd.github+json")
   ...
   body := io.LimitReader(resp.Body, 64*1024)
   ```
   Return typed error on 4xx/5xx (`ErrUpstream`, `ErrRateLimit` for 403/429).
7. **`internal/update/client_test.go`** — `httptest.NewServer`. Mock 200/403/
   404/429/malformed-JSON/slow-(>1.5s)-response cases. Verify UA header is set,
   verify body cap (send 200KB, assert <=64KB read).
8. **`internal/update/cache.go`** + `cache_test.go`. Uses `storage.AtomicWrite`
   + `config.StateDir()`. Defensive load (corrupt → return `Result{}`,
   `false, nil`). 24h TTL is `time.Since(r.CheckedAt) < 24*time.Hour`.
9. **`internal/update/check.go`** + `check_test.go`. Orchestrator:
   - Dev-skip: <!-- SUPERSEDED by red-team-findings 10+14 — see consolidated pseudocode at bottom of phase. -->
     `v := strings.ToLower(strings.TrimSpace(currentVer)); if v == "" || v == "dev" || v == "unknown" || strings.HasPrefix(v, "v0.0.0-") { return nil, nil }`
   - If `!force`: try cache; if fresh, return cached result.
   - Else: `FetchLatest`; if `IsPrerelease(rel.TagName) || rel.Draft || rel.Prerelease`, treat as up-to-date.
   - Build `Result{}` with `CheckedAt: time.Now().UTC()`, `Save()` to cache, return.
10. **Run** `make test-race`, `make lint`, `make size-check`. All green.
11. **Verify** `go list -deps ./internal/update/ | grep -E 'bubbletea|lipgloss'`
    returns empty (UI-free guarantee).

## Todo List

- [ ] storage.atomicWrite → storage.AtomicWrite (export + caller updates)
- [ ] config.StateDir() helper + test
- [ ] internal/update/result.go (types only)
- [ ] compare.go + compare_test.go (TDD)
- [ ] prerelease.go + prerelease_test.go
- [ ] client.go + client_test.go (httptest.Server, 6+ cases)
- [ ] cache.go + cache_test.go (TempDir, TTL, corrupt-file)
- [ ] check.go + check_test.go (integration with stubs)
- [ ] make test-race + make lint + make size-check green
- [ ] UI-free guarantee verified via `go list -deps`

## Success Criteria

- [ ] All new files < 200 LOC each.
- [ ] `internal/update/` imports: stdlib + `internal/storage` + `internal/config` only.
- [ ] `go test ./internal/update/... -race -count=1` passes.
- [ ] Zero `go.mod` diff.
- [ ] Client tests use `httptest.Server` only (no real network).
- [ ] Cache tests use `t.TempDir()` (no filesystem leakage).

## Risk Assessment

| Risk | Mitigation |
|---|---|
| Exporting `AtomicWrite` ripples through `internal/storage` callers | Compiler-checked rename; tests catch breakage |
| `StateDir()` differs from existing dir helpers' error semantics | Mirror existing pattern exactly; copy test shape |
| `httptest.Server` tests flake on slow CI | Use 200ms deadline in mock; not the 1.5s real-world cap |
| Pseudo-version detection misses a `go install` output form | Test the 4 known forms; document as out-of-scope edge case in code comment |
| 64KB cap truncates legitimate response | Real releases are 5-15KB (researcher-01 §1); 64KB has 4x headroom |

## Security Considerations

- No auth — public API only. UA header is identifying but not sensitive.
- TLS via stdlib defaults. Body size cap prevents DoS via runaway response.
- Cache file at `$XDG_STATE_HOME/typeburn/update-check.json` mode 0600
  (same as existing `internal/storage` patterns).
- No PII written to cache (just version strings + URL + timestamp).

## Next Steps

- Phase 2: wire `UpdateCheck bool` into config so Phase 4 can gate on it.
- Phase 3: explicit flag consumer of this package.

## Red Team Review Updates (2026-05-21)

Apply these deltas during implementation; supersedes the original spec where they conflict.

<!-- red-team-finding-2 --> **C2 — cache parent dir.** `internal/storage/atomic_write.go:13` documents
"parent directory must exist". `cache.Save` MUST call
`os.MkdirAll(filepath.Dir(path), 0700)` BEFORE invoking the atomic writer.
Without this, fresh installs silently fail to persist cache → users pay the
1.5s timeout every launch and hit GitHub's 60/hr rate limit on shared NAT.
Add a unit test that deletes the parent dir between Save calls.

<!-- red-team-finding-5 --> **H3 — `CheckRedirect` policy.** Step 6 `http.Client` MUST set:
```go
CheckRedirect: func(req *http.Request, via []*http.Request) error {
    return http.ErrUseLastResponse // /releases/latest is 200 on happy path
}
```
This blocks attacker-controlled 302 redirects from leaking UA/version and from downgrading scheme.

<!-- red-team-finding-6 --> **H4 — cache value re-validation on load.** In `cache.Load`, AFTER successful
JSON parse, re-validate:
- `Latest` matches `^v?\d+\.\d+\.\d+([-+.][\w.-]+)?$` (regexp).
- `ReleaseURL` starts with `https://github.com/bavanchun/Typeburn/`.
- `SchemaVersion == 1` (see finding 12).
On any mismatch, return zero-value Result (treat as "no fresh cache"). Prevents
ANSI injection from a maliciously-seeded cache file rendering into the TUI footer.

<!-- red-team-finding-7 --> **H5 — symlink/TOCTOU.** Do NOT promote `storage.atomicWrite` to `storage.AtomicWrite`.
Instead, **duplicate ~30 LOC into `internal/update/cache.go`** (private helper) and harden
the copy: open temp with `os.O_WRONLY|os.O_CREATE|os.O_EXCL|os.O_NOFOLLOW`, use
unique temp suffix `fmt.Sprintf("%s.%d.tmp", path, os.Getpid())` to avoid concurrent-Save
rename collisions. Leave `internal/storage/atomic_write.go` untouched. Update Implementation
Steps §1 and §8 accordingly. **Trade-off:** ~30 LOC duplication accepted (KISS) vs.
hardening a shared helper that other packages didn't ask for.

<!-- red-team-finding-9 --> **H7 — TLS handshake bound + opportunistic timeout.** Step 6 Transport MUST set:
```go
Transport: &http.Transport{
    Proxy:                 http.ProxyFromEnvironment,
    DialContext:           (&net.Dialer{Timeout: 800 * time.Millisecond}).DialContext,
    TLSHandshakeTimeout:   800 * time.Millisecond,
    ResponseHeaderTimeout: 800 * time.Millisecond,
}
```
Explicit-flag (Phase 3) path keeps the 1.5s outer cap. Opportunistic (Phase 4) path
uses 800ms outer cap. Without these, `http.Client.Timeout` alone does NOT bound
TLS handshake on degraded networks.

<!-- red-team-finding-10 --> **M1 — `(devel)` is unreachable.** Verified at `internal/version/version.go:46`:
`Resolve()` explicitly filters `(devel)` and falls back to `"dev"` at line 65-67.
Drop `"(devel)"` from the dev-skip list. Replace step 9's pseudo-code with the
trim/lowercase form (finding 14).

<!-- red-team-finding-11 --> **M2 — SemVer-2 prerelease detection.** Primary check is the API's `rel.Prerelease`
boolean flag AND `rel.Draft`. The locked-text substring denylist (`-rc/-beta/-alpha/-pre`)
is **belt-and-suspenders only**. For full SemVer compliance, additionally treat any
tag with content after `MAJOR.MINOR.PATCH` (preceded by `-`) as prerelease — except
the git-describe form `vX.Y.Z-N-gSHA` (those have purely numeric suffix segments).

<!-- red-team-finding-12 --> **M3 — schema_version on cache.** Add `SchemaVersion int \`json:"schema_version"\``
field to `Result` (constant `1` in this release). On load, if `SchemaVersion != 1`,
treat as empty cache (force refresh). Matches `internal/cli/cmd_replay.go:18` repo
convention. On save, write `SchemaVersion: 1`.

<!-- red-team-finding-13 --> **M4 — clock-skew TTL clamp.** Step 8 TTL check becomes:
```go
now := time.Now().UTC()
age := now.Sub(r.CheckedAt)
if age < 0 || age > 7*24*time.Hour { return zeroResult }  // future-dated or wildly stale → refresh
if age < 24*time.Hour { return r, true }                  // fresh
return zeroResult                                          // stale
```
Prevents clock-skew breaking TTL both directions (cloned VMs, suspended laptops, NTP drift).

<!-- red-team-finding-14 --> **M8 — dev-skip whitespace/case-insensitive.** Replace step 9's exact-string
compare with:
```go
v := strings.ToLower(strings.TrimSpace(currentVer))
if v == "" || v == "dev" || v == "unknown" || strings.HasPrefix(v, "v0.0.0-") {
    return nil, nil // dev-skip
}
```
Note: `(devel)` removed per finding 10; `unknown` added defensively.

<!-- red-team-finding-15 --> **M12 — `Resolve()` signature pin.** Verified at `internal/version/version.go:28-32, 40`:
returns struct `Info{Version, Commit, Date string}`. Phase 1 prose "returns three
strings" is incorrect — fix to "returns `Info` struct". All callers use
`version.Resolve().Version`. Aligns with Phase 3 (already correct).

### Step 9 (consolidated post-redteam pseudocode)

```go
func Check(ctx context.Context, currentVer string, force bool) (*Result, error) {
    v := strings.ToLower(strings.TrimSpace(currentVer))
    if v == "" || v == "dev" || v == "unknown" || strings.HasPrefix(v, "v0.0.0-") {
        return nil, nil // dev/pseudo-version skip (finding 10, 14)
    }
    if !force {
        if cached, fresh := cacheLoad(); fresh { return cached, nil }  // findings 6, 12, 13
    }
    rel, err := FetchLatest(ctx, currentVer)
    if err != nil { return nil, err } // silent-degrade at caller
    if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
        return &Result{Current: currentVer, Latest: currentVer, UpgradeAvailable: false, CheckedAt: time.Now().UTC(), SchemaVersion: 1}, nil
    }
    upgrade := Compare(currentVer, rel.TagName) < 0
    r := &Result{Current: currentVer, Latest: rel.TagName, UpgradeAvailable: upgrade, ReleaseURL: rel.HTMLURL, CheckedAt: time.Now().UTC(), SchemaVersion: 1}
    _ = cacheSave(r) // silent on error
    return r, nil
}
```
