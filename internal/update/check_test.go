package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func stubServer(t *testing.T, rel Release) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	}))
}

func TestCheck_DevSkip(t *testing.T) {
	for _, ver := range []string{"", "dev", "DEV ", "unknown", "v0.0.0-rc.test", "v0.0.0-123"} {
		r, err := Check(context.Background(), ver, false)
		if r != nil || err != nil {
			t.Errorf("Check(%q) = (%v, %v), want (nil, nil)", ver, r, err)
		}
	}
}

func TestCheck_UpgradeAvailable(t *testing.T) {
	defer withTempCache(t)()
	srv := stubServer(t, Release{TagName: "v2.1.0", HTMLURL: "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0"})
	defer srv.Close()
	origURL := getFetchURL()
	setFetchURL(srv.URL)
	defer setFetchURL(origURL)

	r, err := Check(context.Background(), "v2.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if !r.UpgradeAvailable {
		t.Error("UpgradeAvailable: want true")
	}
	if r.Latest != "v2.1.0" {
		t.Errorf("Latest: want v2.1.0, got %q", r.Latest)
	}
	if r.SchemaVersion != cacheSchemaVersion {
		t.Errorf("SchemaVersion: want %d, got %d", cacheSchemaVersion, r.SchemaVersion)
	}
}

func TestCheck_UpToDate(t *testing.T) {
	defer withTempCache(t)()
	srv := stubServer(t, Release{TagName: "v2.0.0", HTMLURL: "https://github.com/bavanchun/Typeburn/releases/tag/v2.0.0"})
	defer srv.Close()
	origURL := getFetchURL()
	setFetchURL(srv.URL)
	defer setFetchURL(origURL)

	r, err := Check(context.Background(), "v2.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil {
		t.Fatal("expected non-nil result")
	}
	if r.UpgradeAvailable {
		t.Error("UpgradeAvailable: want false for up-to-date")
	}
}

func TestCheck_PrereleaseIgnored(t *testing.T) {
	defer withTempCache(t)()
	srv := stubServer(t, Release{TagName: "v2.1.0-rc.1", Prerelease: true})
	defer srv.Close()
	origURL := getFetchURL()
	setFetchURL(srv.URL)
	defer setFetchURL(origURL)

	r, err := Check(context.Background(), "v2.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.UpgradeAvailable {
		t.Error("UpgradeAvailable: want false when latest is prerelease")
	}
}

func TestCheck_CacheHit(t *testing.T) {
	defer withTempCache(t)()
	// Pre-populate a fresh cache.
	cached := &Result{
		SchemaVersion:    cacheSchemaVersion,
		Current:          "v2.0.0",
		Latest:           "v2.1.0",
		UpgradeAvailable: true,
		ReleaseURL:       "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
		CheckedAt:        time.Now().UTC(),
	}
	if err := cacheSave(cached); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}

	// No server needed — cache should be hit.
	r, err := Check(context.Background(), "v2.0.0", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r == nil || !r.UpgradeAvailable {
		t.Error("expected cached upgrade result")
	}
}

func TestCheck_PrereleaseCached(t *testing.T) {
	defer withTempCache(t)()
	srv := stubServer(t, Release{TagName: "v2.1.0-rc.1", Prerelease: true})
	origURL := getFetchURL()
	setFetchURL(srv.URL)
	defer setFetchURL(origURL)

	// force=true → fetch + (now) cache the synthetic no-upgrade result.
	r1, err := Check(context.Background(), "v2.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r1.UpgradeAvailable {
		t.Error("prerelease must not be reported as an upgrade")
	}

	// Close the server now — the second call must succeed from cache without hitting the network.
	srv.Close()

	r2, err := Check(context.Background(), "v2.0.0", false)
	if err != nil {
		t.Fatalf("non-forced call after prerelease cache write must not error: %v", err)
	}
	if r2 == nil || r2.UpgradeAvailable {
		t.Errorf("expected cached no-upgrade result, got %#v", r2)
	}
}

func TestCheck_ForceBypassesCache(t *testing.T) {
	defer withTempCache(t)()
	// Pre-populate a fresh cache saying up-to-date.
	cached := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "v2.0.0",
		CheckedAt:     time.Now().UTC(),
	}
	if err := cacheSave(cached); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}

	// Server says there IS an upgrade — force=true should bypass cache.
	srv := stubServer(t, Release{TagName: "v2.1.0", HTMLURL: "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0"})
	defer srv.Close()
	origURL := getFetchURL()
	setFetchURL(srv.URL)
	defer setFetchURL(origURL)

	r, err := Check(context.Background(), "v2.0.0", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !r.UpgradeAvailable {
		t.Error("expected force=true to bypass cache and return upgrade")
	}
}
