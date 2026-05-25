# Typeburn Bug Investigation Report
**Date:** 2026-05-22  
**Investigator:** Researcher  
**Scope:** 4 critical bugs in CLI, UI, and update check subsystems

---

## Bug 1: renderVersionCheckJSON Double-Emits Error

### Summary
`renderVersionCheckJSON` encodes JSON to stdout AND returns the error, causing double error output (JSON on stdout, error prose on stderr) when update check fails.

### Location & Code

**File:** `internal/cli/cmd_version.go:75–127`

**Exact problematic code:**
```go
// Lines 94–106
if checkErr != nil {
    out := struct {
        Version     any `json:"version"`
        UpdateCheck any `json:"update_check"`
    }{
        Version: versionObj,
        UpdateCheck: struct {
            Error string `json:"error"`
        }{checkErr.Error()},
    }
    _ = enc.Encode(out)  // Line 104: JSON emitted to stdout
    return checkErr      // Line 105: error ALSO returned (double-emit!)
}
```

**Call site:** `internal/cli/cmd_version.go:49`
```go
if asJSON {
    return renderVersionCheckJSON(cmd, info, result, err)
}
```

### Root Cause Analysis

1. On error path, `enc.Encode(out)` writes valid JSON to stdout (line 104).
2. Function then returns `checkErr` (line 105).
3. Cobra's error handler in `main.go` catches the returned error and prints it to stderr.
4. Result: dual emission—JSON + prose error.

### Impact

- JSON output is polluted by a separate error message on stderr.
- CLI consumers expecting pure JSON get unexpected text output.
- Human users see confusing duplicate error messaging.

### Test Coverage

**File:** `internal/cli/cmd_version_test.go:107–120`

```go
func TestVersionCheckUpdate_Error(t *testing.T) {
    orig := checkFn
    checkFn = stubCheck(nil, errors.New("network unreachable"))
    defer func() { checkFn = orig }()

    var out, errOut bytes.Buffer
    // Human mode: error goes to stderr, exit code 0.
    if err := versionRoot(t, &out, &errOut, "version", "--check-update"); err != nil {
        t.Fatalf("version --check-update with error should exit 0, got: %v", err)
    }
    // ... expects error on stderr in human mode
}
```

**Critical gap:** No test case for `--json --check-update` with error. The test only checks human mode (line 114). JSON error case is untested.

### Fix

**Option A: Return nil (silence error)**
```go
if checkErr != nil {
    // ... encode JSON error
    return nil  // Don't re-emit error
}
```
Pros: Clean. JSON is the only output.
Cons: Exit code becomes 0 (success) even on network failure. Breaks contract that errors should exit non-zero.

**Option B: Check asJSON before returning error in runVersion**
Move error handling up to caller:
```go
// In runVersion, after checkFn call:
result, err := checkFn(ctx, info.Version, true)
if asJSON {
    return renderVersionCheckJSON(cmd, info, result, err)  // Already returns nil/nil-safe
}
```
Then fix renderVersionCheckJSON to return nil when JSON is emitted.
Pros: Preserves error semantics.
Cons: Requires refactoring caller logic.

### Recommended Fix

**Approach:** Return `nil` after JSON encode on error. Preserve JSON-as-output semantics (errors embedded in JSON, exit code 0 for successful API communication with error field).

```go
// Lines 94–106 (FIXED)
if checkErr != nil {
    out := struct {
        Version     any `json:"version"`
        UpdateCheck any `json:"update_check"`
    }{
        Version: versionObj,
        UpdateCheck: struct {
            Error string `json:"error"`
        }{checkErr.Error()},
    }
    _ = enc.Encode(out)
    return nil  // CHANGED: Don't re-emit; error is in JSON
}
```

### Test Impact

