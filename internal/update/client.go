package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

// fetchURL is the GitHub Releases API endpoint. It is a var so tests can
// override it with an httptest.Server URL without touching real network.
var (
	fetchURLMu sync.Mutex
	fetchURL   = "https://api.github.com/repos/bavanchun/Typeburn/releases/latest"
)

func getFetchURL() string {
	fetchURLMu.Lock()
	defer fetchURLMu.Unlock()
	return fetchURL
}

func setFetchURL(u string) {
	fetchURLMu.Lock()
	defer fetchURLMu.Unlock()
	fetchURL = u
}

const bodyLimit = 64 * 1024

// ErrUpstream indicates a non-2xx response from the GitHub API.
var ErrUpstream = errors.New("update: upstream error")

// ErrRateLimit indicates a 403 or 429 response (rate-limited or blocked).
var ErrRateLimit = errors.New("update: rate limited")

// FetchLatest queries the GitHub Releases API and returns the latest release.
// The HTTP client enforces a 1.5s total timeout with explicit dial and TLS
// sub-timeouts so degraded networks don't hang the caller.
func FetchLatest(ctx context.Context, currentVer string) (Release, error) {
	client := &http.Client{
		Timeout: 1500 * time.Millisecond,
		// Block redirects: /releases/latest returns 200 on the happy path.
		// Following an attacker-controlled 302 could leak UA/version or downgrade to HTTP.
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			DialContext:           (&net.Dialer{Timeout: 800 * time.Millisecond}).DialContext,
			TLSHandshakeTimeout:   800 * time.Millisecond,
			ResponseHeaderTimeout: 800 * time.Millisecond,
		},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, getFetchURL(), nil)
	if err != nil {
		return Release{}, fmt.Errorf("update: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Typeburn/"+currentVer+" (+https://github.com/bavanchun/Typeburn)")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, fmt.Errorf("update: http: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusForbidden, http.StatusTooManyRequests:
		return Release{}, ErrRateLimit
	default:
		return Release{}, fmt.Errorf("%w: status %d", ErrUpstream, resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(io.LimitReader(resp.Body, bodyLimit)).Decode(&rel); err != nil {
		return Release{}, fmt.Errorf("update: decode response: %w", err)
	}
	return rel, nil
}
