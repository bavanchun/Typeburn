package ui

import (
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// PersistenceNotice renders a single-line, dismissible toast shown when a
// disk write (history or settings) failed. It is a transient overlay on the
// frame's last row — RoleWarning message + RoleTextFaint dismiss hint — so it
// is legible under NO_COLOR (attribute-only) without shifting any layout.
//
// The caller is responsible for horizontal placement; this returns the bare
// styled line. Empty msg yields an empty string (caller should not invoke it
// in that case, but guard anyway).
func PersistenceNotice(msg string, th theme.Theme) string {
	if msg == "" {
		return ""
	}
	warn := th.Style(theme.RoleWarning).Render("⚠ " + msg)
	hint := th.Style(theme.RoleTextFaint).Render("  ·  press any key to dismiss")
	return warn + hint
}
