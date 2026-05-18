// Command typeburn is a distraction-free terminal typing test.
// This entrypoint loads persisted settings (XDG config dir, atomic JSON),
// builds the themed root model, and starts the Bubble Tea program.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/app"
)

func main() {
	p := tea.NewProgram(app.NewFromDisk())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "typeburn:", err)
		os.Exit(1)
	}
}
