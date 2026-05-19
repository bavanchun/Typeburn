package app

import (
	tea "charm.land/bubbletea/v2"
)

// handleKey processes global key bindings then delegates to the active screen.
//
// Priority order:
//  1. ctrl+c → hard quit everywhere, including during quit-prompt overlay.
//  2. Quit-prompt overlay (when active on Home) → route to prompt handler.
//  3. Typing screen → delegate all keys to TypingModel (no global nav mid-test).
//  4. Global nav keys (1/2/3/esc) on non-typing screens.
//  5. Remaining keys → delegate to the active sub-model.
func (m Model) handleKey(key tea.Key) (tea.Model, tea.Cmd) {
	// ctrl+c hard-quits everywhere — even during the quit-prompt overlay.
	if m.keys.Quit.Matches(key) {
		return m, tea.Quit
	}

	// Any other key dismisses the persistence-failure toast (it is purely
	// informational). The keystroke still proceeds to its normal handling
	// below; m is a value receiver so this cleared copy propagates out.
	m.persistErr = ""

	// Quit-prompt overlay: route keys into the prompt; dismiss or quit as needed.
	// ctrl+c (above) already bypasses this, so only esc/enter/y/n/arrows reach here.
	if m.quitPrompt != nil && m.screen == ScreenHome {
		updated, dismissed, quit := m.quitPrompt.handleKey(key)
		if quit {
			return m, tea.Quit
		}
		if dismissed {
			m.quitPrompt = nil
			return m, nil
		}
		m.quitPrompt = &updated
		return m, nil
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
		m.quitPrompt = nil // dismiss any lingering prompt on explicit nav
		return m, nil
	case m.keys.NavSettings.Matches(key):
		m.screen = ScreenSettings
		m.quitPrompt = nil
		return m, nil
	case m.keys.NavHistory.Matches(key):
		m.quitPrompt = nil
		m = m.handleNavHistory()
		return m, nil
	case m.keys.Back.Matches(key):
		if m.screen == ScreenHome {
			// esc on Home → show quit-prompt instead of immediate quit (design §8.1).
			p := newQuitPrompt()
			m.quitPrompt = &p
			return m, nil
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
	case ScreenHistory:
		var cmd tea.Cmd
		m.hist, cmd = m.hist.Update(tea.KeyPressMsg(key))
		return m, cmd
	case ScreenCodePaste:
		// The paste sub-model only consumes tea.PasteMsg; keys are no-ops
		// here. esc/back already routed above (global Back else → Home).
		var cmd tea.Cmd
		m.codePaste, cmd = m.codePaste.Update(tea.KeyPressMsg(key))
		return m, cmd
	}

	return m, nil
}
