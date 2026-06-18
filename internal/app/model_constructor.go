package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
	"github.com/bavanchun/Typeburn/internal/update"
)

// New builds the root model with the given theme, settings, optional code
// text/hint, and an optional update hint.
// codeText is the snippet loaded from --text (empty = Code disabled);
// codeHint is a user-facing load-failure reason (empty = no error);
// updateHint is non-nil when an opportunistic check found a newer release.
func New(th theme.Theme, settings config.Settings, codeText, codeHint string, updateHint *update.Result) Model {
	km := config.DefaultKeymap()
	home := ui.NewHome(settings, th, km, codeText, codeHint)
	m := Model{
		screen:     ScreenHome,
		theme:      th,
		keys:       km,
		settings:   settings,
		home:       home,
		codeText:   codeText,
		codeHint:   codeHint,
		updateHint: updateHint,
	}
	m.sett = ui.NewSettings(settings, th, km)
	return m
}

// Init performs no startup command.
func (m Model) Init() tea.Cmd { return nil }
