package app

import "github.com/bavanchun/Typeburn/v2/internal/ui"

// openCodePaste switches to ScreenCodePaste with a fresh, sized paste
// sub-model. Triggered by ui.NavCodePasteMsg (Home Code row, no snippet).
func (m Model) openCodePaste() Model {
	m.codePaste = ui.NewCodePaste(m.theme).SetSize(m.w, m.h)
	m.screen = ScreenCodePaste
	return m
}

// applyCodePaste records a valid in-app paste and returns to Home. The
// snippet is pushed into the EXISTING Home via WithCodeText — NOT a NewHome
// rebuild, which would reset modeIdx to DefaultMode and lose the Code
// selection. The Code row was active when paste opened, so on return Enter
// starts the Code test immediately. Any prior load-failure hint is cleared.
func (m Model) applyCodePaste(text string) Model {
	m.codeText = text
	m.codeHint = ""
	m.home = m.home.WithCodeText(text, "")
	m.screen = ScreenHome
	return m
}
