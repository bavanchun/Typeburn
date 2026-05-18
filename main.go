// Command typeburn is a distraction-free terminal typing test.
// This entrypoint loads persisted settings (XDG config dir, atomic JSON),
// builds the themed root model, and starts the Bubble Tea program.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/app"
	"github.com/bavanchun/Typeburn/internal/version"
)

// decide maps process args to the single startup decision: print the version
// banner, or launch the TUI. It is a pure function (no os.Exit, no I/O) so the
// arg-handling contract is unit-testable.
//
// Only an explicitly parsed --version short-circuits. A dedicated
// ContinueOnError FlagSet with discarded output means unknown flags, -h and
// the reserved -v all return a parse error, which we treat as "just launch the
// TUI" — preserving the long-standing behavior that `typeburn <anything>`
// starts the test rather than exiting 2 with a usage dump.
func decide(args []string) (printVersion bool) {
	fs := flag.NewFlagSet("typeburn", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	showVersion := fs.Bool("version", false, "print version and exit")
	if err := fs.Parse(args); err != nil {
		return false
	}
	return *showVersion
}

func main() {
	if decide(os.Args[1:]) {
		fmt.Println(version.String())
		return
	}

	p := tea.NewProgram(app.NewFromDisk())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "typeburn:", err)
		os.Exit(1)
	}
}
