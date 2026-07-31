package cli

import (
	"fmt"
	"io"
	"runtime"

	"github.com/bavanchun/Typeburn/v2/internal/update"
	"github.com/spf13/cobra"
)

// runApply performs the install half of `typeburn update`: it runs update.Apply
// and renders its progress. Split out of cmd_update.go so that file keeps
// owning only the cobra command, the check/preflight flow, and the prompt.
func runApply(cmd *cobra.Command, ver, latest, execPath string) error {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "updating %s → %s ...\n", ver, latest)

	outcome, err := getApplyFn()(cmd.Context(), ver, latest, execPath,
		runtime.GOOS, runtime.GOARCH, plainReporter(out))
	if err != nil {
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
