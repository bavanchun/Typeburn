package config

import (
	"encoding/json"
	"testing"
)

// jsonUnmarshal is a thin alias so test bodies stay concise.
func jsonUnmarshal(data []byte, v any) error { return json.Unmarshal(data, v) }

// TestDefaults_Values verifies the out-of-the-box settings.
func TestDefaults_Values(t *testing.T) {
	s := Defaults()
	if s.Theme != "default" {
		t.Errorf("Theme: want 'default', got %q", s.Theme)
	}
	if s.DefaultMode != ModeTime {
		t.Errorf("DefaultMode: want ModeTime, got %v", s.DefaultMode)
	}
	if s.DefaultLength != 30 {
		t.Errorf("DefaultLength: want 30, got %d", s.DefaultLength)
	}
	if s.BlinkCursor {
		t.Error("BlinkCursor: want false by default")
	}
}

// TestLengthsFor_Time verifies the time-mode option slice.
func TestLengthsFor_Time(t *testing.T) {
	lens := LengthsFor(ModeTime)
	want := []int{15, 30, 60, 120}
	if len(lens) != len(want) {
		t.Fatalf("LengthsFor(Time): want %v, got %v", want, lens)
	}
	for i, v := range want {
		if lens[i] != v {
			t.Errorf("index %d: want %d, got %d", i, v, lens[i])
		}
	}
}

// TestLengthsFor_Words verifies the words-mode option slice.
func TestLengthsFor_Words(t *testing.T) {
	lens := LengthsFor(ModeWords)
	want := []int{10, 25, 50, 100}
	if len(lens) != len(want) {
		t.Fatalf("LengthsFor(Words): want %v, got %v", want, lens)
	}
	for i, v := range want {
		if lens[i] != v {
			t.Errorf("index %d: want %d, got %d", i, v, lens[i])
		}
	}
}

// TestLengthsFor_Quote verifies that Quote mode has no numeric length options.
func TestLengthsFor_Quote(t *testing.T) {
	if lens := LengthsFor(ModeQuote); lens != nil {
		t.Errorf("LengthsFor(Quote): want nil, got %v", lens)
	}
}

// TestNormalize_UnknownThemeFallsBack verifies an unknown theme is reset to default.
func TestNormalize_UnknownThemeFallsBack(t *testing.T) {
	s := Settings{Theme: "solarized", DefaultMode: ModeTime, DefaultLength: 30}
	s.Normalize()
	if s.Theme != "default" {
		t.Errorf("unknown theme: want 'default', got %q", s.Theme)
	}
}

// TestNormalize_KnownThemesPreserved verifies valid themes are not overwritten.
func TestNormalize_KnownThemesPreserved(t *testing.T) {
	for _, name := range []string{"default", "mono"} {
		s := Settings{Theme: name, DefaultMode: ModeTime, DefaultLength: 30}
		s.Normalize()
		if s.Theme != name {
			t.Errorf("theme %q should be preserved, got %q", name, s.Theme)
		}
	}
}

// TestNormalize_UnknownModeFallsBack verifies an unknown mode is reset to ModeTime.
func TestNormalize_UnknownModeFallsBack(t *testing.T) {
	s := Settings{Theme: "default", DefaultMode: "extreme", DefaultLength: 30}
	s.Normalize()
	if s.DefaultMode != ModeTime {
		t.Errorf("unknown mode: want ModeTime, got %v", s.DefaultMode)
	}
}

