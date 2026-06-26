package config

import (
	"testing"
)

func TestDefaults_StrictMode(t *testing.T) {
	s := Defaults()
	if s.StrictMode {
		t.Error("StrictMode: want false by default")
	}
}

func TestLoadSettings_MissingStrictModeField(t *testing.T) {
	// JSON without strict_mode — simulates legacy settings.
	raw := `{"theme":"default","default_mode":"time","default_length":30,"blink_cursor":false}`
	var s Settings
	if err := jsonUnmarshal([]byte(raw), &s); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}
	if s.StrictMode {
		t.Error("StrictMode: want false when field is absent from JSON")
	}
}
