package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/anim"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// Caret animation timing. blinkHalfMs is half the 530ms blink cycle (the
// long-standing Windows caret rate); caretFadeMs is the fast-frame window after
// each keystroke during which the new cell fades and the vacated cell trails.
const (
	blinkHalfMs int64 = 265
	caretFadeMs int64 = 150
)

// caretAnim carries per-frame caret timing into the stream renderers. The zero
// value (cursorIdx == 0 but lastKeyMs == 0, blinkOn false) does NOT animate the
// way disabledCaret does; always construct via the View or disabledCaret().
//
// When cursorIdx < 0 no caret overrides apply at all, so the render is
// byte-identical to the static stream — this protects existing goldens and the
// settled end state.
type caretAnim struct {
	nowMs     int64
	lastKeyMs int64
	blinkOn   bool
	cursorIdx int // index of the Current cell; <0 = none (complete/disabled)
}

// disabledCaret returns a caretAnim that applies no overrides.
func disabledCaret() caretAnim { return caretAnim{cursorIdx: -1} }

// blinkOnAt derives the blink phase purely from time, so it needs no stored
// toggle and stays deterministic under a pinned clock.
func blinkOnAt(nowMs int64) bool {
	if nowMs <= 0 {
		return true
	}
	return (nowMs/blinkHalfMs)%2 == 0
}

// elapsed is milliseconds since the last keystroke (clamped at ≥0).
func (c caretAnim) elapsed() int64 {
	d := c.nowMs - c.lastKeyMs
	if d < 0 {
		return 0
	}
	return d
}

// fadeActive reports whether the post-keystroke fast-frame window is still open.
// This is what HasActiveAnim keys on, so the 33ms loop self-stops ~150ms after
// the last keystroke and falls back to the 100ms blink cadence.
func (c caretAnim) fadeActive() bool {
	return c.lastKeyMs > 0 && c.elapsed() < caretFadeMs
}

// caretCellStyle returns the animated style for the cell at distance d from the
// cursor (0 = cursor, 1 = freshly-typed cell, 2 = just-vacated trail) and whether
// an override applies. When it returns false the renderer keeps the base style,
// so non-animated frames cost nothing extra and settle byte-identical.
//
// Color themes interpolate foreground; under NO_COLOR (theme.Color == nil) the
// animation degrades to an attribute step (bold/faint) with identical cell width.
func (c caretAnim) caretCellStyle(d int, st typing.CharState, th theme.Theme) (lipgloss.Style, bool) {
	switch d {
	case 0:
		// Cursor cell: blink. When "on" (or steady) the base already renders the
		// cursor block, so no override is needed; when "off" it looks like the
		// upcoming untyped cell. NO_COLOR: base cursor is Reverse(true); off drops
		// the reverse (RoleTextFaint), so the blink is an attribute toggle.
		if c.blinkOn {
			return lipgloss.NewStyle(), false
		}
		return th.Style(theme.RoleTextFaint), true
	case 1:
		// Freshly-typed correct cell fades accent → muted over the window.
		if st != typing.Correct || !c.fadeActive() {
			return lipgloss.NewStyle(), false
		}
		return c.fadeStyle(th, theme.RoleAccent, theme.RoleTextMuted, true), true
	case 2:
		// Just-vacated correct cell trails one notch dim → muted over the window.
		if st != typing.Correct || !c.fadeActive() {
			return lipgloss.NewStyle(), false
		}
		return c.fadeStyle(th, theme.RoleTextFaint, theme.RoleTextMuted, false), true
	}
	return lipgloss.NewStyle(), false
}

// fadeStyle builds the interpolated style for an animated cell. boldStep selects
// the NO_COLOR fallback: true → bold for the first half of the window (the new
// cell), false → faint for the whole window (the trail). The settled colors equal
// the cell's base role, so the end frame matches the static render.
func (c caretAnim) fadeStyle(th theme.Theme, from, to theme.Role, boldStep bool) lipgloss.Style {
	fc, tc := th.Color(from), th.Color(to)
	if fc == nil || tc == nil { // NO_COLOR: attribute step, never color math
		if boldStep {
			if c.elapsed() < caretFadeMs/2 {
				return lipgloss.NewStyle().Bold(true)
			}
			return lipgloss.NewStyle()
		}
		return lipgloss.NewStyle().Faint(true)
	}
	t := anim.EaseOutQuad(anim.Clamp01(float64(c.elapsed()) / float64(caretFadeMs)))
	return lipgloss.NewStyle().Foreground(anim.LerpColor(fc, tc, t))
}

// indexOfCurrent returns the index of the Current (cursor) cell, or -1 if none
// (a completed test has no cursor).
func indexOfCurrent(states []typing.CharState) int {
	for i, s := range states {
		if s == typing.Current {
			return i
		}
	}
	return -1
}
