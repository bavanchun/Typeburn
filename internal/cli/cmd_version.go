package cli

import (
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/bavanchun/Typeburn/internal/version"
	"github.com/spf13/cobra"
)

func newVersionCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:           "version",
		Short:         "Print version information",
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print version as JSON")
	return cmd
}

func runVersion(cmd *cobra.Command, asJSON bool) error {
	if !asJSON {
		fmt.Fprintln(cmd.OutOrStdout(), version.String())
		return nil
	}
	info := version.Resolve()
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
