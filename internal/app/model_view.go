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

	// Transient notice (a failed write, or a run withheld from history).
	// Cleared on the next keypress.
	if m.persistErr != "" && m.w > 0 && m.h > 0 {
		out = overlayNotice(out, ui.PersistenceNotice(m.persistErr, m.w, m.theme), m.w, m.h)
	}

	return altView(out)
}

// overlayNotice puts a one-line notice on the last row the terminal actually
// shows, which is not the same as the frame's last line.
//
// Writing to the frame's last line is wrong in both directions. When the frame
// is taller than the terminal that line is clipped away, so the notice is
// invisible in exactly the situation that produced it. When the frame is
// shorter, the last line belongs to the screen and overwriting it destroys
// content the user needs.
//
// So the notice takes the chosen row only when that row is blank padding, and
// is otherwise inserted above it, pushing the frame down by one. The one
// exception is a frame that already exceeds the terminal: a pushed-down row
// falls off the bottom regardless, and lengthening the frame further would
// only widen the overflow.
func overlayNotice(frame, notice string, w, h int) string {
	lines := strings.Split(frame, "\n")
	row := len(lines) - 1
	if h-1 < row {
		row = h - 1
	}
	if row < 0 {
		return frame
	}

	placed := lipgloss.PlaceHorizontal(w, lipgloss.Center, notice)
	if strings.TrimSpace(lines[row]) == "" || len(lines) > h {
		lines[row] = placed
	} else {
		lines = append(lines[:row], append([]string{placed}, lines[row:]...)...)
	}
	return strings.Join(lines, "\n")
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
