package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Nord is the official Nord palette (https://www.nordtheme.com/) mapped onto
// the 16 roles: nord0 polar-night ground, nord8 frost accent, aurora signal
// colors. Hex values pinned; see docs/design-guidelines.md §2.1.
func Nord() Theme {
	return Theme{
		name: "nord",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#2e3440"),
			RoleSurface:     lipgloss.Color("#3b4252"),
			RoleSurfaceAlt:  lipgloss.Color("#434c5e"),
			RoleTextPrimary: lipgloss.Color("#eceff4"),
			RoleTextMuted:   lipgloss.Color("#d8dee9"),
			RoleTextFaint:   lipgloss.Color("#4c566a"),
			RoleAccent:      lipgloss.Color("#88c0d0"),
			RoleAccentDim:   lipgloss.Color("#5e81ac"),
			RoleError:       lipgloss.Color("#bf616a"),
			RoleErrorBg:     lipgloss.Color("#4a2326"),
			RoleWarning:     lipgloss.Color("#ebcb8b"),
			RoleSuccess:     lipgloss.Color("#a3be8c"),
			RoleCursorBg:    lipgloss.Color("#88c0d0"),
			RoleCursorFg:    lipgloss.Color("#2e3440"),
			RoleBorder:      lipgloss.Color("#4c566a"),
			RoleBorderFocus: lipgloss.Color("#88c0d0"),
		},
	}
}
