package cli

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/bavanchun/Typeburn/v2/internal/cli/updateui"
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
//
// Both branches return through the same drain-and-report contract, so an
// interrupt behaves identically whether the user is on a terminal or piping the
// output into a log.
func runApply(cmd *cobra.Command, ver, latest, execPath string) error {
	out := cmd.OutOrStdout()

	var outcome update.Outcome
	var err error
	if animatable(out) {
		outcome, err = applyAnimated(cmd, ver, latest, execPath)
	} else {
		outcome, err = applyPlain(cmd, ver, latest, execPath)
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
