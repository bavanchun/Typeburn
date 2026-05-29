package update

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestAssetName(t *testing.T) {
	cases := []struct{ version, goos, goarch, want string }{
		{"v2.2.0", "linux", "amd64", "typeburn_2.2.0_linux_amd64.tar.gz"},
		{"2.2.0", "linux", "arm64", "typeburn_2.2.0_linux_arm64.tar.gz"},
		{"v2.2.0", "darwin", "amd64", "typeburn_2.2.0_darwin_amd64.tar.gz"},
		{"v2.2.0", "darwin", "arm64", "typeburn_2.2.0_darwin_arm64.tar.gz"},
		{"v2.2.0", "windows", "amd64", "typeburn_2.2.0_windows_amd64.zip"},
		{"v2.2.0", "windows", "arm64", "typeburn_2.2.0_windows_arm64.zip"},
	}
	for _, c := range cases {
		if got := assetName(c.version, c.goos, c.goarch); got != c.want {
			t.Errorf("assetName(%q,%q,%q) = %q, want %q", c.version, c.goos, c.goarch, got, c.want)
		}
	}
}

// fakeRelease serves a checksums.txt + one archive for the given asset bytes.
func fakeRelease(t *testing.T, tag, asset string, archive []byte) *httptest.Server {
	t.Helper()
	checksums := sha256Hex(archive) + "  " + asset + "\n"
	mux := http.NewServeMux()
	srv := httptest.NewServer(mux)
	mux.HandleFunc("/"+tag+"/checksums.txt", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(checksums))
	})
	mux.HandleFunc("/"+tag+"/"+asset, func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(archive)
	})
	t.Cleanup(srv.Close)
	return srv
}

func TestDownloadVerified_HappyPath(t *testing.T) {
	archive := []byte("fake archive content")
	asset := assetName("2.2.0", "linux", "amd64")
	srv := fakeRelease(t, "v2.2.0", asset, archive)
	old := getDownloadBase()
	setDownloadBase(srv.URL)
	defer setDownloadBase(old)

	dir := t.TempDir()
	got, err := downloadVerified(context.Background(), "v2.2.0", "linux", "amd64", dir, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(got)
	if string(data) != string(archive) {
		t.Errorf("archive content mismatch")
	}
	// checksums.txt temp must be cleaned up.
	if _, err := os.Stat(strings.Replace(got, asset, "checksums.txt", 1)); !os.IsNotExist(err) {
		t.Errorf("checksums temp not cleaned up")
	}
}

func TestDownloadVerified_ChecksumMismatch(t *testing.T) {
	asset := assetName("2.2.0", "linux", "amd64")
	// checksums.txt lists the hash of "real", but the asset body is "TAMPERED".
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "checksums.txt") {
			_, _ = w.Write([]byte(sha256Hex([]byte("real")) + "  " + asset + "\n"))
			return
		}
		_, _ = w.Write([]byte("TAMPERED"))
	}))
	defer srv.Close()
	old := getDownloadBase()
	setDownloadBase(srv.URL)
	defer setDownloadBase(old)

	dir := t.TempDir()
	got, err := downloadVerified(context.Background(), "v2.2.0", "linux", "amd64", dir, nil)
	if err == nil {
		t.Error("expected checksum mismatch error, got nil")
	}
	if got != "" {
		t.Errorf("expected empty path on mismatch, got %q", got)
	}
}

func TestDownloadTo_SizeCap(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 100))
	}))
	defer srv.Close()
	dir := t.TempDir()
	err := downloadTo(context.Background(), newDownloadClient(), srv.URL+"/big", dir+"/out", 10)
	if err == nil || !strings.Contains(err.Error(), "cap") {
		t.Errorf("expected size-cap error, got %v", err)
	}
	if _, statErr := os.Stat(dir + "/out"); !os.IsNotExist(statErr) {
		t.Error("oversized download not cleaned up")
	}
}

func TestDownloadTo_EmptyBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()
	dir := t.TempDir()
	if err := downloadTo(context.Background(), newDownloadClient(), srv.URL, dir+"/out", 1000); err == nil {
		t.Error("expected empty-body error, got nil")
	}
}

func TestDownloadTo_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	dir := t.TempDir()
	if err := downloadTo(context.Background(), newDownloadClient(), srv.URL, dir+"/out", 1000); err == nil {
		t.Error("expected non-200 error, got nil")
	}
}

func TestDownloadClient_RefusesNonGithubRedirect(t *testing.T) {
	// Origin server redirects to an arbitrary external host → must be refused.
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("evil"))
	}))
	defer evil.Close()
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL, http.StatusFound)
	}))
	defer origin.Close()

	dir := t.TempDir()
	err := downloadTo(context.Background(), newDownloadClient(), origin.URL+"/asset", dir+"/out", 1000)
	if err == nil {
		t.Error("expected refusal of redirect to non-github host, got nil")
	}
}

func TestDownloadClient_AllowsSameHostRedirect(t *testing.T) {
	// A server that 302s to itself (same host) must be followed.
	var srv *httptest.Server
	srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/asset" {
			http.Redirect(w, r, srv.URL+"/real", http.StatusFound)
			return
		}
		_, _ = w.Write([]byte("real bytes"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	if err := downloadTo(context.Background(), newDownloadClient(), srv.URL+"/asset", dir+"/out", 1000); err != nil {
		t.Errorf("same-host redirect should succeed, got %v", err)
	}
}
