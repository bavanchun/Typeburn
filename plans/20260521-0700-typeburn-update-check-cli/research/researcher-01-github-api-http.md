# Research: GitHub Releases API & Stdlib HTTP Client Patterns for Offline-Tolerant Update Checker

**Date:** 2026-05-21  
**Work Context:** `/Users/vchun/Codes/My-projects/Typeburn`

## 1. GitHub Releases API Response Shape

### Endpoint
```
GET https://api.github.com/repos/{owner}/{repo}/releases/latest
```

### Minimal Fields Needed (extract only these)
Per [GitHub REST Releases API](https://docs.github.com/en/rest/releases/releases):

- **`tag_name`** (string) — semantic version identifier (`v2.1.0`)
- **`html_url`** (string, URI) — link to release page on GitHub
- **`name`** (string or null) — human-readable release name
- **`draft`** (boolean) — if true, exclude from latest
- **`prerelease`** (boolean) — if true, exclude unless already on pre-release
- **`published_at`** (string or null, ISO 8601) — release timestamp

Full response is 20-50+ fields (assets, reactions, author, etc.); cap body read at 64KB to prevent parse bloat from hostile or malformed responses.

## 2. Rate Limiting Reality

### Unauthenticated Limits
**60 requests per hour per IP** (GitHub REST API rate limits, [verified](https://docs.github.com/en/rest/overview/rate-limits-for-the-rest-api)).

### Rate-Limit Headers (in every response)
- `X-RateLimit-Limit` — max requests per hour (60 for unauth)
- `X-RateLimit-Remaining` — requests left in window
- `X-RateLimit-Reset` — UTC epoch seconds when window resets
- `X-RateLimit-Used` — requests consumed this window
- `X-RateLimit-Resource` — which bucket this counted against (e.g., "core")

### Detection: 403 + Rate-Limited
**Response code:** 403 or 429  
**Header check:** `X-RateLimit-Remaining: 0`  
**Silent degradation:** Do not parse body on 403; treat as "check failed, use stale version."

## 3. User-Agent Header

### Requirement
GitHub API **rejects all requests without a valid User-Agent header** (returns 403). Per [GitHub API resource overview](https://docs.github.com/en/rest/overview/resources-in-the-rest-api):

> "All API requests must include a valid User-Agent header"

### Recommended Format
```
User-Agent: Typeburn/<version> (+https://github.com/bavanchun/Typeburn)
```

Example:
```
User-Agent: Typeburn/2.1.0 (+https://github.com/bavanchun/Typeburn)
User-Agent: Typeburn/dev (+https://github.com/bavanchun/Typeburn)  [for bare go install]
```

This format is compliant with the spec ("Awesome-Octocat-App" pattern) and identifies the app + link.

## 4. Idiomatic Stdlib HTTP Client: Timeout Strategy

### Two Mechanisms

1. **`http.Client{Timeout: 1500ms}`** — blanket timeout on entire request (connect + redirect + read body). Timer continues running after `Do()` returns. Per [Go net/http docs](https://pkg.go.dev/net/http#Client.Timeout).

2. **`context.WithTimeout()` + `http.NewRequestWithContext()`** — request-level deadline, cancellable, integrates with Go cancellation patterns.

### Recommendation for This Use Case
**Use BOTH (layered defense):**

```go
// Client-level safety net (catch all requests)
client := &http.Client{
    Timeout: 1500 * time.Millisecond,
    Transport: &http.Transport{
        Proxy: http.ProxyFromEnvironment,  // Honor HTTPS_PROXY, NO_PROXY
    },
}

// Request-level deadline (explicit control)
ctx, cancel := context.WithTimeout(context.Background(), 1500*time.Millisecond)
defer cancel()

req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
req.Header.Set("User-Agent", "Typeburn/"+version+" (+https://github.com/bavanchun/Typeburn)")

resp, err := client.Do(req)
```

**Why both:**
- `Client.Timeout` is a blanket safety net; catches runaway reads.
- `context.WithTimeout` is explicit and composable; signals intent clearly.
- Layering ensures: if context deadline fires, client timeout still backs it up.

## 5. Response Body Size Cap

### Pattern: `io.LimitReader`
Per [Go io.LimitReader docs](https://pkg.go.dev/io#LimitReader):

```go
resp, _ := client.Do(req)
defer resp.Body.Close()

// Cap at 64 KB; releases JSON is ~5-15 KB
limitedBody := io.LimitReader(resp.Body, 64*1024)

// Parse from limited reader
var release struct {
    TagName   string `json:"tag_name"`
    HtmlUrl   string `json:"html_url"`
    Draft     bool   `json:"draft"`
    Prerelease bool  `json:"prerelease"`
}
err := json.NewDecoder(limitedBody).Decode(&release)
```

**Cap choice: 64 KB** — safe margin above typical release JSON (~5-15 KB), protects against malicious/runaway responses, negligible memory cost.

## 6. TLS, Proxy, & Accept Headers

### Environment Proxy Support
`http.DefaultTransport` does **NOT** automatically honor `HTTPS_PROXY`, `NO_PROXY` env vars. Must explicitly configure:

```go
Transport: &http.Transport{
    Proxy: http.ProxyFromEnvironment,  // Enables env var proxy support
}
```

This respects `HTTPS_PROXY`, `NO_PROXY` (case-insensitive), and skips proxy for localhost.

### Accept Header
GitHub Releases API does **not require** `Accept: application/vnd.github+json`. Default content negotiation works; optional header does not change response format for `/releases/latest`. Omit unless explicitly needed.

### X-GitHub-Api-Version Header
**Not required.** Requests without this header default to version `2022-11-28` (stable). Current latest is `2026-03-10`. For update checking, the stable default is sufficient; no need to pin.

## 7. httptest.Server Test Pattern

### Mock API for CI Testing

```go
import (
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
)

func TestCheckUpdate(t *testing.T) {
    // Mock server returns a release response
    ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Verify request headers
        if ua := r.Header.Get("User-Agent"); ua == "" {
            w.WriteHeader(http.StatusForbidden)
            return
        }

        // Return controlled response
        w.Header().Set("Content-Type", "application/json")
        w.Header().Set("X-RateLimit-Remaining", "59")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "tag_name":   "v2.2.0",
            "html_url":   "https://github.com/bavanchun/Typeburn/releases/tag/v2.2.0",
            "draft":      false,
            "prerelease": false,
        })
    }))
    defer ts.Close()

    // Test against mock (replace real URL with ts.URL)
    release, err := CheckLatestRelease(ts.URL + "/repos/bavanchun/Typeburn/releases/latest")
    if err != nil {
        t.Fatal(err)
    }
    if release.TagName != "v2.2.0" {
        t.Errorf("got %s, want v2.2.0", release.TagName)
    }
}
```

**Key points:**
- `httptest.NewServer(handler)` starts immediately; use `ts.URL` for requests.
- Handler receives `*http.Request` for inspection (headers, method, path).
- Use `httptest.NewRecorder()` for unit tests without full server (faster, no network).
- Always `defer ts.Close()`.

## 8. Edge Cases Worth Handling

### 8.1 Repo Has Zero Published Releases Yet
**Response:** 404 with `{"message": "Not Found"}`  
**Handle:** Treat as "no update available" — silent, no error logged.

### 8.2 Latest Is Pre-Release Only (No Stable)
**Scenario:** `/releases/latest` returns `"prerelease": true`, and no stable release exists.  
**Handle:** 
- If current version is also pre-release (git-describe output like `v2.0.0-7-gabc123`), compare.
- If current is stable, ignore pre-release; no update suggested.

### 8.3 DNS Failure, Connection Refused, Slow TLS Handshake
**All are `context` timeout violations** — hits the 1500ms deadline.  
**Handle:** `context.Err()` or `net.Error{Timeout: true}` — catch and return "check failed."

### 8.4 Version Comparison: `v2.0.0-7-gabc123` (Git-Describe)
**Source:** `internal/version` injects `Version` from ldflags (Makefile) or `debug.ReadBuildInfo()` fallback.  
**Comparator:** Semantic versioning libraries (e.g., `go-semver`) do NOT handle `-7-gabc123` — that's Git output, not SemVer.  
**Design choice:** Store *latest stable* as SemVer; compare against current *major.minor.patch only* (strip git suffix if present).  
**Example:**
- Installed: `v2.0.0-7-gabc123` → normalize to `v2.0.0` for comparison.
- Latest: `v2.1.0` → update available.
- Latest: `v2.0.0` → no update.

This avoids pulling in semver parsers and keeps the checker simple.

---

## Unresolved Questions

1. **Comparator library:** Should we use `go-semver` (charmbracelet-compatible) or implement a naive string-based major.minor.patch comparison? (Defer to planner.)
2. **Version format in binary:** Confirm `internal/version` always provides clean SemVer or git-describe suffix that needs stripping. (Inspect at implementation time.)
3. **Cache/persistence:** Should the checker cache the "last checked" time (e.g., 24h) to avoid API spam on repeated TUI launches? (Defer to feature scope.)

---

## Sources

- [GitHub REST Releases API](https://docs.github.com/en/rest/releases/releases)
- [GitHub REST Rate Limiting](https://docs.github.com/en/rest/overview/rate-limits-for-the-rest-api)
- [GitHub REST API Resources Overview](https://docs.github.com/en/rest/overview/resources-in-the-rest-api)
- [Go net/http Client Timeout](https://pkg.go.dev/net/http#Client.Timeout)
- [Go context.WithTimeout](https://pkg.go.dev/context#WithTimeout)
- [Go io.LimitReader](https://pkg.go.dev/io#LimitReader)
- [Go http.Transport Proxy](https://pkg.go.dev/net/http#Transport.Proxy)
- [Go httptest.Server](https://pkg.go.dev/net/http/httptest#Server)
- [Go net/http ProxyFromEnvironment](https://pkg.go.dev/net/http#ProxyFromEnvironment)
