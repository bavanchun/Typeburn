package app

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
	"github.com/bavanchun/Typeburn/internal/update"
)

// NewFromDisk builds the root model loading persisted settings from disk.
// Falls back to config.Defaults() if the file is missing or corrupt.
// codeText is the loaded snippet (empty string = no code mode); codeHint is
// a user-facing reason string when loading failed (empty = no error);
// updateHint is non-nil when an opportunistic check found a newer release.
func NewFromDisk(codeText, codeHint string, updateHint *update.Result) Model {
	s := storage.LoadSettings()
	th := theme.Load(s.Theme, theme.EnvNoColor())
	return New(th, s, codeText, codeHint, updateHint)
}

// applySettings applies a settings change to the LIVE model the program
// renders: persist atomically, rebuild the theme, and re-inject into every
// sub-model so theme swap, blink toggle, and default mode/length take effect
// immediately. It is invoked from Update on receipt of ui.SettingsChangedMsg
// (value receiver → the returned copy is what Bubble Tea continues to drive,
// unlike the previous pointer callback which mutated an orphaned New() copy).
// The Settings selection index is preserved so the user stays on the row they
// just changed.
func (m Model) applySettings(s config.Settings) Model {
	m.settings = s
	// Persist best-effort: a disk failure must not crash the UI, but it must
	// not be silent either — surface a dismissible notice.
	if err := storage.SaveSettings(s); err != nil {
		m.persistErr = "Couldn't save settings to disk"
	}
	// Rebuild theme from the new name, then propagate to every sub-model.
	m.theme = theme.Load(s.Theme, theme.EnvNoColor())
	// Preserve the loaded code text and hint across settings changes.
	m.home = m.home.WithSettings(m.theme, m.keys).SetSize(m.w, m.h)
	m.typing = m.typing.ApplySettings(s, m.theme)
	m.result = m.result.ApplyTheme(m.theme)
	sel := m.sett.Sel()
	m.sett = ui.NewSettings(m.settings, m.theme, m.keys).WithSel(sel).SetSize(m.w, m.h)
	return m
}
