package ui

import (
	"errors"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/codetext"
	"github.com/bavanchun/Typeburn/internal/theme"
)

// CodePasteModel is the sub-model for ScreenCodePaste. It captures one
// bracketed paste (tea.PasteMsg), runs codetext.Normalize, and on success
// emits CodePastedMsg{Text}; on failure it stays in an error state showing
// the reason so the user can paste again.
//
// Approach A: this sub-model owns ONLY normalization + error/retry state. It
// never mutates app state and performs no I/O (codetext.Normalize is pure).
// It also never handles esc/cancel — the global Back handler routes esc on
// any non-Home screen back to Home before this sub-model is reached, so a
// cancel message would be dead code.
type CodePasteModel struct {
	th     theme.Theme
	w, h   int
	errMsg string // non-empty ⇒ errored state (user-facing reason); "" ⇒ waiting
}

// NewCodePaste constructs a fresh paste sub-model in the waiting state.
func NewCodePaste(th theme.Theme) CodePasteModel {
	return CodePasteModel{th: th}
}

// SetSize stores terminal dimensions for layout.
func (m CodePasteModel) SetSize(w, h int) CodePasteModel {
	m.w, m.h = w, h
	return m
}

// Update handles ONLY tea.PasteMsg. Every other message (keys incl. esc,
// ticks, resizes routed elsewhere) is a no-op here: the model is returned
// unchanged with a nil cmd. On a paste, msg.Content (tea.PasteMsg is a struct
// with a Content string) is normalized: success emits CodePastedMsg and
// clears any prior error; failure stores the mapped reason and stays so the
// next paste is a fresh attempt (last paste wins).
func (m CodePasteModel) Update(msg tea.Msg) (CodePasteModel, tea.Cmd) {
	pm, ok := msg.(tea.PasteMsg)
	if !ok {
		return m, nil
	}
	text, err := codetext.Normalize(pm.Content)
	if err != nil {
		m.errMsg = pasteErrReason(err)
		return m, nil
	}
	m.errMsg = ""
	return m, func() tea.Msg { return CodePastedMsg{Text: text} }
}

// pasteErrReason maps a codetext sentinel to a user-facing one-liner. It
// branches with errors.Is against the package sentinels (never string
// matching) so wrapped messages (e.g. ErrTooLarge with a cap suffix) still
// classify correctly.
func pasteErrReason(err error) string {
	switch {
	case errors.Is(err, codetext.ErrEmpty):
		return "paste was empty or whitespace-only"
	case errors.Is(err, codetext.ErrTooLarge):
		return "paste is too large (max 10000 runes / 500 lines)"
	case errors.Is(err, codetext.ErrBinary):
		return "paste is not valid text"
	default:
		return "could not read the pasted text"
	}
}
