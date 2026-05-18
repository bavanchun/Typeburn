package app

import (
	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/storage"
	"monkeytype-tui/internal/theme"
	"monkeytype-tui/internal/ui"
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
	// Persist best-effort; ignore error to avoid crashing the UI on disk issues.
	_ = storage.SaveSettings(s)
	// Rebuild theme from the new name, then propagate to every sub-model.
	m.theme = theme.Load(s.Theme, theme.EnvNoColor())
	m.home = ui.NewHome(s, m.theme, m.keys).SetSize(m.w, m.h)
	m.typing = m.typing.ApplySettings(s, m.theme)
	m.result = m.result.ApplyTheme(m.theme)
	// Re-create SettingsModel so the live theme is visible in the settings view.
	m.sett = ui.NewSettings(&m.settings, m.theme, m.keys, m.onSettingsChange).SetSize(m.w, m.h)
}
