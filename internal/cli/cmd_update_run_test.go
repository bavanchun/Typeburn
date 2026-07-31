package cli

import (
	"bytes"
	"strings"
	"sync"
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

// Anything that is not a real terminal — a test buffer, a pipe, a redirect —
// must take the plain path.
func TestAnimatable_RejectsNonTerminalWriters(t *testing.T) {
	if animatable(&bytes.Buffer{}) {
		t.Error("a bytes.Buffer must not be treated as animatable")
	}
}

// The Apply goroutine writes progress while the render loop samples it; this
// runs under -race to prove the hand-off is sound.
func TestTracker_ConcurrentSetAndGet(t *testing.T) {
	var trk tracker
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := range 1000 {
			trk.set(update.Progress{Stage: update.StageDownloading, Done: int64(i), Total: 1000})
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			_ = trk.get()
		}
	}()
	wg.Wait()

	if got := trk.get(); got.Done != 999 {
		t.Errorf("final Done = %d, want 999", got.Done)
	}
}

// A zero tracker must read as a valid starting state, since the renderer polls
// before the download has reported anything.
func TestTracker_ZeroValueIsFirstStage(t *testing.T) {
	var trk tracker
	if got := trk.get(); got.Stage != update.StageChecksums {
		t.Errorf("zero tracker stage = %v, want StageChecksums", got.Stage)
	}
}

// The plain path is the contract for pipes, redirects, CI, and any terminal too
// narrow for the frame. A whole-string comparison — rather than substring
// checks — is what actually pins it, so an accidental extra line or reordering
// during UI work cannot slip through.
func TestUpdate_PlainPathExactOutput(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)
	withExecPath(t)

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--yes"); err != nil {
		t.Fatalf("update --yes: %v", err)
	}

	want := "Release notes: https://github.com/bavanchun/Typeburn/releases/tag/v2.3.0\n" +
		"updating dev → v2.3.0 ...\n" +
		"  checksums...\n" +
		"  downloading...\n" +
		"  verifying...\n" +
		"  installing...\n" +
		"updated dev → v2.3.0. restart typeburn to use the new version.\n"
	if got := out.String(); got != want {
		t.Errorf("plain output =\n%q\nwant\n%q", got, want)
	}
}
