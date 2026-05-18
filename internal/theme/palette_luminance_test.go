package theme

import "testing"

// perceivedLuminance returns 0..1 luminance from a role's color using the
// Rec.601 weights. color.Color.RGBA() yields 16-bit premultiplied channels;
// theme colors are fully opaque so premultiplication is a no-op here.
func perceivedLuminance(th Theme, r Role) float64 {
	c := th.Color(r)
	if c == nil {
		return -1
	}
	rr, gg, bb, _ := c.RGBA()
	return (0.299*float64(rr) + 0.587*float64(gg) + 0.114*float64(bb)) / 65535.0
}

// TestPalettes_LightVsDarkNotInverted is an objective guard against an
// accidentally inverted palette: a light theme must have a light background
// and dark primary text; a dark theme the inverse. Subjective per-screen
// contrast review is still the real gate (see docs/design-guidelines.md),
// but this catches the gross failure cheaply.
func TestPalettes_LightVsDarkNotInverted(t *testing.T) {
	light := map[string]bool{"solarized-light": true, "gruvbox-light": true}
	for _, name := range packThemes {
		if name == "mono" {
			continue // mono is a fixed near-monochrome ramp, not bg/text paired
		}
		th := Load(name, false)
		bg := perceivedLuminance(th, RoleBg)
		fg := perceivedLuminance(th, RoleTextPrimary)
		if light[name] {
			if bg <= fg || bg < 0.5 {
				t.Errorf("light theme %q: want light bg (>0.5) brighter than text; got bg=%.2f fg=%.2f", name, bg, fg)
			}
		} else {
			if bg >= fg || bg > 0.5 {
				t.Errorf("dark theme %q: want dark bg (<0.5) darker than text; got bg=%.2f fg=%.2f", name, bg, fg)
			}
		}
	}
}
