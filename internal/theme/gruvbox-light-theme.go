package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// GruvboxLight is the Gruvbox light palette (https://github.com/morhetz/gruvbox)
// mapped onto the 16 roles: bg0 light ground, dark-yellow accent. Light-bg
// role contrast reviewed across all screens; see docs/design-guidelines.md
// §2.1.
func GruvboxLight() Theme {
	return Theme{
		name: "gruvbox-light",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#fbf1c7"),
			RoleSurface:     lipgloss.Color("#ebdbb2"),
			RoleSurfaceAlt:  lipgloss.Color("#d5c4a1"),
			RoleTextPrimary: lipgloss.Color("#3c3836"),
			RoleTextMuted:   lipgloss.Color("#7c6f64"),
			RoleTextFaint:   lipgloss.Color("#bdae93"),
			RoleAccent:      lipgloss.Color("#b57614"),
			RoleAccentDim:   lipgloss.Color("#79740e"),
			RoleError:       lipgloss.Color("#9d0006"),
			RoleErrorBg:     lipgloss.Color("#f2d0cd"),
			RoleWarning:     lipgloss.Color("#af3a03"),
			RoleSuccess:     lipgloss.Color("#79740e"),
			RoleCursorBg:    lipgloss.Color("#b57614"),
			RoleCursorFg:    lipgloss.Color("#fbf1c7"),
			RoleBorder:      lipgloss.Color("#bdae93"),
			RoleBorderFocus: lipgloss.Color("#b57614"),
		},
	}
}