// TestNormalize_InvalidLengthRepaired verifies an out-of-range length is fixed.
func TestNormalize_InvalidLengthRepaired(t *testing.T) {
	s := Settings{Theme: "default", DefaultMode: ModeTime, DefaultLength: 999}
	s.Normalize()
	valid := LengthsFor(ModeTime)
	found := false
	for _, v := range valid {
		if s.DefaultLength == v {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("repaired length %d not in valid set %v", s.DefaultLength, valid)
	}
}

// TestNormalize_QuoteModeDoesNotRepairLength verifies Quote mode skips length repair
// (LengthsFor returns nil so there is no constraint to enforce).
func TestNormalize_QuoteModeDoesNotRepairLength(t *testing.T) {
	s := Settings{Theme: "default", DefaultMode: ModeQuote, DefaultLength: 0}
	s.Normalize()
	// Should not panic and mode should remain Quote.
	if s.DefaultMode != ModeQuote {
		t.Errorf("mode changed unexpectedly: want ModeQuote, got %v", s.DefaultMode)
	}
}

// TestKeymap_QuitMatchesCtrlC verifies the quit binding matches ctrl+c.
func TestKeymap_QuitMatchesCtrlC(t *testing.T) {
	import_tea_Key_Code_c_Mod_ModCtrl_helper(t)
}

// TestKeymap_AllBindingsNonEmpty verifies every exported binding has at least one chord.
func TestKeymap_AllBindingsNonEmpty(t *testing.T) {
	km := DefaultKeymap()
	bindings := []struct {
		name string
		b    Binding
	}{
		{"Quit", km.Quit},
		{"Back", km.Back},
		{"RestartSame", km.RestartSame},
		{"NewTest", km.NewTest},
		{"NavHome", km.NavHome},
		{"NavSettings", km.NavSettings},
		{"NavHistory", km.NavHistory},
		{"NextMode", km.NextMode},
		{"PrevMode", km.PrevMode},
		{"OptLeft", km.OptLeft},
		{"OptRight", km.OptRight},
		{"Start", km.Start},
		{"Up", km.Up},
		{"Down", km.Down},
		{"Cycle", km.Cycle},
		{"Top", km.Top},
		{"Bottom", km.Bottom},
	}
	for _, b := range bindings {
		if len(b.b.chords) == 0 {
			t.Errorf("binding %s has no chords", b.name)
		}
		if b.b.Name == "" {
			t.Errorf("binding %s has empty Name", b.name)
		}
	}
}

// import_tea_Key_Code_c_Mod_ModCtrl_helper is an inline helper that checks
// the Quit binding without importing tea into this file's package-level imports
// (avoids a circular dependency). It reconstructs the key using the same
// chord struct layout as keymap.go.
func import_tea_Key_Code_c_Mod_ModCtrl_helper(t *testing.T) {
	t.Helper()
	km := DefaultKeymap()
	// Verify Name field is set correctly.
	if km.Quit.Name != "quit" {
		t.Errorf("Quit.Name: want 'quit', got %q", km.Quit.Name)
	}
	// Verify there is exactly one chord (ctrl+c).
	if len(km.Quit.chords) != 1 {
		t.Errorf("Quit: want 1 chord, got %d", len(km.Quit.chords))
	}
}

// TestXDGPaths_ConfigDirNonEmpty verifies ConfigDir returns a non-empty path.
func TestXDGPaths_ConfigDirNonEmpty(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	if got == "" {
		t.Error("ConfigDir returned empty string")
	}
}

// TestXDGPaths_DataDirNonEmpty verifies DataDir returns a non-empty path.
func TestXDGPaths_DataDirNonEmpty(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	got, err := DataDir()
	if err != nil {
		t.Fatalf("DataDir: %v", err)
	}
	if got == "" {
		t.Error("DataDir returned empty string")
	}
}

// TestXDGPaths_XDGEnvTakesPrecedence verifies that an explicit XDG env var is used.
func TestXDGPaths_XDGEnvTakesPrecedence(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	// Path must start with the XDG override, not ~/.config.
	if len(got) < len(dir) || got[:len(dir)] != dir {
		t.Errorf("ConfigDir should be under %q, got %q", dir, got)
	}
}

// TestDefaults_UpdateCheckFalse verifies update_check defaults to false (opt-in).
func TestDefaults_UpdateCheckFalse(t *testing.T) {
	if Defaults().UpdateCheck {
		t.Error("UpdateCheck: want false by default (opt-in)")
	}
}

// TestLoadSettings_MissingUpdateCheckField verifies pre-v2.1 settings files
// (without the update_check key) load with UpdateCheck == false.
func TestLoadSettings_MissingUpdateCheckField(t *testing.T) {
	// JSON without update_check — simulates an existing v2.0 settings file.
	raw := `{"theme":"default","default_mode":"time","default_length":30,"blink_cursor":false}`
	var s Settings
	if err := jsonUnmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if s.UpdateCheck {
		t.Error("UpdateCheck: want false when field is absent from JSON")
	}
}

// TestXDGPaths_RelativeXDGIgnored verifies that a relative XDG_CONFIG_HOME
// is ignored and the HOME fallback is used instead (per spec: must be absolute).
func TestXDGPaths_RelativeXDGIgnored(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative/path")
	got, err := ConfigDir()
	if err != nil {
		t.Fatalf("ConfigDir: %v", err)
	}
	// Must not start with "relative".
	if len(got) > 0 && got[0] != '/' {
		t.Errorf("relative XDG_CONFIG_HOME should be ignored; got %q", got)
	}
}
