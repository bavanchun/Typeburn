package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"runtime"
	"sync"

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

// applyAnimated runs the update behind the framed renderer. The program owns
// the terminal only for the duration of the download — the confirmation prompt
// has already completed, and the success line is printed by the caller after
// this returns, so both survive in scrollback.
func applyAnimated(cmd *cobra.Command, ver, latest, execPath string) (update.Outcome, error) {
	th := theme.Load(storage.LoadSettings().Theme, theme.EnvNoColor())

	var trk tracker
	results := make(chan updateui.Result, 1)

	go func() {
		outcome, err := getApplyFn()(cmd.Context(), ver, latest, execPath,
			runtime.GOOS, runtime.GOARCH, trk.set)
		results <- updateui.Result{Outcome: outcome, Err: err}
	}()

	model := updateui.New(ver, latest, th, trk.get, results)
	final, err := tea.NewProgram(model, tea.WithContext(cmd.Context())).Run()
	if err != nil {
		return update.Outcome{}, err
	}

	res := final.(updateui.Model).Result()
	return res.Outcome, res.Err
}
