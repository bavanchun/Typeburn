package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// viewportSizes are terminal sizes the window height differs at, so a pass at
// one of them says nothing about the others: the window is derived from the
// terminal height, and the wrap width from its width.
var viewportSizes = [][2]int{{60, 20}, {80, 24}, {120, 32}}

// longRunTyping is a model whose target is several windows long at every size
// under test — the shape of run the stream used to render in full and the
// terminal then clipped. The audit's caret went off screen around 950 runes at
// 60x20; this is longer than that, and short enough that a per-keystroke sweep
// over it stays a second or two.
func longRunTyping(th theme.Theme, w, h int) TypingModel {
	return newTypingWithSeed(config.ModeWords, 180, words.QuoteShort,
		th, config.DefaultKeymap(), false, false, false, false, 42).SetSize(w, h)
}

// caretToken is the exact styled cell the cursor renders as. With blink off the
// caret cell takes no animation override, so this string appears in the frame
// if and only if the caret's row is on screen.
func caretToken(m TypingModel, th theme.Theme) string {
	states := m.eng.States()
	i := indexOfCurrent(states)
	if i < 0 {
		return ""
	}
	return th.Style(theme.RoleCursorBg).Render(string(runeAtIndex(i, []rune(m.target), m.eng.Typed())))
}

// TestTypingView_CaretVisibleAtEveryKeystroke types a whole target one rune at
// a time and asserts the cursor is on screen after every single one, at three
// sizes. The window height is derived from the terminal height, so a pass at
// 24 rows proves nothing about 32.
//
// Past roughly a screenful the stream used to run off the bottom of the cell
// buffer, taking the footer and then the caret with it: the user carried on
// typing with no idea where they were.
func TestTypingView_CaretVisibleAtEveryKeystroke(t *testing.T) {
	th := theme.Load("default", false)

	for _, sz := range viewportSizes {
		w, h := sz[0], sz[1]
		m := longRunTyping(th, w, h)
		target := []rune(m.target)
		if len(target) < 400 {
			t.Fatalf("fixture is only %d runes — too short to leave any window", len(target))
		}

		for i := 0; i <= len(target); i++ {
			frame := m.View()
			lines := strings.Split(frame, "\n")
			if len(lines) > h {
				t.Fatalf("%dx%d after %d keystrokes: frame is %d lines", w, h, i, len(lines))
			}
			if tok := caretToken(m, th); tok != "" && !strings.Contains(frame, tok) {
				t.Fatalf("%dx%d after %d keystrokes: the caret is off screen", w, h, i)
			}
			if i < len(target) {
				m, _ = m.Update(pressText(string(target[i])))
			}
		}
	}
}

// TestTypingView_FitsAndKeepsItsFooter asserts the footer survives at every
// size, in the default Time mode as well as Words. The stream is what grows
// without bound, so the footer is the first thing an unwindowed stream pushes
// out of the buffer — and Time 30 is what the app opens with.
func TestTypingView_FitsAndKeepsItsFooter(t *testing.T) {
	th := theme.Load("default", false)
	models := map[string]func(theme.Theme, int, int) TypingModel{
		"words": longRunTyping,
		"time30": func(th theme.Theme, w, h int) TypingModel {
			return newTypingWithSeed(config.ModeTime, 30, words.QuoteShort,
				th, config.DefaultKeymap(), false, false, false, false, 42).SetSize(w, h)
		},
	}
	for name, build := range models {
		for _, sz := range viewportSizes {
			w, h := sz[0], sz[1]
			lines := strings.Split(build(th, w, h).View(), "\n")
			if len(lines) > h {
				t.Errorf("%s %dx%d: frame is %d lines", name, w, h, len(lines))
			}
			if last := stripANSI(lines[len(lines)-1]); !strings.Contains(last, "tab") {
				t.Errorf("%s %dx%d: last line is not the footer: %q", name, w, h, last)
			}
		}
	}
}

// TestTypingView_WideRuneCodeTargetFitsItsTerminal asserts a Code target made
// entirely of double-width runes stays inside the terminal. --text and the
// paste screen both accept whatever the user has, and a buffer of CJK or emoji
// drew at twice the width it was given until rows were counted in cells.
func TestTypingView_WideRuneCodeTargetFitsItsTerminal(t *testing.T) {
	th := theme.Load("default", false)
	for _, src := range []string{strings.Repeat("函", 120), strings.Repeat("🚀", 60)} {
		for _, sz := range viewportSizes {
			w, h := sz[0], sz[1]
			m := NewTypingCode(src, th, config.DefaultKeymap(), false, false).SetSize(w, h)
			for i, line := range strings.Split(stripANSI(m.View()), "\n") {
				if got := lipgloss.Width(line); got > w {
					t.Errorf("%.4q at %dx%d: line %d is %d cells wide", src, w, h, i, got)
				}
			}
		}
	}
}

