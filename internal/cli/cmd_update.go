package cli

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"

	"github.com/bavanchun/Typeburn/v2/internal/update"
	"github.com/bavanchun/Typeburn/v2/internal/version"
	"github.com/spf13/cobra"

	"golang.org/x/term"
)

// applyFn is update.Apply; overridden in tests via setApplyFn so the command is
// exercised without a real download or binary swap.
var (
	applyFnMu sync.Mutex
	applyFn   = update.Apply
)

func getApplyFn() func(context.Context, string, string, string, string, string, func(update.Stage)) (update.Outcome, error) {
	applyFnMu.Lock()
	defer applyFnMu.Unlock()
	return applyFn
}

func setApplyFn(fn func(context.Context, string, string, string, string, string, func(update.Stage)) (update.Outcome, error)) {
	applyFnMu.Lock()
	defer applyFnMu.Unlock()
	applyFn = fn
}

// execPathFn locates the running binary; overridden in tests with a temp path.
var execPathFn = os.Executable

// isInteractive reports whether r is a real terminal. A bytes.Buffer (tests,
// pipes) is not an *os.File, so this returns false without ever reading — the
// caller refuses to prompt rather than block. Tests override it to force the
// prompt path with a buffered stdin.
var isInteractive = func(r io.Reader) bool {
	f, ok := r.(*os.File)
	return ok && term.IsTerminal(int(f.Fd()))
}

func newUpdateCmd(e env) *cobra.Command {
	var yes, check bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update typeburn to the latest release",
		Long: "Download and install the latest typeburn release over the running binary.\n\n" +
			"Integrity is verified with the published SHA-256 checksums over HTTPS — the\n" +
			"same trust model as `curl install.sh | sh`. Release binaries are unsigned, so\n" +
			"this guards against corruption and truncation, not a compromised release host.\n\n" +
			"Homebrew and `go install` builds are refused with the matching upgrade command.",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runUpdate(cmd, yes, check)
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "skip the confirmation prompt")
	cmd.Flags().BoolVar(&check, "check", false, "report whether an update is available, without installing")
	return cmd
}

func runUpdate(cmd *cobra.Command, yes, check bool) error {
	ver := version.Resolve().Version
	result, checkErr := getCheckFn()(cmd.Context(), ver, true)

	if check {
		return reportCheck(cmd, ver, result, checkErr)
	}

	// Install path. (nil,nil) only happens for dev/pseudo builds (check.go),
	// which cannot self-update — refuse with a non-zero code.
	if result == nil && checkErr == nil {
		return usageError("this build has no release version; install a released build to use self-update")
	}
	if checkErr != nil {
		return ioError("could not check for updates: %v", checkErr)
	}
	if !result.UpgradeAvailable {
		fmt.Fprintf(cmd.OutOrStdout(), "you are on the latest version (%s).\n", ver)
		return nil
	}

	execPath, err := execPathFn()
	if err != nil {
		return ioError("cannot locate the running binary: %v", err)
	}
	plan := update.Preflight(execPath, os.Getenv)
	if plan.Managed {
		fmt.Fprintf(cmd.OutOrStdout(),
			"typeburn %s is available (you have %s).\nThis binary is managed by your package manager; upgrade with:\n  %s\n",
			result.Latest, ver, plan.Instruction)
		return managedInstallError("managed install: use %q", plan.Instruction)
	}
	if !plan.Writable {
		return ioError("install directory %s is not writable; re-run with sufficient permissions or reinstall", plan.Dir)
	}

	printReleaseNotes(cmd.OutOrStdout(), result.ReleaseURL)

	if !yes {
		if !isInteractive(cmd.InOrStdin()) {
			return usageError("refusing to prompt on a non-interactive stream; re-run with --yes")
		}
		ok, err := confirmUpdate(cmd, ver, result.Latest)
		if err != nil {
			return ioError("read confirmation: %v", err)
		}
		if !ok {
			fmt.Fprintln(cmd.OutOrStdout(), "update cancelled.")
			return nil
		}
	}

	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "updating %s → %s ...\n", ver, result.Latest)
	progress := func(s update.Stage) { fmt.Fprintf(out, "  %s...\n", s) }
	outcome, err := getApplyFn()(cmd.Context(), ver, result.Latest, execPath, runtime.GOOS, runtime.GOARCH, progress)
	if err != nil {
		return ioError("update failed: %v", err)
	}
	fmt.Fprintf(out, "updated %s → %s. restart typeburn to use the new version.\n", outcome.From, outcome.To)
	return nil
}

// reportCheck handles `update --check`: detect-only, always exit 0 (mirrors
// `version --check-update`).
func reportCheck(cmd *cobra.Command, ver string, result *update.Result, checkErr error) error {
	out := cmd.OutOrStdout()
	switch {
	case result == nil && checkErr == nil:
		fmt.Fprintln(out, "update check skipped: this build has no release version.")
	case checkErr != nil:
		fmt.Fprintf(cmd.ErrOrStderr(), "could not check for updates: %v\n", checkErr)
	case result.UpgradeAvailable:
		fmt.Fprintf(out, "typeburn %s is available (you have %s).\n", result.Latest, ver)
		printReleaseNotes(out, result.ReleaseURL)
		fmt.Fprintln(out, "Run 'typeburn update' to upgrade.")
	default:
		fmt.Fprintf(out, "you are on the latest version (%s).\n", ver)
	}
	return nil
}

// printReleaseNotes writes a "Release notes: <url>" line when url is non-empty,
// matching the wording of `version --check-update`. url is already repo-guarded
// upstream in update.Check, so no further validation is needed here.
func printReleaseNotes(w io.Writer, url string) {
	if url != "" {
		fmt.Fprintf(w, "Release notes: %s\n", url)
	}
}

func confirmUpdate(cmd *cobra.Command, cur, latest string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "Update typeburn %s → %s? [y/N]: ", cur, latest)
	line, err := bufio.NewReader(cmd.InOrStdin()).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}
