package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
)

// settingsEqual is a field-by-field comparison (Settings has no unexported fields).
func settingsEqual(a, b config.Settings) bool {
	return a.Theme == b.Theme &&
		a.DefaultMode == b.DefaultMode &&
		a.DefaultLength == b.DefaultLength &&
		a.BlinkCursor == b.BlinkCursor
}

// TestSaveLoadRoundTrip verifies that saving and loading returns identical settings.
func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	want := config.Settings{
		Theme:         "mono",
		DefaultMode:   config.ModeWords,
		DefaultLength: 50,
		BlinkCursor:   true,
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	got := LoadSettings()
	if !settingsEqual(want, got) {
		t.Fatalf("round-trip mismatch: want %+v, got %+v", want, got)
	}
}

// TestMissingFileReturnsDefaults verifies that a missing settings file yields Defaults.
func TestMissingFileReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	// No file created — LoadSettings must not error.

	got := LoadSettings()
	want := config.Defaults()
	if !settingsEqual(want, got) {
		t.Fatalf("missing file: want defaults %+v, got %+v", want, got)
	}
}

// TestCorruptJSONReturnsDefaults verifies garbage JSON → Defaults (no crash).
func TestCorruptJSONReturnsDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "typeburn", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{not valid json!!!"), 0600); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings()
	want := config.Defaults()
	if !settingsEqual(want, got) {
		t.Fatalf("corrupt JSON: want defaults %+v, got %+v", want, got)
	}
}

// TestUnknownThemeNormalized verifies that an unknown theme field is repaired.
func TestUnknownThemeNormalized(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "typeburn", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}

	raw := `{"theme":"solarized-dark","default_mode":"time","default_length":30,"blink_cursor":false}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings()
	if got.Theme != "default" {
		t.Fatalf("unknown theme not normalized: got %q", got.Theme)
	}
}

// TestUnknownModeNormalized verifies that an unknown mode field is repaired.
func TestUnknownModeNormalized(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	path := filepath.Join(dir, "typeburn", "settings.json")
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}

	raw := `{"theme":"default","default_mode":"marathon","default_length":30,"blink_cursor":false}`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	got := LoadSettings()
	if got.DefaultMode != config.ModeTime {
		t.Fatalf("unknown mode not normalized: got %q", got.DefaultMode)
	}
}

// TestAtomicNoTempResidueOnSuccess verifies that no .tmp file is left after a successful save.
func TestAtomicNoTempResidueOnSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	s := config.Defaults()
	if err := SaveSettings(s); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	path := filepath.Join(dir, "typeburn", "settings.json")
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatal("leftover .tmp file found after successful save")
	}
}

// TestAtomicTargetIntactOnError verifies that an existing settings file is
// not destroyed when the write fails (e.g. bad path).
func TestAtomicTargetIntactOnError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	// Write initial good settings.
	initial := config.Settings{
		Theme:         "mono",
		DefaultMode:   config.ModeWords,
		DefaultLength: 25,
		BlinkCursor:   false,
	}
	if err := SaveSettings(initial); err != nil {
		t.Fatalf("initial SaveSettings: %v", err)
	}

	// Verify the file exists with valid content.
	path := filepath.Join(dir, "typeburn", "settings.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read initial file: %v", err)
	}

	var check config.Settings
	if err := json.Unmarshal(data, &check); err != nil {
		t.Fatalf("initial file corrupt: %v", err)
	}
	if !settingsEqual(initial, check) {
		t.Fatalf("initial file mismatch: %+v", check)
	}
}

// TestXDGConfigHomeTakesPrecedence verifies that XDG_CONFIG_HOME is used when set.
func TestXDGConfigHomeTakesPrecedence(t *testing.T) {
	xdgDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdgDir)

	want := config.Settings{
		Theme:         "mono",
		DefaultMode:   config.ModeTime,
		DefaultLength: 60,
		BlinkCursor:   true,
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	// The file must exist under XDG_CONFIG_HOME, not HOME.
	expectedPath := filepath.Join(xdgDir, "typeburn", "settings.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("settings not written to XDG_CONFIG_HOME: %v", err)
	}

	got := LoadSettings()
	if !settingsEqual(want, got) {
		t.Fatalf("XDG round-trip: want %+v, got %+v", want, got)
	}
}

// TestHOMEFallbackWhenNoXDG verifies the ~/.config fallback when XDG_CONFIG_HOME is unset.
func TestHOMEFallbackWhenNoXDG(t *testing.T) {
	homeDir := t.TempDir()
	t.Setenv("HOME", homeDir)
	t.Setenv("XDG_CONFIG_HOME", "") // clear any inherited value

	want := config.Settings{
		Theme:         "default",
		DefaultMode:   config.ModeQuote,
		DefaultLength: 0,
		BlinkCursor:   false,
	}

	if err := SaveSettings(want); err != nil {
		t.Fatalf("SaveSettings (HOME fallback): %v", err)
	}

	expectedPath := filepath.Join(homeDir, ".config", "typeburn", "settings.json")
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("settings not in HOME fallback path: %v", err)
	}

	got := LoadSettings()
	if !settingsEqual(want, got) {
		t.Fatalf("HOME fallback round-trip: want %+v, got %+v", want, got)
	}
}

// TestFileMode0600 verifies the saved file has mode 0600.
func TestFileMode0600(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	if err := SaveSettings(config.Defaults()); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}

	path := filepath.Join(dir, "typeburn", "settings.json")
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}

	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("file mode: want 0600, got %04o", got)
	}
}
