// Command typeburn is a distraction-free terminal typing test.
// This entrypoint loads persisted settings (XDG config dir, atomic JSON),
// builds the themed root model, and starts the Bubble Tea program.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/app"
	"github.com/bavanchun/Typeburn/internal/codetext"
	"github.com/bavanchun/Typeburn/internal/version"
)

// decide maps process args to the startup decisions: print the version banner,
// or launch the TUI, and optionally which file path to load as code text.
// It is a pure function (no os.Exit, no I/O) so the arg-handling contract is
// unit-testable.
//
// Only an explicitly parsed --version short-circuits. A dedicated
// ContinueOnError FlagSet with discarded output means unknown flags, -h and
// the reserved -v all return a parse error, which we treat as "just launch the
// TUI" — preserving the long-standing behavior that `typeburn <anything>`
// starts the test rather than exiting 2 with a usage dump.
//
// textPath is "" when --text is absent or when parse fails (fall-through).
func decide(args []string) (printVersion bool, textPath string) {
	fs := flag.NewFlagSet("typeburn", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print version and exit")
	textFlag := fs.String("text", "", "file to load as code typing target (\"-\" = stdin)")
	if err := fs.Parse(args); err != nil {
		return false, ""
	}
	return *showVersion, *textFlag
}

// codeHintFor returns a user-facing one-liner explaining a codetext load error.
// It branches on the sentinel errors defined by the codetext package.
func codeHintFor(err error) string {
	switch {
	case errors.Is(err, codetext.ErrEmpty):
		return "text file is empty"
	case errors.Is(err, codetext.ErrTooLarge):
		return "text file too large"
	case errors.Is(err, codetext.ErrBinary):
		return "file is not text"
	default:
		return "could not read text file"
	}
}

func main() {
	printVersion, textPath := decide(os.Args[1:])
	if printVersion {
		fmt.Println(version.String())
		return
	}

	var codeText, codeHint string
	if textPath != "" {
		var err error
		codeText, err = codetext.Load(textPath)
		if err != nil {
			codeHint = codeHintFor(err)
		}
	}

	p := tea.NewProgram(app.NewFromDisk(codeText, codeHint))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "typeburn:", err)
		os.Exit(1)
	}
}
