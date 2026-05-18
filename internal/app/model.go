// Package app holds the Bubble Tea root model. It owns global state
// (terminal size, theme, keymap, active screen) and routes messages to the
// active screen's sub-model.
package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/theme"
	"monkeytype-tui/internal/ui"
	"monkeytype-tui/internal/words"
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
	typing   ui.TypingModel
}

// New builds the root model with the given theme and settings.
func New(th theme.Theme, settings config.Settings) Model {
	km := config.DefaultKeymap()
	typing := ui.NewTyping(
		settings.DefaultMode,
		settings.DefaultLength,
		words.QuoteMedium,
		th,
		km,
		settings.BlinkCursor,
	)
	return Model{
		screen:   ScreenHome,
		theme:    th,
		keys:     km,
		settings: settings,
		typing:   typing,
	}
}

// Init performs no startup command in Phase 1–4 skeleton.
func (m Model) Init() tea.Cmd { return nil }

// Update handles global concerns (resize, quit, navigation) and delegates to
// the active screen sub-model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// ResultMsg: test completed → placeholder transition until Phase 6.
	if _, ok := msg.(ui.ResultMsg); ok {
		m.screen = ScreenResult
		return m, nil
	}

	// abortMsg from typing screen → return to Home.
	if _, ok := msg.(ui.AbortMsg); ok {
		m.screen = ScreenHome
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		if m.screen == ScreenTyping {
			m.typing = m.typing.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())
	}

	// Delegate remaining messages to the active screen sub-model.
	if m.screen == ScreenTyping {
		var cmd tea.Cmd
		m.typing, cmd = m.typing.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey processes global bindings then delegates to the active screen.
func (m Model) handleKey(key tea.Key) (tea.Model, tea.Cmd) {
	// Quit is always global.
	if m.keys.Quit.Matches(key) {
		return m, tea.Quit
	}

	// While typing, delegate all key handling to the typing sub-model.
	if m.screen == ScreenTyping {
		var cmd tea.Cmd
		m.typing, cmd = m.typing.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	// Non-typing screens: handle global nav.
	switch {
	case m.keys.NavHome.Matches(key):
		m.screen = ScreenHome
	case m.keys.NavSettings.Matches(key):
		m.screen = ScreenSettings
	case m.keys.NavHistory.Matches(key):
		m.screen = ScreenHistory
	case m.keys.Back.Matches(key):
		if m.screen == ScreenHome {
			return m, tea.Quit
		}
		m.screen = ScreenHome
	// Space/Enter on Home starts the typing test.
	case m.screen == ScreenHome && m.keys.Start.Matches(key):
		m.typing = ui.NewTyping(
			m.settings.DefaultMode,
			m.settings.DefaultLength,
			words.QuoteMedium,
			m.theme,
			m.keys,
			m.settings.BlinkCursor,
		).SetSize(m.w, m.h)
		m.screen = ScreenTyping
	}
	return m, nil
}

// View centers the active screen's content in the terminal.
func (m Model) View() tea.View {
	var body string
	if m.screen == ScreenTyping {
		body = m.typing.View()
	} else {
		body = placeholderView(m.screen, m.theme)
	}

	content := body
	if m.w > 0 && m.h > 0 {
		content = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
	}
	return tea.NewView(content)
}
