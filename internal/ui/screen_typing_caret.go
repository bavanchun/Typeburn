package ui

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// HasActiveAnim reports whether a fast-frame caret animation is live at nowMs:
// true only inside the post-keystroke fade/trail window. The blink itself rides
// the 100ms timer tick (not the frame loop), so once the fade window closes the
// 33ms loop self-stops and the caret keeps blinking at the cheaper cadence.
func (m TypingModel) HasActiveAnim(nowMs int64) bool {
	return m.lastKeyMs > 0 && nowMs-m.lastKeyMs < caretFadeMs
}

// InitCmd bootstraps the 100ms timer tick when the typing screen becomes active
// so the caret blinks BEFORE the first keystroke (the tick otherwise starts only
// once typing begins). The root returns this when entering ScreenTyping.
func (m TypingModel) InitCmd() tea.Cmd { return tickCmd() }

// caretAnimState builds the per-frame caret animation descriptor from the current
// model clock and cursor position. When blink is off (steady block) the cursor is
// always "on"; otherwise the blink phase derives purely from nowMs.
func (m TypingModel) caretAnimState(states []typing.CharState) caretAnim {
	blinkOn := true
	if m.blink {
		blinkOn = blinkOnAt(m.nowMs)
	}
	return caretAnim{
		nowMs:     m.nowMs,
		lastKeyMs: m.lastKeyMs,
		blinkOn:   blinkOn,
		cursorIdx: indexOfCurrent(states),
	}
}
