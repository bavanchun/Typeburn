package theme

import (
	"image/color"
	"os"

	"charm.land/lipgloss/v2"
)

// Theme resolves semantic Roles to concrete styles. Under NO_COLOR the
// color map is ignored and roles map to text attributes only, so layout is
// identical with or without color.
type Theme struct {
	name    string
	colors  map[Role]color.Color
	noColor bool
}

// Name returns the theme identifier ("default", "mono", or "no-color").
func (t Theme) Name() string { return t.name }

// Available lists the user-selectable theme names in v1 order.
// "solarized-dark" is reserved but not implemented in v1.
func Available() []string { return []string{"default", "mono"} }

// EnvNoColor reports whether the NO_COLOR convention is active. Any non-empty
// value disables color (https://no-color.org).
func EnvNoColor() bool { return os.Getenv("NO_COLOR") != "" }

// Load builds a theme by name. Unknown names fall back to the default theme.
// When noColor is true an attribute-only theme is returned regardless of name.
func Load(name string, noColor bool) Theme {
	if noColor {
		return noColorTheme()
	}
	switch name {
	case "mono":
		return Mono()
	default:
		return Default()
	}
}

// Style returns the lipgloss style for a role. With color it applies the
// mapped foreground (and background for the few bg roles). Without color it
// applies an attribute that keeps the role distinguishable in monochrome.
func (t Theme) Style(r Role) lipgloss.Style {
	s := lipgloss.NewStyle()
	if t.noColor {
		return attrOnlyStyle(s, r)
	}
	switch r {
	case RoleErrorBg:
		return s.Foreground(t.colors[RoleError]).
			Background(t.colors[RoleErrorBg]).Underline(true)
	case RoleCursorBg:
		return s.Foreground(t.colors[RoleCursorFg]).
			Background(t.colors[RoleCursorBg])
	case RoleError:
		// Never color-alone: errors also carry an underline.
		return s.Foreground(t.colors[RoleError]).Underline(true)
	default:
		return s.Foreground(t.colors[r])
	}
}

// Color exposes the raw color for a role (for callers that compose their own
// styles, e.g. borders). Returns nil under NO_COLOR.
func (t Theme) Color(r Role) color.Color {
	if t.noColor {
		return nil
	}
	return t.colors[r]
}

// attrOnlyStyle maps a role to a text attribute so the UI stays legible with
// no color: the cursor reverses, errors underline, accents bold, faint stays
// faint. Layout is unaffected.
func attrOnlyStyle(s lipgloss.Style, r Role) lipgloss.Style {
	switch r {
	case RoleCursorBg:
		return s.Reverse(true)
	case RoleError, RoleErrorBg:
		return s.Underline(true)
	case RoleAccent, RoleSuccess, RoleBorderFocus:
		return s.Bold(true)
	case RoleTextFaint:
		return s.Faint(true)
	default:
		return s
	}
}
