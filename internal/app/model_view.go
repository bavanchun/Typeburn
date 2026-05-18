package app

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"monkeytype-tui/internal/ui"
)

// View renders the active screen centered in the terminal.
//
// Single chokepoint: if the terminal is below the 60×20 safe minimum, the
// degraded notice is shown instead of any screen content. This prevents any
// screen from partial-painting at small sizes.
//
// When the quit-prompt overlay is active (esc pressed on Home), it is rendered
// instead of the Home screen content.
func (m Model) View() tea.View {
	// Degraded gate — must check before any screen delegation.
	if m.w > 0 && m.h > 0 && (m.w < 60 || m.h < 20) {
		notice := ui.DegradedNotice(m.w, m.h, m.theme)
		return tea.NewView(lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, notice))
	}

	// Quit-prompt overlay on Home screen.
	if m.quitPrompt != nil && m.screen == ScreenHome {
		return tea.NewView(m.quitPrompt.view(m.w, m.h, m.theme))
	}

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
	case ScreenHistory:
		// HistoryModel.View() calls lipgloss.Place internally.
		body = m.hist.View()
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