// unanimatedWindow renders the stream the way View does but with the caret
// animation disabled, so the rows are byte-comparable with wrapTokens output.
func unanimatedWindow(m TypingModel, th theme.Theme) (window, rows []string, caretRow int) {
	states := m.eng.States()
	target, typed := []rune(m.target), m.eng.Typed()
	cw := ContentWidth(m.w, WidthTier(m.w, m.h))

	// The cache holds the tokens the render just built, so the rows are wrapped
	// from the same bytes rather than a second, independently styled copy.
	cache := &streamTokenCache{}
	out := renderWordStreamAnim(states, target, typed, cw, wordStreamHeight(m.h), th, disabledCaret(), cache)
	rows, caretRow = wrapTokens(cache.base, states, target, typed, cw)
	return strings.Split(out, "\n"), rows, caretRow
}

// windowStart returns where window sits in rows, or -1 when it is not a
// contiguous run of them.
func windowStart(rows, window []string) int {
	for i := 0; i+len(window) <= len(rows); i++ {
		match := true
		for j := range window {
			if rows[i+j] != window[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}

// TestTypingView_WindowFollowsTheCaretOneRowAtATime asserts the window is a
// contiguous run of the wrapped rows, always holds the caret's row, and never
// jumps by more than one row for one keystroke — forwards or backwards over a
// backspace. A window that leapt would lose the reader's place as surely as one
// that left the caret behind.
func TestTypingView_WindowFollowsTheCaretOneRowAtATime(t *testing.T) {
	th := theme.Load("default", false)
	m := longRunTyping(th, 60, 20)
	target := []rune(m.target)
	height := wordStreamHeight(20)

	// A prefix is enough: it crosses the first scroll and many more after it,
	// and every later row is the same arithmetic.
	walk := 500
	if walk > len(target) {
		walk = len(target)
	}

	prev := 0
	step := func(i int) {
		window, rows, caretRow := unanimatedWindow(m, th)
		start := windowStart(rows, window)
		switch {
		case start < 0:
			t.Fatalf("keystroke %d: the window is not a contiguous run of rows", i)
		case len(rows) > height && len(window) != height:
			t.Fatalf("keystroke %d: window is %d rows, budget is %d", i, len(window), height)
		case caretRow >= 0 && (caretRow < start || caretRow >= start+len(window)):
			t.Fatalf("keystroke %d: caret row %d outside window [%d,%d)", i, caretRow, start, start+len(window))
		case start-prev > 1 || prev-start > 1:
			t.Fatalf("keystroke %d: window jumped from row %d to row %d", i, prev, start)
		}
		prev = start
	}

	for i := 0; i < walk; i++ {
		m, _ = m.Update(pressText(string(target[i])))
		step(i)
	}
	// Walk back out the same way. Backspacing must not scroll further than
	// typing did, and must never strand the caret above the window.
	for i := walk - 1; i >= 0; i-- {
		m, _ = m.Update(press(tea.KeyBackspace, 0))
		step(i)
	}
}

// TestTypingView_WindowIsAFunctionOfTheRunSoFar asserts the frame after typing
// forward past a wrap and backspacing over it equals the frame first reached at
// that same point. Scroll state that remembered how it got there could sit at a
// different offset for identical content, which is what makes a window feel
// like it is drifting.
func TestTypingView_WindowIsAFunctionOfTheRunSoFar(t *testing.T) {
	th := theme.Load("default", false)
	m := longRunTyping(th, 60, 20)
	target := []rune(m.target)

	// Type well past the first scroll, remembering the frame at each step.
	const walk = 250
	frames := make([]string, 0, walk)
	for i := 0; i < walk; i++ {
		window, _, _ := unanimatedWindow(m, th)
		frames = append(frames, strings.Join(window, "\n"))
		m, _ = m.Update(pressText(string(target[i])))
	}
	for i := walk - 1; i >= 0; i-- {
		m, _ = m.Update(press(tea.KeyBackspace, 0))
		window, _, _ := unanimatedWindow(m, th)
		if got := strings.Join(window, "\n"); got != frames[i] {
			t.Fatalf("backspacing to keystroke %d shows a different window than typing to it did", i)
		}
	}
}
