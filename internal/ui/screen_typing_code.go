package ui

// screen_typing_code.go contains the ModeCode constructor and test accessors
// that extend TypingModel for code-snippet typing. Kept separate to hold
// screen_typing.go under 200 LOC (modularization boundary: code-mode-specific
// construction vs general typing engine lifecycle).

import (
	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// NewTypingCode constructs a TypingModel for ModeCode using the supplied text
// verbatim as the target — words.ForMode is NOT called. This keeps the pure
// words/typing packages free of code-mode knowledge (I/O boundary is in main).
func NewTypingCode(target string, th theme.Theme, km config.Keymap, blink bool) TypingModel {
	eng := typing.New(target, config.ModeCode, 0)
	return TypingModel{
		eng:    eng,
		mode:   config.ModeCode,
		target: target,
		th:     th,
		keys:   km,
		blink:  blink,
		seed:   0,
	}
}

// ExportedStartMs returns the startMs field for test assertions (white-box
// access needed by code_mode_test.go to verify engine timer state).
func (m TypingModel) ExportedStartMs() int64 { return m.startMs }

// ExportedLog returns the engine's keystroke log for test assertions.
func (m TypingModel) ExportedLog() []typing.Keystroke { return m.eng.Log() }
