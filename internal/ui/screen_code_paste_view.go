package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// View renders the paste screen as a centered block string. Like Home/Settings
// it self-places to w×h. The line structure is identical in every state and
// under NO_COLOR (theme Role styling only, no layout branching on color):
// title, instruction, then one status line (waiting or the error reason).
func (m CodePasteModel) View() string {
	title := m.th.Style(theme.RoleAccent).Bold(true).Render("P A S T E   C O D E")
	instr := m.th.Style(theme.RoleTextPrimary).Render("Paste your snippet · esc to cancel")

	var status string
	if m.errMsg != "" {
		status = m.th.Style(theme.RoleWarning).Render(m.errMsg + " · paste again")
	} else {
		status = m.th.Style(theme.RoleTextFaint).Render("waiting for paste…")
	}

	var b strings.Builder
	b.WriteString(title)
	b.WriteString("\n\n")
	b.WriteString(instr)
	b.WriteString("\n\n")
	b.WriteString(status)

	content := b.String()
	if m.w > 0 && m.h > 0 {
		return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}
