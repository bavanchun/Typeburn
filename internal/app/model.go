// Package app holds the Bubble Tea root model. It owns global state
// (terminal size, theme, keymap, active screen) and routes messages to the
// active screen's sub-model.
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
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
	hist     ui.HistoryModel

	// quitPrompt is non-nil when the esc-on-Home quit confirmation overlay is
	// active. ctrl+c always hard-quits regardless of this field.
	quitPrompt *quitPromptModel
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

	// ResultMsg: test completed → persist record, detect new-best, show result.
	if rm, ok := msg.(ui.ResultMsg); ok {
		m = m.handleResultMsg(rm)
		return m, nil
	}

	// NavHistoryMsg: navigate to History screen (load fresh from disk).
	if _, ok := msg.(ui.NavHistoryMsg); ok {
		m = m.handleNavHistory()
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
		m.hist = m.hist.SetSize(msg.Width, msg.Height)
		if m.screen == ScreenTyping {
			m.typing = m.typing.SetSize(msg.Width, msg.Height)
		}
		if m.screen == ScreenResult {
			m.result = m.result.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())

	case tea.PasteMsg:
		// Feed paste runes sequentially into the typing engine (project decision
		// documented in charm-v2-api-cheatsheet.md). Ignored on all other screens.
		if m.screen == ScreenTyping {
			var cmd tea.Cmd
			m.typing, cmd = m.typing.Update(msg)
			return m, cmd
		}
		return m, nil
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
	if m.screen == ScreenHistory {
		var cmd tea.Cmd
		m.hist, cmd = m.hist.Update(msg)
		return m, cmd
	}

	return m, nil
}
