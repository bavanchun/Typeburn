package app

import (
	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/ui"
)

// appCase builds a root model already sized to w×h and parked in the state
// under test. Rebuilt per size because the sub-models compute layout in
// SetSize.
type appCase struct {
	name  string
	build func(w, h int) Model
}

func baseModel(w, h int) Model {
	m := New(theme.Load("default", false), config.Defaults(), "", "", nil)
	sized, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return sized.(Model)
}

// typingModel builds the typing screen from a fixed target. NewTyping seeds
// the word generator from the clock, so a frame built with it changes line
// count between runs, and newTypingWithSeed is unexported to this package.
//
// Consequence worth knowing: NewTypingCode routes to the code-stream renderer,
// so the word-stream renderer is never measured at root composition. The ui
// package's harness covers it — this one covers what only the root can show.
func typingModel(m Model, w, h int) ui.TypingModel {
	return ui.NewTypingCode(typingTarget, m.theme, m.keys, false, false).SetSize(w, h)
}

// typingTarget is long enough that the stream wraps at every measured width.
const typingTarget = "the quick brown fox jumps over the lazy dog while a second sentence " +
	"keeps the stream long enough to wrap at sixty columns and again at two hundred"

// appResultMsg mirrors the ui harness fixture: a value whose digits exercise
// the block glyphs rather than one that happens to render cleanly.
func appResultMsg() ui.ResultMsg {
	per := make([]metrics.PerSecond, 30)
	for i := range per {
		per[i] = metrics.PerSecond{Sec: i, RawWPM: float64(70 + i%13), CorrectChars: 5, TotalChars: 5}
	}
	return ui.ResultMsg{
		Result: metrics.Result{
			NetWPM: 106, RawWPM: 112, Accuracy: 96.4, Consistency: 83.2,
			CorrectChars: 268, DurationMs: 30000, PerSecond: per,
		},
		Mode: config.ModeTime, Length: 30,
	}
}

// The Result cases assign m.result directly instead of routing a ResultMsg:
// handleResultMsg persists to disk, and a layout test has no business writing
// the user's history.
func appCases() []appCase {
	return []appCase{
		{"home", baseModel},
		{"home/quit-prompt", func(w, h int) Model {
			m := baseModel(w, h)
			p := newQuitPrompt()
			m.quitPrompt = &p
			return m
		}},
		{"typing", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenTyping
			m.typing = typingModel(m, w, h)
			return m
		}},
		{"result", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenResult
			m.result = ui.NewResult(appResultMsg(), m.theme, m.keys).SetSize(w, h)
			return m
		}},
		{"result/persist-notice", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenResult
			m.result = ui.NewResult(appResultMsg(), m.theme, m.keys).SetSize(w, h)
			m.persistErr = "could not write history.json: permission denied"
			return m
		}},
		{"settings", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenSettings
			return m
		}},
		{"history", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenHistory
			return m
		}},
		{"codepaste", func(w, h int) Model {
			m := baseModel(w, h)
			m.screen = ScreenCodePaste
			m.codePaste = ui.NewCodePaste(m.theme).SetSize(w, h)
			return m
		}},
		{"transition/early", transitionModel(60)},
	}
}

// transitionModel parks the root mid-transition at a fixed clock offset, so the
// blended frame is measured rather than whichever half the wall clock lands in.
//
// Only the first half is worth a case. renderTransition is a colour crossfade
// in the default theme, so past the midpoint its ANSI-stripped output is
// byte-identical to the destination screen — a late case would re-measure the
// Result frame under a second name and imply coverage that does not exist.
func transitionModel(elapsedMs int64) func(w, h int) Model {
	return func(w, h int) Model {
		m := baseModel(w, h)
		m.screen = ScreenTyping
		m.typing = typingModel(m, w, h)
		from := m.composeScreen(ScreenTyping)

		m.screen = ScreenResult
		m.result = ui.NewResult(appResultMsg(), m.theme, m.keys).SetSize(w, h)
		m.transition = &transitionState{
			fromFrame: from, toScreen: ScreenResult,
			startMs: 0, durMs: transitionDurMs,
		}
		m.animNowMs = elapsedMs
		return m
	}
}