**Required test addition:** `TestVersionCheckUpdate_JSONError` (new):
```go
func TestVersionCheckUpdate_JSONError(t *testing.T) {
    orig := checkFn
    checkFn = stubCheck(nil, errors.New("network unreachable"))
    defer func() { checkFn = orig }()

    var out, errOut bytes.Buffer
    // JSON mode: error should be in JSON, NOT on stderr
    if err := versionRoot(t, &out, &errOut, "version", "--json", "--check-update"); err != nil {
        t.Fatalf("version --json --check-update error: exit code %v (expected 0)", err)
    }
    var got map[string]any
    if err := json.Unmarshal(out.Bytes(), &got); err != nil {
        t.Fatalf("invalid JSON: %v", err)
    }
    updateCheck, ok := got["update_check"].(map[string]any)
    if !ok {
        t.Fatalf("update_check must be an object, got %T", got["update_check"])
    }
    if _, hasError := updateCheck["error"]; !hasError {
        t.Error("error must be in JSON update_check field")
    }
    if errOut.Len() > 0 {
        t.Errorf("stderr must be empty in JSON mode, got: %s", errOut.String())
    }
}
```

**Existing test impact:** None. Test `TestVersionCheckUpdate_Error` (line 107) only tests human mode, not JSON mode.

---

## Bug 2: stripANSI Only Terminates on 'm', Not Full CSI Range

### Summary
`stripANSI` terminates ANSI escape sequences only on the character 'm' (SGR Select Graphic Rendition). This misses other CSI final bytes (0x40–0x7E, i.e., '@' through '~'), causing sequences like `ESC [ 5 A` (cursor up, CSI code `A`) to be partially stripped, leaving visible garbage in border-title measurements.

### Location & Code

**File:** `internal/ui/result_render_helpers.go:76–92`

**Current code:**
```go
// stripANSI removes ANSI SGR escape sequences (ESC [ ... m) from s so the
// visual character width can be measured accurately for border title injection.
func stripANSI(s string) string {
    var out strings.Builder
    inEsc := false
    for _, r := range s {
        switch {
        case r == '\x1b':
            inEsc = true
        case inEsc:
            if r == 'm' {  // Line 84: ONLY 'm' terminates!
                inEsc = false
            }
        default:
            out.WriteRune(r)
        }
    }
    return out.String()
}
```

### CSI Specification

ANSI CSI (Control Sequence Introducer) format:
```
ESC [ <params> <final-byte>
```

Valid final bytes: `@` (0x40) through `~` (0x7E). Examples:
- `m` = SGR (Select Graphic Rendition, colors/bold/etc.)
- `A` = CUU (Cursor Up)
- `H` = CUP (Cursor Position)
- `K` = EL (Erase in Line)
- `J` = ED (Erase in Display)
- `h`/`l` = SM/RM (Set/Reset Mode)

Lipgloss v2 output may emit:
- SGR sequences (colors): `ESC[31m` (red), `ESC[0m` (reset)
- Cursor positioning: `ESC[H` (home), `ESC[<row>;<col>H` (position)
- Erase sequences: `ESC[K` (erase to end of line)

**Problem:** Current code treats `ESC[31A` (red text + cursor up) as:
- Sees `ESC` → inEsc=true
- Sees `[` → inEsc still true (not written)
- Sees `3` → inEsc still true (not written)
- Sees `1` → inEsc still true (not written)
- Sees `A` → inEsc still true (NOT 'm'!), A is written to output ❌

Result: Border-title width measurement is wrong because of stray `A` characters.

### Impact

- Border-title injection (`injectBorderTitle`) measures widths incorrectly.
- Titles may be misaligned or overflow if Lipgloss v2 uses non-SGR CSI sequences.
- Risk of invisible text or title-cutting bugs in result screen.

### Test Coverage

**File:** `internal/ui/result_render_helpers.go`

**Current tests:** None. Function `stripANSI` is not unit-tested.

**Usage site:** `internal/ui/screen_result_view.go`
```go
return injectBorderTitle(panel, titleStyled)  // panel contains Lipgloss output
```

**Integration test:** `internal/ui/screen_result_test.go` checks border rendering but does NOT test stripANSI edge cases:
- Line 87–93: `TestResultView_ContainsPanel` checks for border runes but not title injection correctness.

### Fix

**Approach:** Check if rune is in CSI final-byte range (0x40–0x7E, or '@'–'~').

