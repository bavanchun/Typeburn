package update

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// body is large enough that io.Copy issues many writes, so the throttle is
// genuinely exercised rather than trivially satisfied.
var progressBody = bytes.Repeat([]byte("t"), 512<<10) // 512 KiB

func TestDownloadTo_ReportsContentLengthAsTotal(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Length", "524288")
		_, _ = w.Write(progressBody)
	}))
	defer srv.Close()

	var lastDone, lastTotal int64
	var calls int
	dest := filepath.Join(t.TempDir(), "out")
	err := downloadTo(context.Background(), newDownloadClient(), srv.URL, dest, archiveSizeCap,
		func(done, total int64) {
			calls++
			lastDone, lastTotal = done, total
		})
	if err != nil {
		t.Fatalf("downloadTo: %v", err)
	}

	if lastTotal != int64(len(progressBody)) {
		t.Errorf("total = %d, want %d", lastTotal, len(progressBody))
	}
	if lastDone != int64(len(progressBody)) {
		t.Errorf("final done = %d, want %d", lastDone, len(progressBody))
	}
	// The 50ms throttle plus the final flush must keep this far below the
	// number of writes io.Copy performs (512 KiB / 32 KiB = 16+ writes, and
	// far more for a real multi-megabyte archive).
	if calls > 120 {
		t.Errorf("callbacks = %d, want <= 120", calls)
	}
}

// A chunked response carries no Content-Length; the front-end must be able to
// tell "unknown" apart from "zero bytes expected".
func TestDownloadTo_UnknownTotalWhenNoContentLength(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Transfer-Encoding", "chunked")
		_, _ = w.Write([]byte("payload"))
	}))
	defer srv.Close()

	var total int64 = -1
	dest := filepath.Join(t.TempDir(), "out")
	err := downloadTo(context.Background(), newDownloadClient(), srv.URL, dest, archiveSizeCap,
		func(_, tot int64) { total = tot })
	if err != nil {
		t.Fatalf("downloadTo: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0 (unknown)", total)
	}
}

// A nil reporter must remain legal at the download layer — checksums.txt uses it.
func TestDownloadTo_NilReporter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("checksums"))
	}))
	defer srv.Close()

	dest := filepath.Join(t.TempDir(), "out")
	if err := downloadTo(context.Background(), newDownloadClient(), srv.URL, dest, checksumsSizeCap, nil); err != nil {
		t.Fatalf("downloadTo with nil reporter: %v", err)
	}
}
