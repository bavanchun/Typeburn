package app

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/ui"
)

// transitionDurMs is the Typing→Result transition length. Short enough to feel
// like a soft cut, not a wait.
const transitionDurMs int64 = 250

// transitionState holds a root-owned screen transition. It spans two screens, so
// only app.Model can compose it (sub-models can't see each other). fromFrame is
// the already-placed outgoing frame snapshot; toScreen is rendered live each
// frame. nil on app.Model means no transition.
type transitionState struct {
	fromFrame string
	toScreen  Screen
	startMs   int64
	durMs     int64
}

// handleResultWithTransition persists the result and switches to ScreenResult,
// and — only when coming from the typing screen at a non-degraded size — starts
// a Typing→Result transition by snapshotting the outgoing placed frame BEFORE
// the screen flips. A result arriving from elsewhere (e.g. tests) just routes.
func (m Model) handleResultWithTransition(rm ui.ResultMsg) Model {
	transitionable := m.screen == ScreenTyping && m.w >= 60 && m.h >= 20
	var fromFrame string
	if transitionable {
		fromFrame = m.composeScreen(ScreenTyping)
	}
	m = m.handleResultMsg(rm)
	if transitionable {
		m.transition = &transitionState{
			fromFrame: fromFrame,
			toScreen:  ScreenResult,
			startMs:   nowUTC().UnixMilli(),
			durMs:     transitionDurMs,
		}
	}
	return m
}

// progress returns linear progress in [0,1] at nowMs; the caller applies easing.
func (t *transitionState) progress(nowMs int64) float64 {
	if t == nil || t.durMs <= 0 {
		return 1
	}
	p := float64(nowMs-t.startMs) / float64(t.durMs)
	if p < 0 {
		return 0
	}
	if p > 1 {
		return 1
	}
	return p
}

// renderTransition composes the outgoing and incoming frames at eased progress p.
//
//   - Color: a dim-curtain crossfade — the outgoing frame is faint for the first
//     half, the incoming frame faint for the second half. True per-cell alpha is
//     not possible in a terminal; faint is the equivalent. (Layout is untouched:
//     only SGR is added, so runes/line-count/width are preserved.)
//   - NO_COLOR: a top-down wipe — the first floor(rows*p) lines come from the
//     incoming frame, the rest from the outgoing one. Pure line slicing, so width
//     and line count are identical in both frames.
func renderTransition(from, to string, p float64, noColor bool) string {
	fromLines := strings.Split(from, "\n")
	toLines := strings.Split(to, "\n")
	n := len(fromLines)
	if len(toLines) > n {
		n = len(toLines)
	}
	fromLines = padLines(fromLines, n)
	toLines = padLines(toLines, n)

	if noColor {
		visible := int(float64(n) * p)
		if visible < 0 {
			visible = 0
		}
		if visible > n {
			visible = n
		}
		out := make([]string, n)
		for i := 0; i < n; i++ {
			if i < visible {
				out[i] = toLines[i]
			} else {
				out[i] = fromLines[i]
			}
		}
		return strings.Join(out, "\n")
	}

	faint := lipgloss.NewStyle().Faint(true)
	if p < 0.5 {
		return faint.Render(strings.Join(fromLines, "\n"))
	}
	return faint.Render(strings.Join(toLines, "\n"))
}

// padLines pads a line slice up to n entries with empty lines so the wipe can
// mix two frames index-by-index without bounds risk. Placed frames already share
// the same height, so this is defensive only.
func padLines(lines []string, n int) []string {
	for len(lines) < n {
		lines = append(lines, "")
	}
	return lines
}