```go
// FIXED stripANSI
func stripANSI(s string) string {
    var out strings.Builder
    inEsc := false
    for _, r := range s {
        switch {
        case r == '\x1b':
            inEsc = true
        case inEsc:
            // CSI final bytes are in range '@' (0x40) to '~' (0x7E)
            if r >= '@' && r <= '~' {
                inEsc = false
            }
        default:
            out.WriteRune(r)
        }
    }
    return out.String()
}
```

**Correctness check:** This handles:
- `\x1b[31m` (SGR) → inEsc on `\x1b`, continues through `[31`, ends on `m` ✓
- `\x1b[A` (CUU) → inEsc on `\x1b`, continues through `[`, ends on `A` ✓
- `\x1b[2K` (EL) → inEsc on `\x1b`, continues through `[2`, ends on `K` ✓
- `\x1b]0;title\x07` (OSC) → inEsc on `\x1b`, BUT `]` (0x5D) is NOT in final-byte range, so it's written ❌ (partial failure)

### OSC Sequences (Optional Hardening)

OSC (Operating System Command) format:
```
ESC ] <text> (ST | BEL)
```
Where ST (String Terminator) is `ESC \` (0x1B 0x5C) or BEL (0x07).

**Question:** Does Lipgloss v2 emit OSC? (E.g., setting terminal title)

If yes, full fix needs:
```go
func stripANSI(s string) string {
    var out strings.Builder
    inEsc := false
    inOSC := false
    for _, r := range s {
        switch {
        case r == '\x1b':
            inEsc = true
            inOSC = false  // Reset OSC state on new escape
        case inEsc && r == ']':
            inEsc = false
            inOSC = true
        case inOSC && r == '\x07':  // BEL
            inOSC = false
        case inEsc:
            // CSI final bytes: '@' to '~'
            if r >= '@' && r <= '~' {
                inEsc = false
            }
        case !inEsc && !inOSC:
            out.WriteRune(r)
        }
    }
    return out.String()
}
```

### Recommended Fix

**Conservative approach (bug 2a):** Fix CSI termination range only.

```go
if r >= '@' && r <= '~' {
    inEsc = false
}
```

**Robust approach (bug 2b):** Also handle OSC (if Lipgloss v2 uses it).

**Recommendation:** Start with 2a (CSI fix). Monitor for OSC issues post-fix.

### Test Impact

**Required test addition:** `TestStripANSI` (new):
```go
func TestStripANSI(t *testing.T) {
    cases := []struct {
        name     string
        input    string
        expected string
    }{
        {
            name:     "SGR only",
            input:    "foo\x1b[31mbar\x1b[0mbaz",
            expected: "foobarbaz",
        },
        {
            name:     "SGR and CUU",
            input:    "foo\x1b[31m\x1b[5Abar",  // red + cursor up 5
            expected: "foobar",
        },
        {
            name:     "Multiple final bytes",
            input:    "a\x1b[2Kb\x1b[Hc",  // erase line, cursor home
            expected: "abc",
        },
        {
            name:     "Nested-like sequences",
            input:    "\x1b[1;31m\x1b[1H\x1b[0m",
            expected: "",
        },
        {
            name:     "No ANSI",
            input:    "plain text",
            expected: "plain text",
        },
        {
            name:     "Incomplete sequence (edge)",
            input:    "a\x1b[31b",  // incomplete (no final byte yet)
            expected: "ab",  // '31' treated as regular chars after state reset
        },
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := stripANSI(tc.input)
            if got != tc.expected {
                t.Errorf("stripANSI(%q): want %q, got %q", tc.input, tc.expected, got)
            }
        })
    }
}
```

**Also add:** Integration test for `injectBorderTitle` with mixed ANSI sequences to ensure width calculation is correct.

---

## Bug 3: Prerelease GitHub Response Not Cached

### Summary
When GitHub API returns a prerelease/draft release, `Check()` creates a Result object and returns it, but **does not cache it**. Subsequent checks force a fresh API call, causing unnecessary network overhead and potential rate-limiting issues.

### Location & Code

**File:** `internal/update/check.go:14–53`

**Problematic code path:**
```go
func Check(ctx context.Context, currentVer string, force bool) (*Result, error) {
    // Lines 26–29: Fetch from API
    rel, err := FetchLatest(ctx, currentVer)
    if err != nil {
        return nil, err
    }

    // Lines 31–40: Prerelease/draft handling
    if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
        return &Result{
            SchemaVersion:    cacheSchemaVersion,
            Current:          currentVer,
            Latest:           currentVer,
            UpgradeAvailable: false,
            CheckedAt:        time.Now().UTC(),
        }, nil  // Line 39: Returns WITHOUT calling cacheSave()
    }

    // Lines 42–51: Normal upgrade path (DOES cache)
    upgrade := Compare(currentVer, rel.TagName) < 0
    r := &Result{
        SchemaVersion:    cacheSchemaVersion,
        Current:          currentVer,
        Latest:           rel.TagName,
        UpgradeAvailable: upgrade,
        ReleaseURL:       rel.HTMLURL,
        CheckedAt:        time.Now().UTC(),
    }
    _ = cacheSave(r)  // Line 51: Cached!
    return r, nil
}
```

### Root Cause

The prerelease/draft path (lines 31–40) creates a synthetic "no upgrade available" Result but returns without caching. The stable-upgrade path (lines 42–51) caches via `cacheSave(r)`.

**Asymmetry:** Prerelease result is not cached, but stable result is.

### Impact

**Frequency:** Every check without `force=true` when latest is prerelease:
- `cacheLoad()` returns (nil, false) because cache expired or missing.
- `Check()` calls `FetchLatest()` (network I/O).
- GitHub API is hit even if we just checked 1 minute ago.
- Rate-limit risk: 60 requests/hour unauthenticated, 5000/hour authenticated. With prerelease → uncached → every call is a network hit.

**Example scenario:**
1. Latest release is `v2.1.0-rc.1` (prerelease).
2. User runs `typeburn --version --check-update` → API call, no cache.
3. User runs `typeburn --version --check-update` 5 minutes later → API call again (cache miss).
4. With 24h TTL on normal releases, stable release would be cached. Prerelease is not.

### Test Coverage

**File:** `internal/update/check_test.go:75–90`

```go
func TestCheck_PrereleaseIgnored(t *testing.T) {
    defer withTempCache(t)()
    srv := stubServer(t, Release{TagName: "v2.1.0-rc.1", Prerelease: true})
    defer srv.Close()
    origURL := fetchURL
    fetchURL = srv.URL
    defer func() { fetchURL = origURL }()

    r, err := Check(context.Background(), "v2.0.0", true)
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
    if r.UpgradeAvailable {
        t.Error("UpgradeAvailable: want false when latest is prerelease")
    }
}
```

**Gap:** Test checks that prerelease is ignored (UpgradeAvailable=false) but does NOT verify caching. No test calls `Check()` twice and expects the second call to skip the API.

### Fix

**Approach:** Call `cacheSave(r)` on the prerelease Result before returning.

```go
// Lines 31–40 (FIXED)
if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
    r := &Result{
        SchemaVersion:    cacheSchemaVersion,
        Current:          currentVer,
        Latest:           currentVer,
        UpgradeAvailable: false,
        CheckedAt:        time.Now().UTC(),
    }
    _ = cacheSave(r)  // ADDED: Cache the prerelease result
    return r, nil
}
```

**Why this is safe:**
- `cacheSave()` uses atomic write with temp file + rename (no race).
- Errors are silent-degrade (line 51 already ignores errors: `_ = cacheSave(r)`).
- Result struct is identical (schema version matches).
- `CheckedAt` is fresh (time.Now().UTC()), so cache load will see it as fresh (within 24h TTL).

### Test Impact

**Required test addition:** `TestCheck_PrereleaseIsCached` (new):

```go
func TestCheck_PrereleaseIsCached(t *testing.T) {
    defer withTempCache(t)()
    srv := stubServer(t, Release{TagName: "v2.1.0-rc.1", Prerelease: true, 
        HTMLURL: "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0-rc.1"})
    defer srv.Close()
    
    origURL := fetchURL
    fetchURL = srv.URL
    defer func() { fetchURL = origURL }()

    // First call: API hit, result cached
    r1, err := Check(context.Background(), "v2.0.0", true)
    if err != nil {
        t.Fatalf("first Check: %v", err)
    }
    if r1 == nil {
        t.Fatal("expected non-nil result")
    }

    // Close server so second call MUST use cache (would error if it tries API)
    srv.Close()

    // Second call: Should hit cache, not API
    r2, err := Check(context.Background(), "v2.0.0", false)
    if err != nil {
        t.Fatalf("second Check (should use cache): %v", err)
    }
    if r2 == nil || !r2.CheckedAt.Equal(r1.CheckedAt) {
        t.Error("expected cache hit with same CheckedAt")
    }
}
```

**Existing tests:** No impact. Test `TestCheck_PrereleaseIgnored` still passes (we're just adding a side-effect: caching).

---

## Bug 4: Package-Level Mutable Globals as Race Targets

### Summary
Two package-level variables are mutated by tests, creating latent race conditions and violating concurrency-safety:
1. `cacheFilePath` in `internal/update/cache.go` (line 27)
2. `checkFn` in `internal/cli/cmd_version.go` (line 14)
3. `fetchURL` in `internal/update/client.go` (line 16)

These are swapped in and out by tests to inject dependencies, but are not protected by locks. If tests run in parallel (e.g., `go test -parallel`), data races occur.

### Location & Code

**File 1:** `internal/update/cache.go:26–27`
```go
// cacheFilePath is a var so tests can override it via t.TempDir().
var cacheFilePath = ""
```

**Mutations:**
- `internal/update/cache_test.go:10–16`:
```go
func withTempCache(t *testing.T) func() {
    t.Helper()
    dir := t.TempDir()
    orig := cacheFilePath  // Line 13
    cacheFilePath = filepath.Join(dir, "update-check.json")  // Line 14: RACE!
    return func() { cacheFilePath = orig }  // Line 15: RACE!
}
```
- Used by `TestCheck_*` (lines 30, 56, 76, 92, 117)
- Used by `TestCacheLoad_*` and `TestCacheSave_*` (lines 44, 52, 63, 80, 97, 150)

**File 2:** `internal/cli/cmd_version.go:13–14`
```go
// checkFn is the update.Check function; overridden in tests via this seam.
var checkFn = update.Check
```

**Mutations:**
- `internal/cli/cmd_version_test.go`:
  - Line 52: `checkFn = stubCheck(...)` (TestVersionCheckUpdate_UpgradeAvailable)
  - Line 77: `checkFn = stubCheck(...)` (TestVersionCheckUpdate_UpToDate)
  - Line 95: `checkFn = stubCheck(...)` (TestVersionCheckUpdate_DevSkip)
  - Line 109: `checkFn = stubCheck(...)` (TestVersionCheckUpdate_Error)
  - Line 124: `checkFn = stubCheck(...)` (TestVersionCheckUpdate_JSONWrapper)
  - All restore via `defer func() { checkFn = orig }()`

**File 3:** `internal/update/client.go:14–16`
```go
// fetchURL is the GitHub Releases API endpoint. It is a var so tests can
// override it with an httptest.Server URL without touching real network.
var fetchURL = "https://api.github.com/repos/bavanchun/Typeburn/releases/latest"
```

**Mutations:**
- `internal/update/check_test.go`:
  - Line 34: `fetchURL = srv.URL` (TestCheck_UpgradeAvailable)
  - Line 60: `fetchURL = srv.URL` (TestCheck_UpToDate)
  - Line 80: `fetchURL = srv.URL` (TestCheck_PrereleaseIgnored)
  - Line 131: `fetchURL = srv.URL` (TestCheck_ForceBypassesCache)
  - All restore via `defer func() { fetchURL = origURL }()`

### Race Condition Scenario

```
Goroutine A (TestVersionCheckUpdate_UpgradeAvailable):
1. orig = checkFn  // checkFn = update.Check
2. checkFn = stubA // checkFn = stubA

