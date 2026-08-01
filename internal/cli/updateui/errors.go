package updateui

import (
	"errors"

	tea "charm.land/bubbletea/v2"
)

// ErrCancelled reports that the user interrupted the update before the install
// stage began. The caller maps it to the process's abort exit code.
var ErrCancelled = errors.New("update cancelled")

// View renders the frame inline — no alternate screen. The block therefore
// stays in normal scrollback exactly where `typeburn update` printed it, and
// the surrounding command output is not wiped when the program exits.
func (m Model) View() tea.View {
	v := tea.NewView(m.Frame())
	v.AltScreen = false
	return v
}
