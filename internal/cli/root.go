package cli

import (
	"context"

	"github.com/spf13/cobra"
)

// NewRoot builds the root command. Root-level unknown args intentionally fall
// through to the TUI; recognized subcommands keep strict cobra parsing.
func NewRoot(opts ...Option) *cobra.Command {
	e := defaultEnv()
	for _, opt := range opts {
		opt(&e)
	}

	root := &cobra.Command{
		Use:                   "typeburn",
		Short:                 "Distraction-free terminal typing test",
		DisableFlagParsing:    true,
		DisableSuggestions:    true,
		SilenceErrors:         true,
		SilenceUsage:          true,
		Args:                  cobra.ArbitraryArgs,
		DisableFlagsInUseLine: true,
		Example: "  typeburn\n" +
			"  typeburn run --mode time --duration 30\n" +
			"  typeburn history --json\n" +
			"  typeburn config set theme nord",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
				return cmd.Help()
			}
			printVersion, textPath := Decide(args)
			if printVersion {
				return runVersion(cmd, false, false)
			}
			return launchHome(cmd.Context(), e, textPath)
		},
	}
	root.SetOut(e.stdout)
	root.SetErr(e.stderr)
	root.SetIn(e.stdin)
	root.AddCommand(newVersionCmd())
	root.AddCommand(newRunCmd(e))
	root.AddCommand(newHistoryCmd(e))
	root.AddCommand(newConfigCmd(e))
	root.AddCommand(newReplayCmd())
	return root
}

func launchHome(ctx context.Context, e env, textPath string) error {
	var codeText, codeHint string
	if textPath != "" {
		text, err := e.loadCode(textPath)
		if err != nil {
			codeHint = codeHintFor(err)
		} else {
			codeText = text
		}
	}
	return e.runHome(ctx, codeText, codeHint)
}
