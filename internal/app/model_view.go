package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/anim"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
)

// altView wraps a frame string in the alternate screen buffer. The altscreen
// gives the program full-window mode with no scrollback (the TUI cannot be
// scrolled) and restores the prior terminal contents on quit. Every View()
// return path goes through here so altscreen is unconditional.
func altView(s string) tea.View {
	v := tea.NewView(s)
	v.AltScreen = true
	return v
}

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
		return altView(lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, notice))
	}

	// Quit-prompt overlay on Home screen.
	if m.quitPrompt != nil && m.screen == ScreenHome {
		return altView(m.quitPrompt.view(m.w, m.h, m.theme))
	}

	// Compute the final frame string in one place (single return) so the
	// persistence notice can be overlaid uniformly. With no transition and no
	// notice this yields byte-identical output to the previous per-branch returns.
	out := m.composeScreen(m.screen)

	// Screen transition: while a root-owned transition is mid-flight, blend the
	// snapshotted outgoing frame with the live incoming frame (out). Expiry is
	// derived here (View is a value receiver and must not mutate); the actual
	// nil-out happens lazily in Update on the next message.
	if m.transitionActive(m.animNowMs) {
		p := anim.EaseInOutQuad(m.transition.progress(m.animNowMs))
		noColor := m.theme.Color(theme.RoleBg) == nil
		out = renderTransition(m.transition.fromFrame, out, p, noColor)
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

	return altView(out)
}

// composeScreen renders a single screen to its final placed frame. Home/Result/
// Settings/History/CodePaste self-place to w×h inside their own View(); Typing
// and the placeholder are placed here. Used both for the live View and to
// snapshot the outgoing frame when starting a transition.
func (m Model) composeScreen(screen Screen) string {
	switch screen {
	case ScreenHome:
		return m.home.View()
	case ScreenResult:
		return m.result.View()
	case ScreenSettings:
		return m.sett.View()
	case ScreenHistory:
		return m.hist.View()
	case ScreenCodePaste:
		return m.codePaste.View()
	case ScreenTyping:
		out := m.typing.View()
		if m.w > 0 && m.h > 0 {
			out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
		}
		return out
	default:
		out := placeholderView(screen, m.theme)
		if m.w > 0 && m.h > 0 {
			out = lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, out)
		}
		return out
	}
}
