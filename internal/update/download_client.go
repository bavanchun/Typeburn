package update

import (
	"fmt"
	"net/http"
	"time"
)

// maxRedirects bounds a redirect chain for asset downloads.
const maxRedirects = 10

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
