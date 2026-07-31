package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"
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

	switch {
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

// tracker is the hand-off between the blocking Apply goroutine and the render
// loop. Progress is state, not an event stream, so the renderer samples the
// latest snapshot on its own cadence instead of draining a channel: no
// backpressure on the download, and no stage transition can be dropped.
type tracker struct {
	mu  sync.Mutex
	cur update.Progress
}

func (t *tracker) set(p update.Progress) {
	t.mu.Lock()
	t.cur = p
	t.mu.Unlock()
}

func (t *tracker) get() update.Progress {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.cur
}

// applyDrainTimeout bounds how long we wait for a cancelled update to unwind.
// Generous on purpose: if the cancel lands during the swap, the extraction and
// rename take no context and will run to completion, and letting them finish is
// far safer than abandoning them.
const applyDrainTimeout = 15 * time.Second

// applyDrainTimeoutForTest is the value stopApply actually waits for; tests
// shorten it so the timeout branch is exercised without a 15s pause.
var applyDrainTimeoutForTest = applyDrainTimeout

var (
	errUnexpectedModel = errors.New("update ui returned an unexpected model")
	errInterrupted     = errors.New("update interrupted before it reported an outcome; re-run typeburn update to confirm the installed version")
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
		return update.Outcome{}, stopApply(cancel, results, err)
	}

	m, ok := final.(updateui.Model)
	if !ok {
		return update.Outcome{}, stopApply(cancel, results, errUnexpectedModel)
	}

	res, settled := m.Result()
	switch {
	case !settled:
		// Bubble Tea returns directly on QuitMsg and InterruptMsg without
		// routing through the model, so an unsettled model means the program
		// was killed while the update was still running.
		return update.Outcome{}, stopApply(cancel, results, errInterrupted)
	case errors.Is(res.Err, updateui.ErrCancelled):
		return update.Outcome{}, stopApply(cancel, results, updateui.ErrCancelled)
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
func stopApply(cancel context.CancelFunc, results <-chan updateui.Result, cause error) error {
	cancel()
	select {
	case <-results:
		return cause
	case <-time.After(applyDrainTimeoutForTest):
		return fmt.Errorf("%w (it did not stop within %s; remove a stale .typeburn-update.lock from the install directory if the next run refuses to start)",
			cause, applyDrainTimeoutForTest)
	}
}
