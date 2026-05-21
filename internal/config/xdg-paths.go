package config

import (
	"os"
	"path/filepath"
)

// appDir is the per-app subdirectory used under the XDG roots.
const appDir = "typeburn"

// ConfigDir returns the directory for the settings file:
// $XDG_CONFIG_HOME/typeburn, falling back to ~/.config/typeburn.
// The directory is not created here; the storage layer creates it on write.
func ConfigDir() (string, error) {
	return resolveDir("XDG_CONFIG_HOME", ".config")
}

// DataDir returns the directory for the history file:
// $XDG_DATA_HOME/typeburn, falling back to ~/.local/share/typeburn.
func DataDir() (string, error) {
	return resolveDir("XDG_DATA_HOME", filepath.Join(".local", "share"))
}

// StateDir returns the directory for runtime state (e.g. update-check cache):
// $XDG_STATE_HOME/typeburn, falling back to ~/.local/state/typeburn.
func StateDir() (string, error) {
	return resolveDir("XDG_STATE_HOME", filepath.Join(".local", "state"))
}

// resolveDir prefers an absolute XDG env var; otherwise it joins the user's
// home with the conventional fallback subpath. macOS has no XDG vars by
// default, so the HOME fallback is the normal path there.
func resolveDir(env, homeRel string) (string, error) {
	if v := os.Getenv(env); filepath.IsAbs(v) {
		return filepath.Join(v, appDir), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, homeRel, appDir), nil
}
