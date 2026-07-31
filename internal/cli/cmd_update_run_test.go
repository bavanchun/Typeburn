package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// The download stage reports once per throttled chunk. Without deduplication a
// real 4 MB archive would print a hundred identical lines.
func TestPlainReporter_DeduplicatesRepeatedStages(t *testing.T) {
	var out bytes.Buffer
	report := plainReporter(&out)

	report(update.Progress{Stage: update.StageChecksums})
	for i := range 200 {
		report(update.Progress{Stage: update.StageDownloading, Done: int64(i), Total: 200})
	}
	report(update.Progress{Stage: update.StageVerifying})
	report(update.Progress{Stage: update.StageInstalling})

	got := out.String()
	for _, stage := range []string{"checksums", "downloading", "verifying", "installing"} {
		if n := strings.Count(got, "  "+stage+"...\n"); n != 1 {
			t.Errorf("%q printed %d times, want 1\n---\n%s", stage, n, got)
		}
	}
}

// The reporter renders stages in run order, and the checksums fetch — real work
// that was previously invisible — is now surfaced ahead of the download.
func TestPlainReporter_StageOrder(t *testing.T) {
	var out bytes.Buffer
	report := plainReporter(&out)
	for _, s := range []update.Stage{
		update.StageChecksums, update.StageDownloading,
		update.StageVerifying, update.StageInstalling,
	} {
		report(update.Progress{Stage: s})
	}

	want := "  checksums...\n  downloading...\n  verifying...\n  installing...\n"
	if got := out.String(); got != want {
		t.Errorf("output =\n%q\nwant\n%q", got, want)
	}
}
