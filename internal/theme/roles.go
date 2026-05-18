// Package theme defines the role-based color system. Screens reference
// semantic roles only (never literal colors), so a theme is one map and
// adding a theme costs one file.
package theme

// Role is a semantic color slot. Every renderable concept maps to a Role;
// the active Theme resolves a Role to a concrete color (or, under NO_COLOR,
// to a text attribute).
type Role int

const (
	RoleBg          Role = iota // app background (usually terminal default)
	RoleSurface                 // cards, settings rows, result panel
	RoleSurfaceAlt              // selected-row base, sparkline gutter
	RoleTextPrimary             // primary reading text, headers
	RoleTextMuted               // correct typed chars, secondary labels
	RoleTextFaint               // upcoming/untyped text, hints, disabled
	RoleAccent                  // brand/action: logo, WPM number, selection
	RoleAccentDim               // accent at rest (idle progress, unfocused)
	RoleError                   // incorrect char foreground
	RoleErrorBg                 // incorrect-space marker background
	RoleWarning                 // caution states, low accuracy, degraded notice
	RoleSuccess                 // positive deltas, new-best badge
	RoleCursorBg                // block-cursor background
	RoleCursorFg                // char rendered under the cursor
	RoleBorder                  // borders, separators, table rules
	RoleBorderFocus             // border of focused/active panel
	roleCount                   // sentinel: number of roles
)
