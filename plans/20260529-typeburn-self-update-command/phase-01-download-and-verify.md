---
phase: 1
title: Download and Verify
status: completed
priority: P1
effort: 0.5d
dependencies: []
---

# Phase 1: Download and Verify

## Overview

Construct the deterministic release-asset URL, download the archive + the
release `checksums.txt` (HTTPS, official-repo only, size-capped, with the one
GitHub asset redirect allowed), and verify the archive's sha256 against its
`checksums.txt` entry. No extraction yet — this phase ends with verified bytes
on disk (a temp file) or a hard abort.

## Requirements

- Functional:
  - `assetName(version, goos, goarch) string` → `typeburn_<ver-no-v>_<os>_<arch>.{tar.gz|zip}`
    (windows → `.zip`; leading `v` stripped to match GoReleaser).
  - `assetURL` / `checksumsURL` built off the official release base
    (`https://github.com/bavanchun/Typeburn/releases/download/<tag>/...`),
    guarded by the existing `releaseURLPrefix` constant.
  - Download with a bounded `http.Client`: total timeout, size cap (e.g. 50 MB),
    reject empty/truncated bodies. FOLLOW the asset redirect, but only to a
    *verified* github-owned asset-CDN host allowlist (see Architecture — the host
    is confirmed empirically, not hardcoded to one guess); reject any other host.
  - `parseChecksums(data []byte) map[string]string` — parse `<sha256>  <name>`
    lines into name→hash.
  - `verifySHA256(path, wantHex string) error` — stream-hash the file, constant
    nothing-fancy compare, error on mismatch.
- Non-functional: stdlib only; no UI deps; each file <200 LOC.

## Architecture

New file `internal/update/download.go` (URL build + bounded download + redirect
policy) and `internal/update/verify.go` (checksums parse + sha256). Borrows the
bounded-client *timeout/transport* shape from `client.go` (`FetchLatest`), but
NOT its redirect handling.

**Redirect policy — corrected.** `FetchLatest` does NOT "block" redirects: it
returns `http.ErrUseLastResponse` (`client.go:50`), which stops *following* and
hands back the raw 3xx. That is correct for the API endpoint (200 on the happy
path) but WRONG for asset downloads: GitHub release-asset URLs return a 302 to a
CDN host, so a verbatim copy downloads zero bytes. We must *follow* the redirect
— but only to a GitHub-owned asset host. The exact host is NOT assumed: it is
verified empirically against this repo's real release (it has historically been
`objects.githubusercontent.com`, but GitHub also uses
`release-assets.githubusercontent.com` / regional S3). `CheckRedirect` allowlists
the verified github-owned host set and default-denies everything else.

Data flow: live forced `Check` → raw `rel.TagName` (authoritative, keeps the
leading `v`) → `assetURL(rawTag)` + `checksumsURL(rawTag)` → download both to
temp → `parseChecksums` → `verifySHA256(archiveTmp, hashes[assetName(strippedVer)])`
→ return temp path or error.

## Related Code Files

- Create: `internal/update/download.go`, `internal/update/download_test.go`
- Create: `internal/update/verify.go`, `internal/update/verify_test.go`
- Read for context: `internal/update/client.go` (bounded-client + setFetchURL
  test seam), `internal/update/check.go` (`releaseURLPrefix`)

## Implementation Steps (TDD — tests first)

1. **Write `verify_test.go`:** table tests for `parseChecksums` (valid lines,
   blank lines, CRLF, missing entry) and `verifySHA256` (match, mismatch,
   missing file). Use known fixture bytes + precomputed hash.
2. Implement `verify.go` to green.
3. **Write `download_test.go`:** spin an `httptest.Server` (override base via a
   `setDownloadBase`-style seam mirroring `setFetchURL`) serving a fake archive +
   `checksums.txt`. Assert: correct asset name per os/arch; happy-path download
   returns verified temp path; size-cap exceeded → error; non-official host →
   error; one allowed redirect to the CDN host → success; second/other-host
   redirect → error; empty body → error.
4. Implement `download.go` to green: `assetName`, URL builders, bounded client
   with the scoped `CheckRedirect`, download-to-temp, wire `verifySHA256`.
5. `gofmt`, `go vet`, `go test ./internal/update/ -race -count=1`.

## Success Criteria

- [ ] `assetName` matches GoReleaser archive names exactly for all 6 os/arch combos.
- [ ] Download rejects non-official hosts and >cap bodies; allows only the GitHub
      asset-CDN redirect.
- [ ] sha256 mismatch returns an error and never yields a temp path to the caller.
- [ ] All new tests pass under `-race`; no UI deps; files <200 LOC.

## Risk Assessment

- **Redirect SSRF:** `CheckRedirect` allowlists the *verified* github-owned asset
  host(s); default-deny all others. Mitigation: explicit host-allowlist test +
  an e2e test (build-tagged / opt-in) that bytes actually transfer from the real
  redirect — unit httptest alone can pass while prod 404s.
- **Size cap too low/high:** archives are ~2-5 MB; 50 MB cap is generous but
  bounded. Document the constant.
- **Checksums format drift:** GoReleaser emits two-space `<hash>  <file>`; parse
  defensively (split on whitespace, take first+last fields).
- **Trust anchor (honest):** integrity = HTTPS/TLS to a github-owned host +
  sha256. `releaseURLPrefix` and `checksums.txt` are NOT independent anchors
  (same origin, unsigned). This defends a *corrupted/MITM'd download under TLS*,
  not a *compromised release*. Signing stays out of scope (user-confirmed).

## Red Team Adjustments (applied 2026-05-29)

These supersede/augment the body above:

1. **[Critical] Redirect (F1+A2):** see corrected Architecture. Empirically verify
   the real asset-redirect host before locking the allowlist; e2e byte-transfer test.
2. **[High] checksums trust anchor (F5/S2):** success criteria reworded below to
   "integrity under trusted TLS" — do NOT claim MITM/compromise defense. Fetch
   `checksums.txt` over the same TLS to a github-owned host; no extra anchor implied.
3. **[High] Tag eligibility (F7/S5):** before building any URL, re-validate the
   live tag with `validSemverRe` (`cache.go:24`) AND apply install.sh's
   `-rc/-test/-alpha/-beta/-pre/-dev/-snapshot` string guard (`install.sh:54-62`);
   require strict `update.Compare(current, tag) < 0` (`compare.go`). Reject otherwise.
4. **[High] Live-tag only (F8/S6):** the download tag MUST come from the live
   `Check(force=true)` Result, never the on-disk cache (`check.go:41,63` still
   `cacheSave`s — treat cache as untrusted for tag derivation).
5. **[High] v-prefix source of truth (F3):** `rel.TagName` (raw, with `v`) is
   authoritative for the URL path; `assetName` strips the `v` to match GoReleaser
   (`.goreleaser.yaml:55`). Test both against real release asset names.
6. **Tag URL-escape:** `url.PathEscape` the tag segment before interpolation.
