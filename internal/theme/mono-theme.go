package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Mono is a greyscale theme: a single hue ladder, white accent. It keeps the
// app usable/pleasant on terminals where the green accent clashes, and makes
// the Settings "Theme" control meaningful in v1. Errors still receive an
// underline via Theme.Style (never color-alone).
func Mono() Theme {
	return Theme{
		name: "mono",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#0E0E0E"),
			RoleSurface:     lipgloss.Color("#1A1A1A"),
			RoleSurfaceAlt:  lipgloss.Color("#2A2A2A"),
			RoleTextPrimary: lipgloss.Color("#F2F2F2"),
			RoleTextMuted:   lipgloss.Color("#9A9A9A"),
			RoleTextFaint:   lipgloss.Color("#555555"),
			RoleAccent:      lipgloss.Color("#FFFFFF"),
			RoleAccentDim:   lipgloss.Color("#8A8A8A"),
			RoleError:       lipgloss.Color("#FFFFFF"),
			RoleErrorBg:     lipgloss.Color("#3A3A3A"),
			RoleWarning:     lipgloss.Color("#CFCFCF"),
			RoleSuccess:     lipgloss.Color("#FFFFFF"),
			RoleCursorBg:    lipgloss.Color("#F2F2F2"),
			RoleCursorFg:    lipgloss.Color("#0E0E0E"),
			RoleBorder:      lipgloss.Color("#3A3A3A"),
			RoleBorderFocus: lipgloss.Color("#F2F2F2"),
		},
	}
}

// noColorTheme is selected when NO_COLOR is set. The color map is unused;
// Theme.Style emits attributes only (see attrOnlyStyle).
func noColorTheme() Theme {
	return Theme{name: "no-color", noColor: true}
}
