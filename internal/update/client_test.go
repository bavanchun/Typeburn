package update

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestFetchLatest_Happy(t *testing.T) {
	rel := Release{TagName: "v2.1.0", HTMLURL: "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify required headers
		if !strings.HasPrefix(r.Header.Get("User-Agent"), "Typeburn/") {
			t.Errorf("missing Typeburn User-Agent, got %q", r.Header.Get("User-Agent"))
		}
		if r.Header.Get("Accept") != "application/vnd.github+json" {
			t.Errorf("missing Accept header")
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	}))
	defer srv.Close()

	got, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.TagName != "v2.1.0" {
		t.Errorf("TagName: want v2.1.0, got %q", got.TagName)
	}
}

func TestFetchLatest_403(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err != ErrRateLimit {
		t.Errorf("want ErrRateLimit, got %v", err)
	}
}

func TestFetchLatest_429(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()
	_, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err != ErrRateLimit {
		t.Errorf("want ErrRateLimit, got %v", err)
	}
}

func TestFetchLatest_404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	_, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err == nil {
		t.Error("expected error for 404")
	}
}

func TestFetchLatest_MalformedJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{not valid json`))
	}))
	defer srv.Close()
	_, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestFetchLatest_SlowResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	_, err := fetchLatestFrom(ctx, "v2.0.0", srv.URL)
	if err == nil {
		t.Error("expected timeout error for slow response")
	}
}

func TestFetchLatest_BodyCap(t *testing.T) {
	// Send 200KB body; FetchLatest should only read 64KB.
	large := make([]byte, 200*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(large)
	}))
	defer srv.Close()
	// Will fail to decode (not JSON), but should not hang or OOM
	_, err := fetchLatestFrom(context.Background(), "v2.0.0", srv.URL)
	if err == nil {
		t.Error("expected decode error for non-JSON body")
	}
}

// fetchLatestFrom is a test seam that overrides the target URL.
func fetchLatestFrom(ctx context.Context, currentVer, url string) (Release, error) {
	orig := fetchURL
	fetchURL = url
	defer func() { fetchURL = orig }()
	return FetchLatest(ctx, currentVer)
}
