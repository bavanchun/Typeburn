// Package app holds the Bubble Tea root model. It owns global state
// (terminal size, theme, keymap, active screen) and routes messages to the
// active screen's sub-model.
package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/ui"
	"github.com/bavanchun/Typeburn/internal/update"
)

// Model is the root Elm model.
type Model struct {
	screen   Screen
	w, h     int
	theme    theme.Theme
	keys     config.Keymap
	settings config.Settings
	home     ui.HomeModel
	typing   ui.TypingModel
	result   ui.ResultModel
	sett     ui.SettingsModel
	hist     ui.HistoryModel
	// codePaste is the active ScreenCodePaste sub-model. Constructed fresh on
	// NavCodePasteMsg; only consulted while screen == ScreenCodePaste.
	codePaste ui.CodePasteModel

	// codeText is the user-supplied code snippet loaded from --text <file>.
	// Empty string means no code text is available; Code mode is then disabled.
	codeText string
	// codeHint is a user-facing reason when loading the code text failed
	// (e.g., "text file is empty"). Empty when codeText loaded successfully.
	codeHint string

	// updateHint is set when an opportunistic check found a newer release.
	// Nil means no hint to display. Forwarded to ResultModel after each test.
	updateHint *update.Result

	// quitPrompt is non-nil when the esc-on-Home quit confirmation overlay is
	// active. ctrl+c always hard-quits regardless of this field.
	quitPrompt *quitPromptModel

	// persistErr holds a transient, user-visible message when a history or
	// settings disk write failed. Empty = nothing to show. It is set at the
	// failing persist site and cleared on the next keypress (any key) so the
	// notice behaves like a dismissible toast; it never blocks input.
	persistErr string

	// animNowMs is the shared animation clock: the epoch-ms of the most recent
	// FrameTickMsg. Animated screens read it (stored on the sub-model) so every
	// moment derives purely from time, mirroring metrics.Compute's replay. Zero
	// until the first frame tick fires.
	animNowMs int64

	// transition is a root-owned screen transition (currently Typing→Result),
	// non-nil only while one is mid-flight. View derives its expiry from
	// animNowMs; it is nil-ed out lazily in Update on the next message.
	transition *transitionState
}

// Update handles global concerns (resize, quit, navigation) and delegates to
// the active screen sub-model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// FrameTickMsg: animation frame. Handled by an explicit root branch (never
	// the generic delegation below) so the self-stopping re-arm command is not
	// lost. Stamps animNowMs, forwards to the active screen, re-arms iff live.
	if ft, ok := msg.(ui.FrameTickMsg); ok {
		return m.handleFrameTick(ft)
	}

	// Lazily clear an expired transition. View stops using it once animNowMs
	// passes its end (derived expiry); this nil-out reclaims the snapshot on the
	// next message without any reliance on a trailing cleanup tick.
	if m.transition != nil && m.animNowMs >= m.transition.startMs+m.transition.durMs {
		m.transition = nil
	}

	// StartTestMsg: Home screen requested a test start.
	if sm, ok := msg.(ui.StartTestMsg); ok {
		var t ui.TypingModel
		if sm.Mode == config.ModeCode {
			// Code mode: use the supplied snippet verbatim — do not call words.ForMode.
			t = ui.NewTypingCode(sm.CodeText, m.theme, m.keys, m.settings.BlinkCursor, m.settings.StrictMode)
		} else {
			t = ui.NewTyping(sm.Mode, sm.Length, sm.QuoteLen,
				m.theme, m.keys, m.settings.BlinkCursor, m.settings.StrictMode,
				m.settings.Punctuation, m.settings.Numbers)
		}
		m.typing = t.SetSize(m.w, m.h)
		m.screen = ScreenTyping
		// Bootstrap the 100ms tick so the caret blinks before the first keystroke.
		return m, m.typing.InitCmd()
	}

	// ResultMsg: test completed → persist record, detect new-best, show result,
	// and (from the typing screen) start the Typing→Result transition.
	if rm, ok := msg.(ui.ResultMsg); ok {
		m = m.handleResultWithTransition(rm)
		return m, ui.FrameTickCmd()
	}

	// NavHistoryMsg: navigate to History screen (load fresh from disk).
	if _, ok := msg.(ui.NavHistoryMsg); ok {
		m = m.handleNavHistory()
		return m, nil
	}

	// AbortMsg from typing screen → return to Home.
	if _, ok := msg.(ui.AbortMsg); ok {
		m.screen = ScreenHome
		m.transition = nil // abort cancels any in-flight transition
		return m, nil
	}

	// In-app paste wiring (open paste screen / apply a valid paste). Bodies
	// live in model_code_paste.go to keep this router compact.
	if _, ok := msg.(ui.NavCodePasteMsg); ok {
		return m.openCodePaste(), nil
	}
	if cp, ok := msg.(ui.CodePastedMsg); ok {
		return m.applyCodePaste(cp.Text), nil
	}

	// SettingsChangedMsg: a Settings row changed → apply to the live model
	// (persist + theme rebuild + sub-model re-injection). Body in
	// model_settings.go.
	if sc, ok := msg.(ui.SettingsChangedMsg); ok {
		return m.applySettings(sc.Settings), nil
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// A resize invalidates the transition snapshot (taken at the old width),
		// so snap straight to the target screen instead of bleeding old geometry.
		m.transition = nil
		m.w, m.h = msg.Width, msg.Height
		m.home = m.home.SetSize(msg.Width, msg.Height)
		m.sett = m.sett.SetSize(msg.Width, msg.Height)
		m.hist = m.hist.SetSize(msg.Width, msg.Height)
		m.codePaste = m.codePaste.SetSize(msg.Width, msg.Height)
		if m.screen == ScreenTyping {
			m.typing = m.typing.SetSize(msg.Width, msg.Height)
		}
		if m.screen == ScreenResult {
			m.result = m.result.SetSize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyPressMsg:
		return m.handleKey(msg.Key())

	case tea.PasteMsg:
		// Feed paste runes sequentially into the typing engine (project decision
		// documented in charm-v2-api-cheatsheet.md). Ignored on all other screens.
		if m.screen == ScreenTyping {
			var cmd tea.Cmd
			m.typing, cmd = m.typing.Update(msg)
			return m, cmd
		}
		if m.screen == ScreenCodePaste {
			var cmd tea.Cmd
			m.codePaste, cmd = m.codePaste.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	// Delegate remaining messages to the active screen sub-model.
	if m.screen == ScreenTyping {
		var cmd tea.Cmd
		m.typing, cmd = m.typing.Update(msg)
		return m, cmd
	}
	if m.screen == ScreenResult {
		var cmd tea.Cmd
		m.result, cmd = m.result.Update(msg)
		return m, cmd
	}
	if m.screen == ScreenSettings {
		var cmd tea.Cmd
		m.sett, cmd = m.sett.Update(msg)
		return m, cmd
	}
	if m.screen == ScreenHistory {
		var cmd tea.Cmd
		m.hist, cmd = m.hist.Update(msg)
		return m, cmd
	}
	if m.screen == ScreenCodePaste {
		var cmd tea.Cmd
		m.codePaste, cmd = m.codePaste.Update(msg)
		return m, cmd
	}

	return m, nil
}
