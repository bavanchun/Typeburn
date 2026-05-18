// Command monkeytype-tui is a distraction-free terminal typing test.
// This entrypoint wires the theme (honoring NO_COLOR) and starts the
// Bubble Tea program. Settings persistence is added in Phase 7.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"monkeytype-tui/internal/app"
	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

func main() {
	settings := config.Defaults()
	th := theme.Load(settings.Theme, theme.EnvNoColor())

	p := tea.NewProgram(app.New(th, settings))
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "monkeytype-tui:", err)
		os.Exit(1)
	}
}
