package theme

import (
	"reflect"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
)

func TestNamesSyncWithAvailableAndSettingsNormalize(t *testing.T) {
	if !reflect.DeepEqual(Names(), Available()) {
		t.Fatalf("Names and Available diverged: %#v != %#v", Names(), Available())
	}
	for _, name := range Names() {
		s := config.Defaults()
		s.Theme = name
		s.Normalize()
		if s.Theme != name {
			t.Fatalf("Settings.Normalize rejected theme %q", name)
		}
	}
	s := config.Defaults()
	s.Theme = "not-a-theme"
	s.Normalize()
	if s.Theme != "default" {
		t.Fatalf("unknown theme should normalize to default, got %q", s.Theme)
	}
}
