package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/ui"
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

	// Compute the final frame string in one place (single return) so the
	// persistence notice can be overlaid uniformly. Home/Result/Settings/
	// History self-place to w×h inside their own View(); Typing and the
	// placeholder are placed here. With no notice this yields byte-identical
	// output to the previous per-branch returns.
	var out string
	switch m.screen {
	case ScreenHome:
		out = m.home.View() // self-placed
	case ScreenResult:
		out = m.result.View() // self-placed
	case ScreenSettings:
		out = m.sett.View() // self-placed
	case ScreenHistory:
		out = m.hist.View() // self-placed
	case ScreenCodePaste:
		out = m.codePaste.View() // self-placed
	case ScreenTyping:
		out = m.typing.View()
		if m.w > 0 && m.h > 0 {
			out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
		}
	default:
		out = placeholderView(m.screen, m.theme)
		if m.w > 0 && m.h > 0 {
			out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
		}
	}

	// Transient persistence-failure toast: overlay onto the frame's last row
	// (normally blank padding) so the line count — and thus every other
	// screen's layout — is unchanged. Cleared on the next keypress.
	if m.persistErr != "" && m.w > 0 && m.h > 0 {
		lines := strings.Split(out, "\n")
		notice := lipgloss.PlaceHorizontal(
			m.w, lipgloss.Center, ui.PersistenceNotice(m.persistErr, m.theme),
		)
		lines[len(lines)-1] = notice
		out = strings.Join(lines, "\n")
	}

	return tea.NewView(out)
}
