package app

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// quitPromptModel is a minimal inline overlay rendered on the Home screen when
// the user presses esc. It presents a "quit? yes / no" choice.
//
// Design: minimalist per the app's ethos — a single centered line with two
// selectable options. `y` or enter-on-yes → tea.Quit; `n`, esc, or
// enter-on-no → dismiss. ctrl+c hard-quits anywhere (handled by root before
// this is consulted).
type quitPromptModel struct {
	// sel: 0 = yes, 1 = no
	sel int
}

// newQuitPrompt returns a quitPromptModel with "no" pre-selected (safer default).
func newQuitPrompt() quitPromptModel {
	return quitPromptModel{sel: 1}
}

// handleKey processes a key event inside the quit prompt.
// Returns (dismissed bool, quit bool) so the root can decide next action.
func (q quitPromptModel) handleKey(k tea.Key) (updated quitPromptModel, dismissed bool, quit bool) {
	// ctrl+c is handled by root before we get here; y/n/enter/esc handled below.
	switch {
	case k.Code == 'y' || k.Text == "y":
		return q, false, true
	case k.Code == 'n' || k.Text == "n":
		return q, true, false
	case k.Code == tea.KeyEsc:
		return q, true, false
	case k.Code == tea.KeyLeft || k.Code == 'h':
		q.sel = 0
	case k.Code == tea.KeyRight || k.Code == 'l':
		q.sel = 1
	case k.Code == tea.KeyEnter:
		if q.sel == 0 {
			return q, false, true // yes selected
		}
		return q, true, false // no selected
	}
	return q, false, false
}

// view renders the quit-prompt overlay as a centered block. It is placed by the
// root View over the current screen content.
func (q quitPromptModel) view(w, h int, th theme.Theme) string {
	labelStyle := th.Style(theme.RoleTextPrimary)
	accentStyle := th.Style(theme.RoleAccent).Bold(true)
	mutedStyle := th.Style(theme.RoleTextMuted)
	faintStyle := th.Style(theme.RoleTextFaint)

	question := labelStyle.Render("quit Typeburn?")

	var yesPart, noPart string
	if q.sel == 0 {
		yesPart = accentStyle.Render("▎ yes")
		noPart = "  " + mutedStyle.Render("no")
	} else {
		yesPart = "  " + mutedStyle.Render("yes")
		noPart = accentStyle.Render("▎ no")
	}

	options := yesPart + "   " + noPart
	hint := faintStyle.Render("←→ select · enter confirm · esc cancel")

	content := strings.Join([]string{question, "", options, "", hint}, "\n")

	if w > 0 && h > 0 {
		return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, content)
	}
	return content
}
