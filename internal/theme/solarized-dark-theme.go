package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// SolarizedDark is the Ethan Schoonover Solarized dark palette mapped onto the
// 16 semantic roles. Source: https://ethanschoonover.com/solarized/ (base03
// ground, blue accent). Hex values are pinned; see docs/design-guidelines.md
// §2.1 for the role rationale.
func SolarizedDark() Theme {
	return Theme{
		name: "solarized-dark",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#002b36"),
			RoleSurface:     lipgloss.Color("#073642"),
			RoleSurfaceAlt:  lipgloss.Color("#094d5c"),
			RoleTextPrimary: lipgloss.Color("#93a1a1"),
			RoleTextMuted:   lipgloss.Color("#839496"),
			RoleTextFaint:   lipgloss.Color("#586e75"),
			RoleAccent:      lipgloss.Color("#268bd2"),
			RoleAccentDim:   lipgloss.Color("#1f6a9e"),
			RoleError:       lipgloss.Color("#dc322f"),
			RoleErrorBg:     lipgloss.Color("#4a1715"),
			RoleWarning:     lipgloss.Color("#b58900"),
			RoleSuccess:     lipgloss.Color("#859900"),
			RoleCursorBg:    lipgloss.Color("#268bd2"),
			RoleCursorFg:    lipgloss.Color("#002b36"),
			RoleBorder:      lipgloss.Color("#586e75"),
			RoleBorderFocus: lipgloss.Color("#268bd2"),
		},
	}
}
