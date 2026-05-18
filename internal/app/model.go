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
	home     ui.HomeModel
	typing   ui.TypingModel
	result   ui.ResultModel
}

// New builds the root model with the given theme and settings.
func New(th theme.Theme, settings config.Settings) Model {
	km := config.DefaultKeymap()
	home := ui.NewHome(settings, th, km)
	return Model{
		screen:   ScreenHome,
		theme:    th,
		keys:     km,
		settings: settings,
		home:     home,
	}
}

// Init performs no startup command.
func (m Model) Init() tea.Cmd { return nil }

// Update handles global concerns (resize, quit, navigation) and delegates to
// the active screen sub-model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// StartTestMsg: Home screen requested a test start.
	if sm, ok := msg.(ui.StartTestMsg); ok {
		t := ui.NewTyping(sm.Mode, sm.Length, sm.QuoteLen,
			m.theme, m.keys, m.settings.BlinkCursor).SetSize(m.w, m.h)
		m.typing = t
		m.screen = ScreenTyping
		return m, nil
	}

	// ResultMsg: test completed → construct real ResultModel (Phase 6).
	if rm, ok := msg.(ui.ResultMsg); ok {
		m.result = ui.NewResult(rm, m.theme, m.keys).SetSize(m.w, m.h)
		m.screen = ScreenResult
		return m, nil
	}

	// NavHistoryMsg: navigate to History screen.
	if _, ok := msg.(ui.NavHistoryMsg); ok {
		m.screen = ScreenHistory
		return m, nil
	}

	// AbortMsg from typing screen → return to Home.
	if _, ok := msg.(ui.AbortMsg); ok {
		m.screen = ScreenHome
		return m, nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.home = m.home.SetSize(msg.Width, msg.Height)
		if m.screen == ScreenTyping {
			m.typing = m.typing.SetSize(msg.Width, msg.Height)
		}
		if m.screen == ScreenResult {
			m.result = m.result.SetSize(msg.Width, msg.Height)
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
	if m.screen == ScreenResult {
		var cmd tea.Cmd
		m.result, cmd = m.result.Update(msg)
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

	// While on result screen, delegate to result sub-model.
	if m.screen == ScreenResult {
		var cmd tea.Cmd
		m.result, cmd = m.result.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	// Non-typing screens: handle global nav first.
	switch {
	case m.keys.NavHome.Matches(key):
		m.screen = ScreenHome
		return m, nil
	case m.keys.NavSettings.Matches(key):
		m.screen = ScreenSettings
		return m, nil
	case m.keys.NavHistory.Matches(key):
		m.screen = ScreenHistory
		return m, nil
	case m.keys.Back.Matches(key):
		if m.screen == ScreenHome {
			return m, tea.Quit
		}
		m.screen = ScreenHome
		return m, nil
	}

	// Delegate remaining keys to the Home screen when active.
	if m.screen == ScreenHome {
		var cmd tea.Cmd
		m.home, cmd = m.home.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	return m, nil
}

// View renders the active screen centered in the terminal.
func (m Model) View() tea.View {
	var body string
	switch m.screen {
	case ScreenHome:
		// HomeModel.View() already calls lipgloss.Place internally when
		// dimensions are known; pass the raw string through to avoid
		// double-placing.
		body = m.home.View()
		if m.w > 0 && m.h > 0 {
			return tea.NewView(body)
		}
	case ScreenTyping:
		body = m.typing.View()
	case ScreenResult:
		// ResultModel.View() calls lipgloss.Place internally.
		body = m.result.View()
		if m.w > 0 && m.h > 0 {
			return tea.NewView(body)
		}
	default:
		body = placeholderView(m.screen, m.theme)
	}

	content := body
	if m.w > 0 && m.h > 0 {
		content = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, body)
	}
	return tea.NewView(content)
}
