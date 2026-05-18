package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// GruvboxDark is the Pavel Pertsev Gruvbox dark palette
// (https://github.com/morhetz/gruvbox) mapped onto the 16 roles: bg0 ground,
// yellow accent, retro warm signal colors. Hex pinned; see
// docs/design-guidelines.md §2.1.
func GruvboxDark() Theme {
	return Theme{
		name: "gruvbox-dark",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#282828"),
			RoleSurface:     lipgloss.Color("#3c3836"),
			RoleSurfaceAlt:  lipgloss.Color("#504945"),
			RoleTextPrimary: lipgloss.Color("#ebdbb2"),
			RoleTextMuted:   lipgloss.Color("#a89984"),
			RoleTextFaint:   lipgloss.Color("#665c54"),
			RoleAccent:      lipgloss.Color("#fabd2f"),
			RoleAccentDim:   lipgloss.Color("#b57614"),
			RoleError:       lipgloss.Color("#fb4934"),
			RoleErrorBg:     lipgloss.Color("#4a1f1c"),
			RoleWarning:     lipgloss.Color("#fe8019"),
			RoleSuccess:     lipgloss.Color("#b8bb26"),
			RoleCursorBg:    lipgloss.Color("#fabd2f"),
			RoleCursorFg:    lipgloss.Color("#282828"),
			RoleBorder:      lipgloss.Color("#665c54"),
			RoleBorderFocus: lipgloss.Color("#fabd2f"),
		},
	}
}
