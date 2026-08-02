package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/cli/updateui"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

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

// applyPlain runs the update on the non-animated stream: pipes, redirects, CI,
// and any terminal too narrow for the frame.
//
// Nothing upstream installs signal handlers — main hands cobra a plain
// context.Background() — so without the handler below SIGINT and SIGTERM hit
// the runtime's default disposition and kill the process mid-download with zero
// defers run, stranding Apply's O_EXCL lock in the install directory. The
// animated path gets this for free from Bubble Tea; this is the same contract
// for everyone else.
func applyPlain(cmd *cobra.Command, ver, latest, execPath string) (update.Outcome, error) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "updating %s → %s ...\n", ver, latest)

	ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	results := make(chan updateui.Result, 1)
	go func() {
		outcome, err := getApplyFn()(ctx, ver, latest, execPath,
			runtime.GOOS, runtime.GOARCH, plainReporter(out))
		results <- updateui.Result{Outcome: outcome, Err: err}
	}()

	select {
	case res := <-results:
		return res.Outcome, res.Err
	case <-ctx.Done():
		// stop() also cancels ctx, which is already done here; passing it keeps
		// the drain path identical to the animated one.
		return stopApply(stop, results, updateui.ErrCancelled, applyDrainTimeout)
	}
}

// applyAnimated runs the update behind the framed renderer. The program owns
// the terminal only for the duration of the download — the confirmation prompt
// has already completed, and the success line is printed by the caller after
// this returns, so both survive in scrollback.
func applyAnimated(cmd *cobra.Command, ver, latest, execPath string) (update.Outcome, error) {
	th := theme.Load(storage.LoadSettings().Theme, theme.EnvNoColor())

	// Apply must be cancellable from here: Bubble Tea's own interrupt handling
	// returns from Run without stopping the goroutine below.
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
