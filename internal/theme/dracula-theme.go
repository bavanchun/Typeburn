package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Dracula is the official Dracula palette (https://draculatheme.com/contribute)
// mapped onto the 16 roles: #282a36 ground, purple accent, pink/green/yellow
// signal colors. Hex values pinned; see docs/design-guidelines.md §2.1.
func Dracula() Theme {
	return Theme{
		name: "dracula",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#282a36"),
			RoleSurface:     lipgloss.Color("#343746"),
			RoleSurfaceAlt:  lipgloss.Color("#44475a"),
			RoleTextPrimary: lipgloss.Color("#f8f8f2"),
			RoleTextMuted:   lipgloss.Color("#c9c9d1"),
			RoleTextFaint:   lipgloss.Color("#6272a4"),
			RoleAccent:      lipgloss.Color("#bd93f9"),
			RoleAccentDim:   lipgloss.Color("#7d5bbe"),
			RoleError:       lipgloss.Color("#ff5555"),
			RoleErrorBg:     lipgloss.Color("#5c1a1a"),
			RoleWarning:     lipgloss.Color("#f1fa8c"),
			RoleSuccess:     lipgloss.Color("#50fa7b"),
			RoleCursorBg:    lipgloss.Color("#bd93f9"),
			RoleCursorFg:    lipgloss.Color("#282a36"),
			RoleBorder:      lipgloss.Color("#6272a4"),
			RoleBorderFocus: lipgloss.Color("#bd93f9"),
		},
	}
}
