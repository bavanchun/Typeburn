package cli

import (
	"flag"
	"io"
)

// Decide maps root-level legacy args to startup decisions.
// It is pure: no os.Exit, no I/O, no usage dump.
func Decide(args []string) (printVersion bool, textPath string) {
	fs := flag.NewFlagSet("typeburn", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print version and exit")
	textFlag := fs.String("text", "", "file to load as code typing target (\"-\" = stdin)")
	if err := fs.Parse(args); err != nil {
		return false, ""
	}
	return *showVersion, *textFlag
}
