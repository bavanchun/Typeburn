---
name: xdg-storage-semver-research
description: XDG storage patterns, state-dir strategy, and stdlib-only semver comparator design
---

## Part A: Existing Storage Architecture

### A.1 Atomic Write Pattern

**Location:** `internal/storage/atomic_write.go:14-45`

Signature (unexported):
```go
func atomicWrite(path string, data []byte) error
```

Pattern: Write to `path + ".tmp"` → fsync → rename over target. Parent dir created by callers (e.g., `SaveSettings` at line 59). Mode 0600, never panics on corrupt input. Public API: `SaveSettings(config.Settings)`, `AppendHistory(Record)`.

Key defensive behavior (settings_store.go:21-46): Missing/corrupt JSON → return `config.Defaults()` (never error, never panic). Caller logic validates fields via `Normalize()` to repair out-of-range enums.

### A.2 XDG Directory Resolution

**Location:** `internal/config/xdg-paths.go`

Functions exist:
- `ConfigDir()` → `$XDG_CONFIG_HOME/typeburn` (fallback `~/.config/typeburn`)
- `DataDir()` → `$XDG_DATA_HOME/typeburn` (fallback `~/.local/share/typeburn`)

Helper: `resolveDir(env, homeRel)` (line 27-36) — if env var is absolute, use it; else `$HOME/homeRel/appDir`. macOS has no XDG by default; HOME fallback is the normal path.

**CRITICAL GAP:** No `StateDir()` or `CacheDir()` exists. Must add one.

### A.3 Directory Spec (XDG Base Directory)

- `XDG_CONFIG_HOME` (default `~/.config`): settings.json — infrequently written, user-facing configuration.
- `XDG_DATA_HOME` (default `~/.local/share`): history.json — user data, append-only after tests.
- `XDG_STATE_HOME` (default `~/.local/state`): **correct for update-check.json** — transient cache, high-frequency reads, auto-invalidate by age.
- `XDG_CACHE_HOME` (default `~/.cache`): alternative for update-check; less stable (user may `rm ~/.cache` anytime).

**Decision:** Use `XDG_STATE_HOME`. It's the standard for session-like state (not user-facing settings, not long-term data, not throw-away cache). Existing codebase has no state-dir yet; must extend `internal/config/xdg-paths.go` with:
```go
func StateDir() (string, error) {
    return resolveDir("XDG_STATE_HOME", filepath.Join(".local", "state"))
}
```

## Part B: Version.Resolve() Behavior & Git-Describe Format

**Location:** `internal/version/version.go:34-69`

Signature:
```go
func Resolve() Info  // returns {Version, Commit, Date}
func (i Info) String() string  // formats banner
```

Precedence chain:
1. ldflags-injected `Version` var (set by Makefile: `-X github.com/bavanchun/Typeburn/internal/version.Version=...`)
2. If empty, read `debug.ReadBuildInfo()`: `Main.Version` for version, vcs.revision/vcs.time for commit/date
3. If `Main.Version` is `"(devel)"`, treat as empty (skip to step 3)
4. Final fallback: `Version = "dev"`

Test case (version_test.go:18-24): ldflags win; setVars override for testing.

### Actual Runtime Outputs

**Scenario 1: Released via GoReleaser (ldflags injected)**
- `Version = "v2.0.0"`, `Commit = "abc1234def"`, `Date = "2026-05-18T..."`
- Banner: `typeburn v2.0.0 (abc1234, 2026-05-18..., ...)`

**Scenario 2: `go install github.com/bavanchun/Typeburn@v2.0.0` (no ldflags, but tagged release)**
- `debug.ReadBuildInfo().Main.Version = "v2.0.0"` (Go module proxy embeds semantic version)
- `Version = "v2.0.0"` from fallback
- Commit/Date filled from vcs settings if available

**Scenario 3: `go install github.com/bavanchun/Typeburn@latest` on a non-tagged commit (dev install mid-branch)**
- `debug.ReadBuildInfo().Main.Version = "v2.0.0-0.20260520143000-abc1234"` (pseudo-version format)
- This is a **git-describe-like string**, not a release version
- Comparator must handle this as "pre-release of v2.0.0", not as a concrete version to compare

**Scenario 4: Bare `go run .` or fully offline**
- All empty → `Version = "dev"`

### Key Insight for Comparator
The current version string is **sometimes a pseudo-version** (`vX.Y.Z-N-gSHA`), which is NOT a release and should not trigger an update check. The comparator must:
- **Case A:** If local `Version` is a pseudo-version (contains `-` followed by digits and `-g`), skip the check (already on dev/unstable).
- **Case B:** If GitHub's `tag_name` is a pre-release (e.g., `v2.1.0-rc.1`), treat as "no stable update available."

## Part C: Settings Integration & Config Wiring

**Location:** `internal/cli/cmd_config.go` and `internal/config/settings.go`

Current `Settings` struct (config/settings.go:35-40):
```go
type Settings struct {
    Theme         string
    DefaultMode   Mode
    DefaultLength int
    BlinkCursor   bool
}
```

