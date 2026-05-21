package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/bavanchun/Typeburn/internal/update"
	"github.com/bavanchun/Typeburn/internal/version"
	"github.com/spf13/cobra"
)

// checkFn is the update.Check function; overridden in tests via this seam.
var checkFn = update.Check

func newVersionCmd() *cobra.Command {
	var asJSON, checkUpdate bool
	cmd := &cobra.Command{
		Use:           "version",
		Short:         "Print version information",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd, asJSON, checkUpdate)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version as JSON")
	cmd.Flags().BoolVar(&checkUpdate, "check-update", false, "check GitHub for a newer release")
	return cmd
}

func runVersion(cmd *cobra.Command, asJSON, checkUpdate bool) error {
	if !asJSON && !checkUpdate {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
		return nil
	}

	// version.Resolve() returns Info{Version, Commit, Date}.
	info := version.Resolve()

	if !checkUpdate {
		return renderVersionJSON(cmd, info)
	}

	result, err := checkFn(cmd.Context(), info.Version, true)

	if asJSON {
		return renderVersionCheckJSON(cmd, info, result, err)
	}
	return renderVersionCheckHuman(cmd, info.Version, result, err)
}

func renderVersionJSON(cmd *cobra.Command, info version.Info) error {
	out := struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		Date      string `json:"date"`
		GoVersion string `json:"go_version"`
		OS        string `json:"os"`
		Arch      string `json:"arch"`
	}{
		Version:   info.Version,
		Commit:    info.Commit,
		Date:      info.Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func renderVersionCheckJSON(cmd *cobra.Command, info version.Info, result *update.Result, checkErr error) error {
	versionObj := struct {
		Version   string `json:"version"`
		Commit    string `json:"commit"`
		Date      string `json:"date"`
		GoVersion string `json:"go_version"`
		OS        string `json:"os"`
		Arch      string `json:"arch"`
	}{
		Version:   info.Version,
		Commit:    info.Commit,
		Date:      info.Date,
		GoVersion: runtime.Version(),
		OS:        runtime.GOOS,
		Arch:      runtime.GOARCH,
	}
	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")

	if checkErr != nil {
		out := struct {
			Version     any `json:"version"`
			UpdateCheck any `json:"update_check"`
		}{
			Version: versionObj,
			UpdateCheck: struct {
				Error string `json:"error"`
			}{checkErr.Error()},
		}
		_ = enc.Encode(out)
		return checkErr
	}
	if result == nil {
		out := struct {
			Version     any `json:"version"`
			UpdateCheck any `json:"update_check"`
		}{
			Version: versionObj,
			UpdateCheck: struct {
				Skipped string `json:"skipped"`
			}{"build has no release version"},
		}
		return enc.Encode(out)
	}
	out := struct {
		Version     any            `json:"version"`
		UpdateCheck *update.Result `json:"update_check"`
	}{
		Version:     versionObj,
		UpdateCheck: result,
	}
	return enc.Encode(out)
}

func renderVersionCheckHuman(cmd *cobra.Command, currentVer string, result *update.Result, checkErr error) error {
	// Always print the plain version banner first.
	fmt.Fprintln(cmd.OutOrStdout(), version.String())

	switch {
	case result == nil && checkErr == nil:
		fmt.Fprintln(cmd.OutOrStdout(), "version check skipped: build has no release version.")
	case checkErr != nil:
		fmt.Fprintf(cmd.ErrOrStderr(), "could not check for updates: %v\n", checkErr)
	case result.UpgradeAvailable:
		fmt.Fprintf(cmd.OutOrStdout(),
			"\ntypeburn %s is available (you have %s).\nRelease notes: %s\nUpgrade with one of:\n  brew upgrade typeburn\n  curl -fsSL https://raw.githubusercontent.com/bavanchun/Typeburn/main/install.sh | sh\n  go install github.com/bavanchun/Typeburn@latest\n",
			result.Latest, currentVer, result.ReleaseURL,
		)
	default:
		fmt.Fprintf(cmd.OutOrStdout(), "you are on the latest version (%s).\n", currentVer)
	}
	return nil
}
