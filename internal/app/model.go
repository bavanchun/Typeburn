// Package app holds the Bubble Tea root model. It owns global state
// (terminal size, theme, keymap, active screen) and routes messages to the
// active screen's sub-model.
package app

import (
	tea "charm.land/bubbletea/v2"

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
	sett     ui.SettingsModel
}

// New builds the root model loading persisted settings from disk.
// It falls back to config.Defaults() if the settings file is missing or corrupt.
func New(th theme.Theme, settings config.Settings) Model {
	km := config.DefaultKeymap()
	home := ui.NewHome(settings, th, km)
	m := Model{
		screen:   ScreenHome,
		theme:    th,
		keys:     km,
		settings: settings,
		home:     home,
	}
	m.sett = ui.NewSettings(&m.settings, th, km, m.onSettingsChange)
	return m
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
		m.sett = m.sett.SetSize(msg.Width, msg.Height)
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
	if m.screen == ScreenSettings {
		var cmd tea.Cmd
		m.sett, cmd = m.sett.Update(msg)
		return m, cmd
	}

	return m, nil
}

// handleKey processes global bindings then delegates to the active screen.
func (m Model) handleKey(key tea.Key) (tea.Model, tea.Cmd) {
	// Quit is always global, regardless of active screen.
	if m.keys.Quit.Matches(key) {
		return m, tea.Quit
	}

	// While typing, delegate all key handling to the typing sub-model.
	// Typing captures the full keyboard; no global nav applies mid-test.
	if m.screen == ScreenTyping {
		var cmd tea.Cmd
		m.typing, cmd = m.typing.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	// Global navigation keys apply on all non-typing screens so that e.g.
	// pressing '3' from the Settings screen still reaches History.
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

	// Delegate remaining keys to the active sub-model.
	switch m.screen {
	case ScreenResult:
		var cmd tea.Cmd
		m.result, cmd = m.result.Update(tea.KeyPressMsg(key))
		return m, cmd
	case ScreenSettings:
		var cmd tea.Cmd
		m.sett, cmd = m.sett.Update(tea.KeyPressMsg(key))
		return m, cmd
	case ScreenHome:
		var cmd tea.Cmd
		m.home, cmd = m.home.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	return m, nil
}
