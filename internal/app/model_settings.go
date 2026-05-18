package app

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/storage"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
)

// NewFromDisk builds the root model loading persisted settings from disk.
// Falls back to config.Defaults() if the file is missing or corrupt.
func NewFromDisk() Model {
	s := storage.LoadSettings()
	th := theme.Load(s.Theme, theme.EnvNoColor())
	return New(th, s)
}

// onSettingsChange is the onChange callback wired into SettingsModel.
// It persists atomically, rebuilds the theme, and re-injects into all sub-models
// so that theme swap, blink toggle, and default mode/length apply live.
func (m *Model) onSettingsChange(s config.Settings) {
	m.settings = s
	// Persist best-effort: a disk failure must not crash the UI, but it must
	// not be silent either — surface a dismissible notice.
	if err := storage.SaveSettings(s); err != nil {
		m.persistErr = "Couldn't save settings to disk"
	}
	// Rebuild theme from the new name, then propagate to every sub-model.
	m.theme = theme.Load(s.Theme, theme.EnvNoColor())
	m.home = ui.NewHome(s, m.theme, m.keys).SetSize(m.w, m.h)
	m.typing = m.typing.ApplySettings(s, m.theme)
	m.result = m.result.ApplyTheme(m.theme)
	// Re-create SettingsModel so the live theme is visible in the settings view.
	m.sett = ui.NewSettings(&m.settings, m.theme, m.keys, m.onSettingsChange).SetSize(m.w, m.h)
}