Config CLI uses a **rows + switch pattern** (cmd_config.go:86-136):
- `configRows(s)` → list of `[key, value]` rows
- `configGet(s, key)` → find row by key
- `configSet(s, key, value)` → switch on key name, parse, validate, mutate struct

To add `UpdateCheckEnabled bool`:
1. Add field to `Settings` struct (e.g., `UpdateCheckEnabled bool` at line 40)
2. Add default in `Defaults()` (suggest default `false` — opt-in, not forced)
3. Extend `configRows()` with new row
4. Extend `configSet()` with `case "update_check_enabled":`
5. Update `Normalize()` if needed (unlikely; bool has no invalid values)

**Load behavior:** Missing field in JSON → zero-value `false` (safe default). No special handling needed; json.Unmarshal ignores unknown fields; json.Marshal omits zero-bool.

## Part D: Semver Comparison Strategy

### Option 1: Stdlib-only 30-LOC Parser (RECOMMENDED)

Parse leading `v`, split by `.` and `-`, compare first 3 components as integers, skip pre-release flags.

**Pros:**
- KISS: no new deps, ~30 LOC, fully testable
- Explicit: handles all Typeburn's known version formats
- Defensive: gracefully skips malformed input

**Cons:** nil

**Algorithm outline:**
```
stripLeadingV(versionString) -> "X.Y.Z" or "X.Y.Z-..."
splitOnDash(stripped) -> ["X.Y.Z", preReleaseInfo...]
if preReleaseInfo contains "-rc", "-beta", "-alpha", "-pre" → SKIP (prerelease)
parseXYZ(first component) -> (x, y, z, error)
compareXYZ(local, remote) -> "update available", "up to date", "can't compare"
```

### Option 2: golang.org/x/mod/semver

**Status:** NOT in go.mod. Adding it requires explicit user approval (CLAUDE.md dependency policy).

**Pros:** Robust, well-tested, handles edge cases

**Cons:** Requires new dependency; violates current constraint; overkill for our use case

**Recommendation:** Mention but do NOT recommend for v2.1.0. Option 1 is sufficient.

### Option 3: Other

None identified. Stdlib has no semver package.

### Edge Cases & Filters

**Mandatory handling:**
1. Strip leading `v` (all Typeburn versions have it)
2. Parse `vX.Y.Z-N-gSHA` (git-describe mid-branch) → skip entire check (user is on dev build)
3. Reject pre-releases from GitHub API (`tag_name` contains `-rc`, `-beta`, `-alpha`, `-pre`, case-insensitive)
4. Reject `v0.0.0-rc.test` dry-run tag (see CLAUDE.md release procedure)
5. Compare numerically: `2.10.0 > 2.9.0` (strconv.Atoi, not string compare)
6. Malformed input → log warning, treat as "can't compare" (no update shown, but no error)

### Test Cases (Table-Driven)

Minimum table:
```
{local: "v2.0.0", remote: "v2.0.0"} → "up to date"
{local: "v2.0.0", remote: "v2.1.0"} → "update available"
{local: "v2.1.0", remote: "v2.0.0"} → "up to date"
{local: "v2.0.0", remote: "v2.0.1"} → "update available"
{local: "v2.0.1", remote: "v2.0.0"} → "up to date"
{local: "dev", remote: "v2.0.0"} → "can't compare" (or skip check entirely)
{local: "v2.0.0-5-gabc1234", remote: "v2.0.1"} → "skip check" (local is pseudo-version/dev)
{local: "v2.0.0", remote: "v2.1.0-rc.1"} → "no stable release available" (skip)
{local: "v2.0.0", remote: "v0.0.0-rc.test"} → "skip" (dry-run tag)
{local: "v2.a.0", remote: "v2.1.0"} → "can't compare" (malformed local)
{local: "v2.0.0", remote: "invalid"} → "can't compare" (malformed remote)
```

## Part E: 24h Cache File Strategy

**Path:** `$XDG_STATE_HOME/typeburn/update-check.json`

**Format:** JSON with timestamp and latest-stable tag_name.

**Atomic write:** Reuse `atomicWrite(path, []byte)` after JSON-marshaling cache struct.

**Load + validate age:** Read file → unmarshal → check `time.Since(Timestamp) < 24*time.Hour`. If missing, corrupt, or stale → fetch new data from GitHub API.

**API endpoint:** `GET https://api.github.com/repos/bavanchun/Typeburn/releases/latest` → read `tag_name` field.

## Unresolved Questions

1. Should local pseudo-version (`v2.0.0-5-gabc1234`) **skip the check entirely** or **log a message**? (Suggest: silent skip — user is dev/unstable, noise-free)
2. Should the 24h cache be per-run or global? (Suggest: global file with timestamp, checked at startup or on-demand)
3. Should the update notifier live in the TUI (startup banner) or CLI-only (`typeburn version --check`)? (Out of scope for this research; v2.1.0 spec pending)
4. **Config field naming:** `UpdateCheckEnabled`, `update_check_enabled`, or `check_updates`? (Suggest: `update_check_enabled` for consistency with existing hyphen style)
