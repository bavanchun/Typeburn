---
phase: 2
title: "Update URL Validation"
status: completed
priority: P3
effort: "30m"
dependencies: []
---

# Phase 2: Update URL Validation

## Overview
Make `update.Check` validate `ReleaseURL` against `releaseURLPrefix` on the forced/live path, so `version --check-update` (which bypasses cache load) prints only repo-trusted URLs — matching the guard `cacheLoad` already applies on read.

## Requirements
- **Functional:** On the live-fetch path, if `rel.HTMLURL` does not start with `releaseURLPrefix`, do not store/print it (drop to empty string) — symmetric with `cache.go:79`.
- **Non-functional:** No behavior change for valid GitHub URLs (the normal case). No new deps.

## Architecture
- `internal/update/cache.go:25` defines `const releaseURLPrefix = "https://github.com/bavanchun/Typeburn/"`.
- `cacheLoad` (`cache.go:79`) already drops a non-prefixed `ReleaseURL` on read.
- `Check` (`check.go:50`) assigns `ReleaseURL: rel.HTMLURL` with no guard; `version --check-update` uses `force=true` → skips cache load → prints unvalidated URL (`cmd_version.go:157`).
- **Fix:** in `Check`, before assigning, validate: `url := rel.HTMLURL; if !strings.HasPrefix(url, releaseURLPrefix) { url = "" }`. `strings` is already imported.

## Related Code Files
- Modify: `internal/update/check.go` (guard `ReleaseURL` assignment)
- Read for context: `internal/update/cache.go`, `internal/cli/cmd_version.go`, `internal/update/check_test.go`
- Modify (tests): `internal/update/check_test.go` (add forced-path case with a hostile HTMLURL → expect empty ReleaseURL)

## Implementation Steps
1. Add a prefix-check in `Check` that zeroes `ReleaseURL` when `rel.HTMLURL` lacks `releaseURLPrefix`.
2. Add a test: `Check(force=true)` with a fetched release whose `HTMLURL` is e.g. `https://evil.example/x` → `result.ReleaseURL == ""`, and a valid URL passes through unchanged.
3. `go test ./internal/update/ -race -count=1`; full `-race`; gofmt; vet.

## Success Criteria
- [ ] Non-prefixed `ReleaseURL` is dropped on the forced path (test proves it).
- [ ] Valid GitHub release URLs unchanged.
- [ ] `-race` green; gofmt empty; vet clean.

## Risk Assessment
- **Very low.** Single guard mirroring an existing constant. Real-world risk was low (URL comes from GitHub API for this repo); this is consistency hardening, not a live exploit fix.
