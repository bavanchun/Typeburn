package config_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// TestThemeAvailableStaysInSyncWithNormalize guards the intentional
// duplication between theme.Available() and config.Settings.Normalize's
// accepted theme set (the core layering rule forbids config importing theme,
// so the list cannot be shared directly). External test package can import
// both. If a theme is added to Available() but not to Normalize's switch,
// Normalize would reset it to "default" and this test fails.
func TestThemeAvailableStaysInSyncWithNormalize(t *testing.T) {
	for _, name := range theme.Available() {
		s := config.Settings{Theme: name}
		s.Normalize()
		if s.Theme != name {
			t.Errorf("theme %q is in theme.Available() but Normalize() reset it "+
				"to %q — add it to the switch in config/settings.go", name, s.Theme)
		}
	}

	// Unknown names must still fall back to the default.
	s := config.Settings{Theme: "definitely-not-a-real-theme"}
	s.Normalize()
	if s.Theme != "default" {
		t.Errorf("unknown theme: want fallback to \"default\", got %q", s.Theme)
	}
}
