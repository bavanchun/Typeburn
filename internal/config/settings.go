// Package config holds user settings, key bindings, and platform paths.
// It has no UI or persistence dependencies; storage (Phase 7) marshals these
// types to disk.
package config

import "github.com/bavanchun/Typeburn/internal/mode"

// Mode identifies a test mode. Stored as a string for forward-compatible JSON.
type Mode = mode.Mode

const (
	ModeTime  = mode.ModeTime
	ModeWords = mode.ModeWords
	ModeQuote = mode.ModeQuote
	ModeCode  = mode.ModeCode
)

// LengthsFor returns the selectable length options for a mode. Quote mode has
// no numeric length (the quote itself bounds the test) so it returns nil.
func LengthsFor(m Mode) []int {
	return mode.LengthsFor(m)
}

// Settings is the persisted user configuration.
type Settings struct {
	Theme         string `json:"theme"`          // one of theme.Available()
	DefaultMode   Mode   `json:"default_mode"`   // seeds the Home screen
	DefaultLength int    `json:"default_length"` // valid for DefaultMode
	BlinkCursor   bool   `json:"blink_cursor"`   // typing-screen cursor blink
	// UpdateCheck enables an opt-in opportunistic update check on TUI launch.
	// Default is false to preserve offline-first posture.
	UpdateCheck bool `json:"update_check"`
	StrictMode  bool `json:"strict_mode"`
}

// Defaults returns the baseline configuration used on first run or whenever
// the settings file is missing or unreadable.
func Defaults() Settings {
	return Settings{
		Theme:         "default",
		DefaultMode:   ModeTime,
		DefaultLength: 30,
		BlinkCursor:   false,
		UpdateCheck:   false,
		StrictMode:    false,
	}
}

// Normalize repairs out-of-range or unknown values in place, keeping a loaded
// (possibly hand-edited or stale) settings file safe to use.
func (s *Settings) Normalize() {
	// Intentional duplication of theme.Available(): the core layering rule
	// forbids importing internal/theme here. theme_available_sync_test.go
	// asserts this accepted set stays in lockstep with theme.Available().
	switch s.Theme {
	case "default", "mono",
		"solarized-dark", "solarized-light",
		"dracula", "nord",
		"gruvbox-dark", "gruvbox-light":
	default:
		s.Theme = "default"
	}
	switch s.DefaultMode {
	case ModeTime, ModeWords, ModeQuote, ModeCode:
	default:
		s.DefaultMode = ModeTime
	}
	if lens := LengthsFor(s.DefaultMode); lens != nil && !containsInt(lens, s.DefaultLength) {
		s.DefaultLength = lens[len(lens)/2-1] // sensible mid default (30 / 25)
	}
}

func containsInt(xs []int, v int) bool {
	for _, x := range xs {
		if x == v {
			return true
		}
	}
	return false
}
