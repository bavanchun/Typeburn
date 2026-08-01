package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/cli/updateui"
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

// Cancelling must not simply abandon the update: Apply holds an O_EXCL lock in
// the install directory, and returning before its defers run leaves that lock
// behind, which makes every later `typeburn update` refuse to start.
func TestStopApply_WaitsForTheUpdateToUnwind(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	results := make(chan updateui.Result, 1)

	var cleanedUp atomic.Bool
	go func() {
		defer func() {
			cleanedUp.Store(true) // stands in for Apply's deferred lock release
			results <- updateui.Result{Err: ctx.Err()}
		}()
		<-ctx.Done()
	}()

	cause := errors.New("cancelled by user")
	_, err := stopApply(cancel, results, cause, applyDrainTimeout)
	if !errors.Is(err, cause) {
		t.Errorf("stopApply returned %v, want %v", err, cause)
	}
	if !cleanedUp.Load() {
		t.Error("stopApply returned before the update finished unwinding")
	}
}

// A wedged update must not hang the command forever, and the user needs to be
// told the lock may be stale.
func TestStopApply_TimesOutAndNamesTheLock(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	results := make(chan updateui.Result) // never delivers

	cause := errors.New("cancelled by user")
	_, err := stopApply(cancel, results, cause, 20*time.Millisecond)
	if !errors.Is(err, cause) {
		t.Errorf("timeout error lost its cause: %v", err)
	}
	if !errors.Is(err, errDrainTimeout) {
		t.Errorf("timeout error must be identifiable as a drain timeout: %v", err)
	}
	if !strings.Contains(err.Error(), ".typeburn-update.lock") {
		t.Errorf("timeout error should name the lock file, got: %v", err)
	}
}

// Cancelling is a request, not a guarantee: only the download is cancellable,
// so a stop arriving during verify/extract/rename lands after the binary has
// already been replaced. Reporting "nothing was installed" there would be a lie
// the user cannot detect without re-running.
func TestStopApply_ReportsAnUpdateThatFinishedAnyway(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	results := make(chan updateui.Result, 1)
	want := update.Outcome{From: "v2.5.1", To: "v2.6.0"}
	results <- updateui.Result{Outcome: want}

	outcome, err := stopApply(cancel, results, updateui.ErrCancelled, applyDrainTimeout)
	if !errors.Is(err, errStoppedTooLate) {
		t.Errorf("err = %v, want errStoppedTooLate", err)
	}
	if errors.Is(err, updateui.ErrCancelled) {
		t.Error("a completed update must not still report as cancelled")
	}
	if outcome != want {
		t.Errorf("outcome = %+v, want %+v", outcome, want)
	}
}

// A genuine failure must keep the interrupt as the reported cause rather than
// being reported as a completed install.
func TestStopApply_FailedUpdateKeepsItsCause(t *testing.T) {
	_, cancel := context.WithCancel(context.Background())
	results := make(chan updateui.Result, 1)
	results <- updateui.Result{Err: errors.New("download aborted")}

	outcome, err := stopApply(cancel, results, updateui.ErrCancelled, applyDrainTimeout)
	if !errors.Is(err, updateui.ErrCancelled) {
		t.Errorf("err = %v, want ErrCancelled", err)
	}
	if errors.Is(err, errStoppedTooLate) {
		t.Error("a failed update must not report as completed")
	}
	if (outcome != update.Outcome{}) {
		t.Errorf("outcome = %+v, want zero", outcome)
	}
}

// The interrupt cases are not mutually exclusive — a drain timeout wraps its
// cause, which on the cancel path is ErrCancelled — so the switch order in
// reportApplyResult is load-bearing, not cosmetic.
func TestReportApplyResult_MessageAndExitPerOutcome(t *testing.T) {
	done := update.Outcome{From: "v2.5.1", To: "v2.6.0"}
	timedOut := fmt.Errorf("%w: %w (remove a stale .typeburn-update.lock)", errDrainTimeout, updateui.ErrCancelled)

	tests := []struct {
		name     string
		outcome  update.Outcome
		err      error
		wantOut  string
		wantCode int
	}{
		{"success", done, nil, "updated v2.5.1 → v2.6.0.", ExitOK},
		{"stopped too late", done, errStoppedTooLate, "stopped too late — v2.5.1 → v2.6.0 was already installed.", ExitOK},
		{"cancelled in time", update.Outcome{}, updateui.ErrCancelled, "", ExitAbort},
		{"drain timed out", update.Outcome{}, timedOut, "", ExitIO},
		{"plain failure", update.Outcome{}, errors.New("boom"), "", ExitIO},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var out bytes.Buffer
			err := reportApplyResult(&out, tc.outcome, tc.err)

			if got := ExitCode(err); got != tc.wantCode {
				t.Errorf("exit code = %d, want %d (err: %v)", got, tc.wantCode, err)
			}
			if tc.wantOut == "" {
				if out.Len() != 0 {
					t.Errorf("expected no stdout, got %q", out.String())
				}
				return
			}
			if !strings.Contains(out.String(), tc.wantOut) {
				t.Errorf("stdout = %q, want it to contain %q", out.String(), tc.wantOut)
			}
		})
	}
}

// A timed-out drain means the update was still working — the case most likely
// to have installed something — so its stale-lock guidance must survive into
// the user-facing error rather than being replaced by "nothing was installed".
func TestReportApplyResult_TimeoutGuidanceSurvivesCancelMatch(t *testing.T) {
	timedOut := fmt.Errorf("%w: %w (remove a stale .typeburn-update.lock from the install directory)",
		errDrainTimeout, updateui.ErrCancelled)

	err := reportApplyResult(&bytes.Buffer{}, update.Outcome{}, timedOut)
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), ".typeburn-update.lock") {
		t.Errorf("stale-lock guidance was swallowed: %v", err)
	}
	if strings.Contains(err.Error(), "nothing was installed") {
		t.Errorf("a timed-out drain must not claim nothing was installed: %v", err)
	}
}
