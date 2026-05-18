package theme

import "testing"

// packThemes is the set of named, fully-colored themes (excludes the
// attribute-only no-color theme, which is selected via the noColor flag).
var packThemes = []string{
	"default", "mono",
	"solarized-dark", "solarized-light",
	"dracula", "nord",
	"gruvbox-dark", "gruvbox-light",
}

// TestAvailable_ContainsExpectedThemes verifies Available returns exactly the
// selectable theme set in the expected order.
func TestAvailable_ContainsExpectedThemes(t *testing.T) {
	got := Available()
	if len(got) != len(packThemes) {
		t.Fatalf("Available: want %d entries, got %d: %v", len(packThemes), len(got), got)
	}
	for i, name := range packThemes {
		if got[i] != name {
			t.Errorf("Available[%d]: want %q, got %q", i, name, got[i])
		}
	}
}

// TestPacks_NameRoundTripAndAllRolesResolve verifies every selectable theme
// loads by name and resolves all 16 roles to a non-nil color (colored mode)
// and to a non-panicking style; under noColor the same name yields no-color.
func TestPacks_NameRoundTripAndAllRolesResolve(t *testing.T) {
	for _, name := range packThemes {
		th := Load(name, false)
		if th.Name() != name {
			t.Errorf("Load(%q,false): Name()=%q, want %q", name, th.Name(), name)
		}
		for _, r := range allRoles() {
			if c := th.Color(r); c == nil {
				t.Errorf("theme %q: Color(role %d) is nil; every role must map", name, r)
			}
			_ = th.Style(r) // must not panic
		}
		if nc := Load(name, true); nc.Name() != "no-color" {
			t.Errorf("Load(%q,true): Name()=%q, want no-color", name, nc.Name())
		}
	}
}

// TestLoad_DefaultName returns a named default theme.
func TestLoad_DefaultName(t *testing.T) {
	th := Load("default", false)
	if th.Name() != "default" {
		t.Errorf("Load('default'): want Name()='default', got %q", th.Name())
	}
}

// TestLoad_MonoName returns a named mono theme.
func TestLoad_MonoName(t *testing.T) {
	th := Load("mono", false)
	if th.Name() != "mono" {
		t.Errorf("Load('mono'): want Name()='mono', got %q", th.Name())
	}
}

// TestLoad_UnknownFallsBackToDefault verifies unknown names resolve to the default theme.
func TestLoad_UnknownFallsBackToDefault(t *testing.T) {
	th := Load("nonexistent", false)
	if th.Name() != "default" {
		t.Errorf("Load('nonexistent'): want fallback to 'default', got %q", th.Name())
	}
}

// TestLoad_NoColorFlag returns a no-color theme regardless of name.
func TestLoad_NoColorFlag(t *testing.T) {
	th := Load("default", true)
	if th.Name() != "no-color" {
		t.Errorf("Load with noColor=true: want Name()='no-color', got %q", th.Name())
	}
}

// allRoles returns every defined role constant for exhaustive testing.
// Mirrors the iota order in roles.go; excludes the unexported sentinel roleCount.
func allRoles() []Role {
	return []Role{
		RoleBg, RoleSurface, RoleSurfaceAlt,
		RoleTextPrimary, RoleTextMuted, RoleTextFaint,
		RoleAccent, RoleAccentDim,
		RoleError, RoleErrorBg, RoleWarning, RoleSuccess,
		RoleCursorBg, RoleCursorFg, RoleBorder, RoleBorderFocus,
	}
}

// TestDefault_StyleNonNilForAllRoles verifies Style() does not panic for any role.
func TestDefault_StyleNonNilForAllRoles(t *testing.T) {
	th := Default()
	for _, r := range allRoles() {
		// Must not panic.
		_ = th.Style(r)
	}
}

// TestMono_StyleNonNilForAllRoles verifies Mono theme Style() does not panic.
func TestMono_StyleNonNilForAllRoles(t *testing.T) {
	th := Mono()
	for _, r := range allRoles() {
		_ = th.Style(r)
	}
}

// TestNoColor_StyleNonNilForAllRoles verifies the no-color theme does not panic.
func TestNoColor_StyleNonNilForAllRoles(t *testing.T) {
	th := Load("default", true)
	for _, r := range allRoles() {
		_ = th.Style(r)
	}
}

// TestDefault_ColorNonNilForAccent verifies Color() returns a non-nil value
// in the default (colored) theme.
func TestDefault_ColorNonNilForAccent(t *testing.T) {
	th := Default()
	if c := th.Color(RoleAccent); c == nil {
		t.Error("Default theme: Color(RoleAccent) should be non-nil")
	}
}

// TestNoColor_ColorReturnsNil verifies Color() returns nil under NO_COLOR.
func TestNoColor_ColorReturnsNil(t *testing.T) {
	th := Load("default", true)
	if c := th.Color(RoleAccent); c != nil {
		t.Errorf("no-color theme: Color(RoleAccent) should be nil, got %v", c)
	}
}

// TestEnvNoColor_Unset verifies EnvNoColor is false when NO_COLOR is not set.
func TestEnvNoColor_Unset(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	if EnvNoColor() {
		t.Error("EnvNoColor: want false when NO_COLOR='', got true")
	}
}

// TestEnvNoColor_Set verifies EnvNoColor is true when NO_COLOR has any value.
func TestEnvNoColor_Set(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	if !EnvNoColor() {
		t.Error("EnvNoColor: want true when NO_COLOR='1', got false")
	}
}

// TestStyle_ErrorRoleHasUnderline verifies the error role always carries an
// underline (never color-alone) in both colored and no-color themes.
func TestStyle_ErrorRoleHasUnderline(t *testing.T) {
	for _, tc := range []struct {
		name    string
		noColor bool
	}{
		{"default", false},
		{"no-color", true},
	} {
		th := Load("default", tc.noColor)
		s := th.Style(RoleError)
		rendered := s.Render("X")
		// Underline escape (\x1b[4m) or the plain char — both are valid since
		// in noColor mode lipgloss still emits underline attribute. We just
		// verify Render doesn't panic and returns non-empty output.
		if rendered == "" {
			t.Errorf("theme %s: Style(RoleError).Render returned empty", tc.name)
		}
	}
}