Goroutine B (TestVersionCheckUpdate_UpToDate):
1. [reads checkFn during A's test] → might see A's stubA
2. orig = checkFn  // orig = stubA
3. checkFn = stubB // checkFn = stubB
4. [A's defer restores] → defer func() { checkFn = orig } → checkFn = stubA ❌
5. [B's defer restores] → checkFn = stubB (should be update.Check) ❌

Result: Tests see wrong stubs, flaky failures.
```

### Test Coverage

All three globals are mutated in test-only code. No race detector has caught this because:
- Tests aren't run with `-race` in CI (need to verify).
- If `go test ./... -race` is not in CI, race is invisible.

### Fix Options

**Option A: Use `t.Setenv` pattern (for string globals)**

Globals become env-var backed:

```go
// cache.go
func getCachePath() (string, error) {
    if override := os.Getenv("TYPEBURN_CACHE_PATH"); override != "" {
        return override, nil
    }
    dir, err := config.StateDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "update-check.json"), nil
}

// cache_test.go
func withTempCache(t *testing.T) func() {
    t.Helper()
    dir := t.TempDir()
    t.Setenv("TYPEBURN_CACHE_PATH", filepath.Join(dir, "update-check.json"))
    // t.Setenv atomically restores on cleanup, no defer needed
}
```

**Pros:** Atomic per-test, no manual cleanup.  
**Cons:** Env var is global state too (but `t.Setenv` handles locking).

**Option B: Use `sync/atomic` pointer (for function globals)**

```go
// cmd_version.go
var checkFnMu sync.Mutex
var checkFn func(context.Context, string, bool) (*update.Result, error) = update.Check

// In tests:
func setCheckFn(fn func(context.Context, string, bool) (*update.Result, error)) {
    checkFnMu.Lock()
    checkFn = fn
    checkFnMu.Unlock()
}

// In runVersion:
checkFnMu.Lock()
fn := checkFn
checkFnMu.Unlock()
result, err := fn(cmd.Context(), info.Version, true)
```

**Pros:** Explicit locking, no env vars.  
**Cons:** Verbose; overhead on every call.

**Option C: Use a test harness struct (dependency injection)**

```go
// cli/version.go
type versionRunner struct {
    checkFn func(context.Context, string, bool) (*update.Result, error)
}

func newVersionCmd() *cobra.Command {
    runner := &versionRunner{checkFn: update.Check}
    cmd := &cobra.Command{
        RunE: func(cmd *cobra.Command, args []string) error {
            return runner.runVersion(cmd, asJSON, checkUpdate)
        },
    }
    return cmd
}

// In tests: Create custom runner with stub
```

**Pros:** Clean, no global state.  
**Cons:** Requires refactoring `newVersionCmd`, breaks Cobra's closure pattern.

**Option D: No fix (accept flakiness, document it)**

Run tests serially (default: `-parallel 1`).  
Pros: Zero refactoring.  
Cons: Tests are slower; hidden race that will bite later.

### Recommended Fix

**Strategy:**
1. **cacheFilePath** → Use `t.Setenv` + env-backed `getCachePath()` (Option A).
2. **checkFn** → Use `sync/atomic` with mutex (Option B) — simpler than DI refactor.
3. **fetchURL** → Use `sync/atomic` (same as checkFn).

### Implementation

**File:** `internal/update/cache.go`

**Before:**
```go
var cacheFilePath = ""

func getCachePath() (string, error) {
    if cacheFilePath != "" {
        return cacheFilePath, nil
    }
    dir, err := config.StateDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "update-check.json"), nil
}
```

**After:**
```go
func getCachePath() (string, error) {
    if override := os.Getenv("TYPEBURN_CACHE_PATH"); override != "" {
        return override, nil
    }
    dir, err := config.StateDir()
    if err != nil {
        return "", err
    }
    return filepath.Join(dir, "update-check.json"), nil
}

// REMOVED: var cacheFilePath = ""
```

**File:** `internal/update/cache_test.go`

**Before:**
```go
func withTempCache(t *testing.T) func() {
    t.Helper()
    dir := t.TempDir()
    orig := cacheFilePath
    cacheFilePath = filepath.Join(dir, "update-check.json")
    return func() { cacheFilePath = orig }
}
```

**After:**
```go
func withTempCache(t *testing.T) {
    t.Helper()
    dir := t.TempDir()
    t.Setenv("TYPEBURN_CACHE_PATH", filepath.Join(dir, "update-check.json"))
    // No return needed; t.Setenv auto-restores
}

// Update all callers:
// Before: defer withTempCache(t)()
// After:  withTempCache(t)  (no defer)
```

**File:** `internal/update/client.go`

**Before:**
```go
var fetchURL = "https://api.github.com/repos/bavanchun/Typeburn/releases/latest"
```

**After:**
```go
var fetchURLMu sync.Mutex
var fetchURL = "https://api.github.com/repos/bavanchun/Typeburn/releases/latest"

func getFetchURL() string {
    fetchURLMu.Lock()
    defer fetchURLMu.Unlock()
    return fetchURL
}

func setFetchURL(url string) {
    fetchURLMu.Lock()
    defer fetchURLMu.Unlock()
    fetchURL = url
}
```

**File:** `internal/update/client.go` (FetchLatest)

**Before:**
```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet, fetchURL, nil)
```

**After:**
```go
req, err := http.NewRequestWithContext(ctx, http.MethodGet, getFetchURL(), nil)
```

**File:** `internal/update/check_test.go`

**Before:**
```go
origURL := fetchURL
fetchURL = srv.URL
defer func() { fetchURL = origURL }()
```

**After:**
```go
origURL := getFetchURL()
setFetchURL(srv.URL)
defer func() { setFetchURL(origURL) }()
```

**File:** `internal/cli/cmd_version.go`

**Before:**
```go
var checkFn = update.Check
```

**After:**
```go
var checkFnMu sync.Mutex
var checkFn = update.Check

