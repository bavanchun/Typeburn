package theme

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Default is the dark "slate + green accent" theme. Truecolor hex values;
// the lipgloss renderer downsamples to ANSI256/16 on limited terminals.
// Values mirror docs/design-guidelines.md §2.1 (the design source of truth).
func Default() Theme {
	return Theme{
		name: "default",
		colors: map[Role]color.Color{
			RoleBg:          lipgloss.Color("#0E1117"),
			RoleSurface:     lipgloss.Color("#161B22"),
			RoleSurfaceAlt:  lipgloss.Color("#21262D"),
			RoleTextPrimary: lipgloss.Color("#E6EDF3"),
			RoleTextMuted:   lipgloss.Color("#8B949E"),
			RoleTextFaint:   lipgloss.Color("#484F58"),
			RoleAccent:      lipgloss.Color("#22C55E"),
			RoleAccentDim:   lipgloss.Color("#15803D"),
			RoleError:       lipgloss.Color("#F85149"),
			RoleErrorBg:     lipgloss.Color("#5C1A1A"),
			RoleWarning:     lipgloss.Color("#E3B341"),
			RoleSuccess:     lipgloss.Color("#3FB950"),
			RoleCursorBg:    lipgloss.Color("#22C55E"),
			RoleCursorFg:    lipgloss.Color("#0E1117"),
			RoleBorder:      lipgloss.Color("#30363D"),
			RoleBorderFocus: lipgloss.Color("#22C55E"),
		},
	}
}
