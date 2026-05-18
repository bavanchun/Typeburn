package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// SolarizedLight is the Solarized light palette (base3 ground, blue accent),
// the first light theme. Source: https://ethanschoonover.com/solarized/.
// Light-bg role contrast reviewed across all screens; see
// docs/design-guidelines.md §2.1.
func SolarizedLight() Theme {
	return Theme{
		name: "solarized-light",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#fdf6e3"),
			RoleSurface:     lipgloss.Color("#eee8d5"),
			RoleSurfaceAlt:  lipgloss.Color("#ded8c5"),
			RoleTextPrimary: lipgloss.Color("#586e75"),
			RoleTextMuted:   lipgloss.Color("#657b83"),
			RoleTextFaint:   lipgloss.Color("#93a1a1"),
			RoleAccent:      lipgloss.Color("#268bd2"),
			RoleAccentDim:   lipgloss.Color("#3a7ca5"),
			RoleError:       lipgloss.Color("#dc322f"),
			RoleErrorBg:     lipgloss.Color("#f5d0cd"),
			RoleWarning:     lipgloss.Color("#b58900"),
			RoleSuccess:     lipgloss.Color("#859900"),
			RoleCursorBg:    lipgloss.Color("#268bd2"),
			RoleCursorFg:    lipgloss.Color("#fdf6e3"),
			RoleBorder:      lipgloss.Color("#93a1a1"),
			RoleBorderFocus: lipgloss.Color("#268bd2"),
		},
	}
}
