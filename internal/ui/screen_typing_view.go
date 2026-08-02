package ui

import (
	"time"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

// fixedOverhead is the number of non-stream rows in the typing screen layout:
// header + blank + blank-before-footer + footer = 4 rows.
const fixedOverhead = 4

// The word-stream window grows with the terminal instead of pinning to a fixed
// three lines, so a tall terminal is used rather than padded. It stops at seven
// because past that the eye stops tracking the caret's line.
const (
	minWordStreamRows = 3
	maxWordStreamRows = 7
)

// codeStreamHeight gives Code mode every row the rest of the frame is not
// using: the buffer is the user's own file, so showing as much of it as fits is
// the point.
func codeStreamHeight(termH int) int {
	if rows := termH - fixedOverhead; rows > 0 {
		return rows
	}
	return 1
}

// wordStreamHeight sizes the word-stream window from the terminal height. Words
// and Quote generate far more text than any terminal can show, so without a
// budget the stream renders every row it has and the cell buffer drops the
// overflow — footer first, and eventually the caret itself.
//
// The divisor is what makes the window actually track the terminal: a 24-row
// terminal gets 3 rows, the same as Monkeytype, and a 50-row one gets 7. A
// larger share of the height would pin every supported terminal to the upper
// clamp and the window would never adapt at all. The upper bound is 7 because
// past that the eye stops tracking which line the caret is on.
func wordStreamHeight(termH int) int {
	rows := (termH - fixedOverhead) / 6
	if rows < minWordStreamRows {
		return minWordStreamRows
	}
	if rows > maxWordStreamRows {
		return maxWordStreamRows
	}
	return rows
}

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

	// Both branches window their stream to a height budget. Bubble Tea draws
	// into a w×h cell buffer in altscreen, so anything past the budget is
	// dropped without scrolling — it is not off-screen, it is gone.
	var stream string
	if m.mode == config.ModeCode {
		stream = renderCodeStreamAnim(states, target, typed, cw, codeStreamHeight(m.h), m.th, ca)
	} else {
		stream = renderWordStreamAnim(states, target, typed, cw, wordStreamHeight(m.h), m.th, ca, m.wordCache)
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
