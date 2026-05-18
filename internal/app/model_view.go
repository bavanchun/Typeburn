package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// View renders the active screen centered in the terminal.
// Screens that own their own lipgloss.Place call (Home, Result, Settings) are
// returned directly to avoid double-placing; others are centered here.
func (m Model) View() tea.View {
	var body string
	switch m.screen {
	case ScreenHome:
		// HomeModel.View() calls lipgloss.Place internally.
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
	case ScreenSettings:
		// SettingsModel.View() calls lipgloss.Place internally.
		body = m.sett.View()
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
