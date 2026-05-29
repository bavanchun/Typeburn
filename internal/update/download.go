package update

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// archiveSizeCap bounds an archive download. Release archives are ~2-5 MB;
	// the cap is generous but prevents an unbounded body from filling the disk.
	archiveSizeCap = 50 << 20 // 50 MiB
	// checksumsSizeCap bounds the checksums.txt download.
	checksumsSizeCap = 64 << 10 // 64 KiB
	maxRedirects     = 10
)

// githubAssetHosts are the GitHub-owned hosts a release-asset download may
// redirect to. A release URL on github.com 302s to one of these CDN hosts;
// any other redirect target is refused (SSRF guard). The set reflects the CDN
// hosts GitHub's release-asset 302s actually land on.
var githubAssetHosts = map[string]bool{
	"github.com":                           true,
	"objects.githubusercontent.com":        true,
	"release-assets.githubusercontent.com": true,
	"codeload.github.com":                  true,
}

// downloadBase is the GitHub release-download base. A var so tests can point it
// at an httptest server (mirrors the setFetchURL seam in client.go).
var (
	downloadBaseMu sync.Mutex
	downloadBase   = "https://github.com/bavanchun/Typeburn/releases/download"
)

func getDownloadBase() string {
	downloadBaseMu.Lock()
	defer downloadBaseMu.Unlock()
	return downloadBase
}

func setDownloadBase(b string) {
	downloadBaseMu.Lock()
	defer downloadBaseMu.Unlock()
	downloadBase = b
}

// assetExt returns the archive extension GoReleaser uses for goos.
func assetExt(goos string) string {
	if goos == "windows" {
		return "zip"
	}
	return "tar.gz"
}

// assetName builds the GoReleaser archive filename. version must be the
// v-stripped form: GoReleaser's {{ .Version }} drops the leading "v", so the
// asset name carries no "v" even though the git tag and binary banner do.
func assetName(version, goos, goarch string) string {
	v := strings.TrimPrefix(version, "v")
	return fmt.Sprintf("typeburn_%s_%s_%s.%s", v, goos, goarch, assetExt(goos))
}

// assetURL builds the download URL for one asset under a release tag. rawTag is
// the authoritative tag WITH its leading "v" (the URL path keeps it); the tag
// segment is path-escaped before interpolation.
func assetURL(rawTag, name string) string {
	return getDownloadBase() + "/" + url.PathEscape(rawTag) + "/" + name
}

// newDownloadClient builds the bounded HTTP client used for asset downloads.
// Unlike FetchLatest (which refuses to follow redirects), this client FOLLOWS
// redirects but only to a github-owned asset host — release assets always 302
// to a CDN, so a non-following client would download zero bytes.
func newDownloadClient() *http.Client {
	return &http.Client{
		Timeout: 90 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return fmt.Errorf("update: too many redirects")
			}
			// Refuse a TLS downgrade. checksums.txt is the integrity root and is
			// fetched with this same client, so following a redirect to cleartext
			// http would let a network attacker swap both archive and checksums.
			// Loopback is exempt so test servers (http on 127.0.0.1) work.
			if h := req.URL.Hostname(); req.URL.Scheme != "https" && h != "127.0.0.1" && h != "localhost" && h != "::1" {
				return fmt.Errorf("update: refusing non-https redirect to %q", req.URL.Hostname())
			}
			if githubAssetHosts[req.URL.Hostname()] {
				return nil
			}
			// Permit a redirect back to the exact same host:port (the origin of
			// the first request) so a test server may 302 to itself. Compare the
			// full Host (with port): a different port is a different endpoint.
			if len(via) > 0 && req.URL.Host == via[0].URL.Host {
				return nil
			}
			return fmt.Errorf("update: refusing redirect to non-github host %q", req.URL.Hostname())
		},
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			TLSHandshakeTimeout:   10 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

// downloadTo streams rawURL into dest (created O_EXCL, 0600), bounded by cap.
// Rejects non-200 status, empty bodies, and bodies exceeding cap; cleans up the
// partial file on any failure.
func downloadTo(ctx context.Context, client *http.Client, rawURL, dest string, sizeCap int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("update: build request: %w", err)
	}
	req.Header.Set("User-Agent", "Typeburn-updater (+https://github.com/bavanchun/Typeburn)")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("update: download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("update: download %s: status %d", rawURL, resp.StatusCode)
	}

	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("update: create download file: %w", err)
	}
	n, copyErr := io.Copy(f, io.LimitReader(resp.Body, sizeCap+1))
	closeErr := f.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("update: write download: %w", copyErr)
	}
	if n == 0 {
		_ = os.Remove(dest)
		return fmt.Errorf("update: empty download: %s", rawURL)
	}
	if n > sizeCap {
		_ = os.Remove(dest)
		return fmt.Errorf("update: download exceeds %d-byte cap", sizeCap)
	}
	return nil
}

// downloadVerified downloads the release archive for rawTag (authoritative tag,
// keeps the leading "v") into destDir, verifies it against the release
// checksums.txt, and returns the verified archive path. The asset name is
// derived from rawTag itself (assetName strips the leading "v"), so the
// downloaded asset always matches the release being installed — never the
// caller's current version. On any failure no archive path is returned and
// temporaries are removed.
func downloadVerified(ctx context.Context, rawTag, goos, goarch, destDir string) (string, error) {
	name := assetName(rawTag, goos, goarch)
	client := newDownloadClient()

	sumPath := filepath.Join(destDir, "checksums.txt")
	if err := downloadTo(ctx, client, assetURL(rawTag, "checksums.txt"), sumPath, checksumsSizeCap); err != nil {
		return "", err
	}
	defer os.Remove(sumPath)

	sumData, err := os.ReadFile(sumPath)
	if err != nil {
		return "", fmt.Errorf("update: read checksums: %w", err)
	}
	want := parseChecksums(sumData)[name]

	archivePath := filepath.Join(destDir, name)
	if err := downloadTo(ctx, client, assetURL(rawTag, name), archivePath, archiveSizeCap); err != nil {
		return "", err
	}
	if err := verifySHA256(archivePath, want); err != nil {
		_ = os.Remove(archivePath)
		return "", err
	}
	return archivePath, nil
}
