// Package app holds the Bubble Tea root model. It owns global state
// (terminal size, theme, keymap, active screen) and routes messages to the
// active screen's sub-model. Sub-models are added in later phases; Phase 1
// renders placeholders so the skeleton is runnable end to end.
package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
)

// Screen enumerates the top-level screens. Routing switches on this value.
type Screen int

const (
	ScreenHome Screen = iota
	ScreenTyping
	ScreenResult
	ScreenSettings
	ScreenHistory
)

// Model is the root Elm model.
type Model struct {
	screen   Screen
	w, h     int
	theme    theme.Theme
	keys     config.Keymap
	settings config.Settings
}

// New builds the root model with the given theme and settings. Settings are
// in-memory defaults in Phase 1; persistence is wired in Phase 7.
func New(th theme.Theme, settings config.Settings) Model {
	return Model{
		screen:   ScreenHome,
		theme:    th,
		keys:     config.DefaultKeymap(),
		settings: settings,
	}
}

// Init performs no startup command in the skeleton.
func (m Model) Init() tea.Cmd { return nil }

// Update handles global concerns (resize, quit, navigation, back) and will
// delegate to the active sub-model once sub-models exist.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())
	}
	return m, nil
}

// handleKey processes global bindings. Per-screen keys are added with each
// screen's phase; the skeleton only needs quit/nav/back to be navigable.
func (m Model) handleKey(key tea.Key) (tea.Model, tea.Cmd) {
	switch {
	case m.keys.Quit.Matches(key):
		return m, tea.Quit
	case m.keys.NavHome.Matches(key):
		m.screen = ScreenHome
	case m.keys.NavSettings.Matches(key):
		m.screen = ScreenSettings
	case m.keys.NavHistory.Matches(key):
		m.screen = ScreenHistory
	case m.keys.Back.Matches(key):
		// esc: leave a sub-screen back to Home; on Home it exits.
		// Phase 9 replaces the Home exit with a confirm prompt.
		if m.screen == ScreenHome {
			return m, tea.Quit
		}
		m.screen = ScreenHome
	}
	return m, nil
}

// View centers the active screen's content in the terminal.
func (m Model) View() tea.View {
	body := placeholderView(m.screen, m.theme)
	content := body
	if m.w > 0 && m.h > 0 {
		content = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
	}
	return tea.NewView(content)
}