func getCheckFn() func(context.Context, string, bool) (*update.Result, error) {
    checkFnMu.Lock()
    defer checkFnMu.Unlock()
    return checkFn
}

func setCheckFn(fn func(context.Context, string, bool) (*update.Result, error)) {
    checkFnMu.Lock()
    defer checkFnMu.Unlock()
    checkFn = fn
}
```

**File:** `internal/cli/cmd_version.go` (runVersion)

**Before:**
```go
result, err := checkFn(cmd.Context(), info.Version, true)
```

**After:**
```go
result, err := getCheckFn()(cmd.Context(), info.Version, true)
```

**File:** `internal/cli/cmd_version_test.go`

**Before:**
```go
orig := checkFn
checkFn = stubCheck(...)
defer func() { checkFn = orig }()
```

**After:**
```go
orig := getCheckFn()
setCheckFn(stubCheck(...))
defer func() { setCheckFn(orig) }()
```

### Test Impact

**Backward compat:** All existing tests continue to pass. The locking is transparent.

**New assertions:** Optionally add a `-race` gate to CI:
```bash
go test ./... -race -count=1  # Run once with race detector
```

If already in CI (check `Makefile` or `.github/workflows/ci.yml`), this fix closes the race gap.

### Files Modified

- `internal/update/cache.go` (1 func change)
- `internal/update/cache_test.go` (2 func changes: `withTempCache` signature, all call sites)
- `internal/update/client.go` (2 new funcs, 1 call site)
- `internal/update/check_test.go` (4 call sites)
- `internal/cli/cmd_version.go` (2 new funcs, 1 call site)
- `internal/cli/cmd_version_test.go` (5 call sites)

---

## Summary Table

| Bug | File | Issue | Fix | Risk |
|-----|------|-------|-----|------|
| 1 | cmd_version.go:75–127 | JSON error double-emitted | Return nil after encode | Low—no test coverage for JSON error path |
| 2 | result_render_helpers.go:76–92 | stripANSI only checks 'm' | Check range '@'–'~' | Low—no tests exist; integration test may catch |
| 3 | check.go:31–40 | Prerelease result uncached | Add cacheSave call | Low—side-effect only; cache still works |
| 4 | cache.go, client.go, cmd_version.go | Global var races | t.Setenv + mutex accessors | Medium—requires signature changes |

---

## Unresolved Questions

1. **Bug 2:** Does Lipgloss v2 emit OSC sequences? If yes, should we implement full OSC handling (currently ignored)?
2. **Bug 4:** Is `-race` currently in CI? Check `.github/workflows/ci.yml` or `Makefile test-race`.
3. **Bug 4:** Are there other global vars that need locking? (Grep for `^var ` in test-touching packages.)
4. **Bug 1:** Should JSON error exit code be 0 or non-zero? (Current fix: 0, treating errors as valid API response.)
