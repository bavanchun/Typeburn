package ui

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// fixedOverhead is the number of non-stream rows in the typing screen layout:
// header + blank + blank-before-footer + footer = 4 rows.
const fixedOverhead = 4

// View renders the full typing screen content as a string. The root app.Model
// wraps this in lipgloss.Place for centering — View itself does not center.
//
// Degraded mode (w<60 or h<20) is handled by the root View before delegation;
// this method is only called when the terminal is large enough.
func (m TypingModel) View() string {
	tier := WidthTier(m.w, m.h)
	cw := ContentWidth(m.w, tier)

	elapsed := elapsedMs(m.startMs, time.Now())
	done, total := m.eng.Progress()

	header := ModeHeader(
		m.mode,
		m.headerWPM,
		done, total,
		float64(elapsed)/1000.0,
		m.length,
		m.th,
	)

	// Select renderer based on mode: Code uses the literal code stream renderer;
	// all other modes use the word stream renderer. Both take the per-frame caret
	// animation; a settled caret renders identically to the static stream.
	states := m.eng.States()
	target := []rune(m.target)
	typed := m.eng.Typed()
	ca := m.caretAnimState(states)

	var stream string
	if m.mode == config.ModeCode {
		// streamHeight = total height minus fixed overhead rows; clamp ≥1.
		streamHeight := m.h - fixedOverhead
		if streamHeight < 1 {
			streamHeight = 1
		}
		stream = renderCodeStreamAnim(states, target, typed, cw, streamHeight, m.th, ca)
	} else {
		stream = renderWordStreamAnim(states, target, typed, cw, m.th, ca, m.wordCache)
	}

	footer := RenderFooter(TypingHints(), m.w, m.th)

	// Emit a compact block (header · stream · footer with single-line gaps).
	// The root wraps this in lipgloss.Place(Center,Center); keeping the block
	// compact lets that vertical centering actually take effect instead of
	// the stream being pinned to the top with the footer at the very bottom.
	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		stream,
		"",
		footer,
	)
}
