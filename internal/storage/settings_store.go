package storage

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/bavanchun/Typeburn/internal/config"
)

// SettingsPath returns the absolute path to the settings file:
// $XDG_CONFIG_HOME/typeburn/settings.json (fallback ~/.config/typeburn/).
func SettingsPath() (string, error) {
	dir, err := config.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "settings.json"), nil
}

// LoadSettings reads and unmarshals the settings file. On any error
// (missing file, parse failure, invalid enum values) it returns
// config.Defaults() so the app always starts in a valid state.
// This function never returns an error and never panics.
func LoadSettings() config.Settings {
	path, err := SettingsPath()
	if err != nil {
		return config.Defaults()
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// Missing file is expected on first run.
		return config.Defaults()
	}

	var s config.Settings
	if err := json.Unmarshal(data, &s); err != nil {
		// Corrupt or unreadable JSON — return safe defaults.
		return config.Defaults()
	}

	// Repair unknown enum values (e.g. from a future version or hand-edit).
	s.Normalize()
	return s
}

// SaveSettings atomically persists settings to the XDG config directory.
// It creates the config directory (mode 0700) if it does not exist.
// Returns an error only when the write itself fails; callers may ignore it
// without corrupting existing data (atomic rename prevents partial writes).
func SaveSettings(s config.Settings) error {
	path, err := SettingsPath()
	if err != nil {
		return err
	}

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return atomicWrite(path, data)
}
