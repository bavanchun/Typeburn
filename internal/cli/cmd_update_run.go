package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bavanchun/Typeburn/v2/internal/cli/updateui"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// minAnimWidth is the narrowest terminal the framed block fits in: the box's
// own width plus a small margin. The box is deliberately fixed-width, so a
// narrower terminal gets the plain output rather than a frame that wraps.
const minAnimWidth = updateui.BoxWidth + 6

// animatable reports whether the framed renderer can run. A bytes.Buffer (tests,
// pipes, redirects) is not an *os.File, so those always take the plain path.
// Overridable so tests can force either branch.
var animatable = func(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok || !term.IsTerminal(int(f.Fd())) {
		return false
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	return err == nil && cols >= minAnimWidth
}

// runApply performs the install half of `typeburn update`. Split out of
// cmd_update.go so that file keeps owning only the cobra command, the
// check/preflight flow, and the confirmation prompt.
func runApply(cmd *cobra.Command, ver, latest, execPath string) error {
	out := cmd.OutOrStdout()

	var outcome update.Outcome
	var err error
	if animatable(out) {
		outcome, err = applyAnimated(cmd, ver, latest, execPath)
	} else {
		fmt.Fprintf(out, "updating %s → %s ...\n", ver, latest)
		outcome, err = getApplyFn()(cmd.Context(), ver, latest, execPath,
			runtime.GOOS, runtime.GOARCH, plainReporter(out))
	}

	return reportApplyResult(out, outcome, err)
}

// reportApplyResult turns an Apply outcome into the command's final output and
// exit code. Split out of runApply so the ordering below is directly testable:
// the cases are not mutually exclusive — a drain timeout wraps its cause, which
// on the cancel path is ErrCancelled — so matching ErrCancelled first would
// swallow the stale-lock guidance on exactly the path it was written for.
func reportApplyResult(out io.Writer, outcome update.Outcome, err error) error {
	switch {
	case errors.Is(err, errStoppedTooLate):
		// Interrupted, but the swap had already happened. Saying "cancelled"
		// here would be the one lie the user cannot detect without re-running.
		fmt.Fprintf(out, "stopped too late — %s → %s was already installed. restart typeburn to use the new version.\n",
			outcome.From, outcome.To)
		return nil
	case errors.Is(err, errDrainTimeout):
		return ioError("update failed: %v", err)
	case errors.Is(err, updateui.ErrCancelled):
		return abortError("update cancelled; nothing was installed")
	case err != nil:
		return ioError("update failed: %v", err)
	}
	fmt.Fprintf(out, "updated %s → %s. restart typeburn to use the new version.\n", outcome.From, outcome.To)
	return nil
}

// plainReporter renders one line per stage on a non-animated stream. The
// download stage reports repeatedly as bytes arrive, so transitions are
// deduplicated — otherwise a 4 MB download would emit a hundred identical
// "downloading..." lines.
func plainReporter(out io.Writer) func(update.Progress) {
	last := update.Stage(-1)
	return func(p update.Progress) {
		if p.Stage == last {
			return
		}
		last = p.Stage
		fmt.Fprintf(out, "  %s...\n", p.Stage)
	}
}

// applyDrainTimeout bounds how long we wait for a cancelled update to unwind.
// Generous on purpose: if the cancel lands during the swap, the extraction and
// rename take no context and will run to completion, and letting them finish is
// far safer than abandoning them.
const applyDrainTimeout = 15 * time.Second

var (
	errUnexpectedModel = errors.New("update ui returned an unexpected model")
	errInterrupted     = errors.New("update interrupted before it reported an outcome; re-run typeburn update to confirm the installed version")
	errDrainTimeout    = errors.New("the update did not stop in time")

	// errStoppedTooLate reports that the run was interrupted but finished
	// anyway. Only the download is cancellable — verification, extraction and
	// the rename take no context — so a stop arriving late lands after the
	// binary has already been replaced.
	errStoppedTooLate = errors.New("stopped too late; the update had already completed")
)

// applyAnimated runs the update behind the framed renderer. The program owns
// the terminal only for the duration of the download — the confirmation prompt
// has already completed, and the success line is printed by the caller after
// this returns, so both survive in scrollback.
func applyAnimated(cmd *cobra.Command, ver, latest, execPath string) (update.Outcome, error) {
	th := theme.Load(storage.LoadSettings().Theme, theme.EnvNoColor())

	// Apply must be cancellable from here. The root context is never cancelled
	// on its own: main passes context.Background() to fang without any signal
	// options, so nothing upstream would ever stop the goroutine below.
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel()

	var trk tracker
	results := make(chan updateui.Result, 1)

	go func() {
		outcome, err := getApplyFn()(ctx, ver, latest, execPath,
			runtime.GOOS, runtime.GOARCH, trk.set)
		results <- updateui.Result{Outcome: outcome, Err: err}
	}()

	model := updateui.New(ver, latest, th, trk.get, results)
	final, err := tea.NewProgram(model, tea.WithContext(ctx)).Run()
	if err != nil {
		return stopApply(cancel, results, err, applyDrainTimeout)
	}

	m, ok := final.(updateui.Model)
	if !ok {
		return stopApply(cancel, results, errUnexpectedModel, applyDrainTimeout)
	}

	res, settled := m.Result()
	switch {
	case !settled:
		// Bubble Tea returns directly on QuitMsg and InterruptMsg without
		// routing through the model, so an unsettled model means the program
		// was killed while the update was still running.
		return stopApply(cancel, results, errInterrupted, applyDrainTimeout)
	case errors.Is(res.Err, updateui.ErrCancelled):
		return stopApply(cancel, results, updateui.ErrCancelled, applyDrainTimeout)
	}
	return res.Outcome, res.Err
}

// stopApply cancels an update that is still in flight and waits for it to
// return, so its deferred cleanup actually runs. Apply holds an O_EXCL lock
// file in the install directory; leaking it makes every later `typeburn update`
// refuse to start until the user deletes it by hand. The partial archive and
// the checksums file are removed by the same defers.
//
// Returning without waiting is not an option: the caller exits the process
// immediately afterwards, which would kill the goroutine mid-cleanup.
// The drained result is returned, not discarded: cancelling is a request, not a
// guarantee, and the caller has to report what actually happened on disk.
//
// timeout is a parameter rather than a package var so tests can shorten it
// without mutating shared state.
func stopApply(cancel context.CancelFunc, results <-chan updateui.Result, cause error, timeout time.Duration) (update.Outcome, error) {
	cancel()
	select {
	case res := <-results:
		if res.Err == nil {
			return res.Outcome, errStoppedTooLate
		}
		return update.Outcome{}, cause
	case <-time.After(timeout):
		return update.Outcome{}, fmt.Errorf("%w: %w (it did not stop within %s; remove a stale .typeburn-update.lock from the install directory if the next run refuses to start)",
			errDrainTimeout, cause, timeout)
	}
}
