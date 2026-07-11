package ui

import (
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// StatCard renders a labelled stat value pair per design §5.3.
// label is rendered in RoleTextMuted; value is rendered in the given role.
// Example: StatCard("acc", "97%", theme.RoleSuccess, th) → "acc 97%"
func StatCard(label, value string, role theme.Role, th theme.Theme) string {
	labelStyle := th.Style(theme.RoleTextMuted)
	valueStyle := th.Style(role)
	return labelStyle.Render(label) + " " + valueStyle.Render(value)
}
