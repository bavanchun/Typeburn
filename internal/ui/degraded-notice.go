package ui

import (
	"fmt"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// DegradedNotice renders the "terminal too small" notice per design §4.3.
//
// Exact copy per spec:
//
//	Terminal too small          ← RoleWarning
//	Need at least 60×20 (WxH)  ← text-muted
//	Resize to continue · ctrl+c quit ← text-faint
//
// The caller (root View) is responsible for centering this in the terminal via
// lipgloss.Place before returning it as a tea.View.
func DegradedNotice(w, h int, th theme.Theme) string {
	warn := th.Style(theme.RoleWarning).Render("Terminal too small")
	info := th.Style(theme.RoleTextMuted).Render(
		fmt.Sprintf("Need at least 60×20 (current %d×%d)", w, h),
	)
	hint := th.Style(theme.RoleTextFaint).Render("Resize to continue · ctrl+c quit")

	return lipgloss.JoinVertical(lipgloss.Center, warn, info, hint)
}
