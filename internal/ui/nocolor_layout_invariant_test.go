package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// The animation system's core invariant: under NO_COLOR every animated moment is
// layout-identical to its static frame — same line count and same per-line rune
// width at every point in the animation. Only attributes (and, for the
// celebration, blank-cell glyph content) may differ. This consolidates the
// guarantee across the word-stream caret and the result reveal + celebration;
// the screen transition has its own invariant test in package app.

func noColorTheme() theme.Theme { return theme.Load("default", true) }

func assertLayoutIdentical(t *testing.T, label, want, got string) {
	t.Helper()
	wl := strings.Split(stripANSI(want), "\n")
	gl := strings.Split(stripANSI(got), "\n")
	if len(wl) != len(gl) {
		t.Fatalf("%s: line count %d != %d", label, len(gl), len(wl))
	}
	for i := range wl {
		if len([]rune(wl[i])) != len([]rune(gl[i])) {
			t.Fatalf("%s: line %d width %d != %d", label, i, len([]rune(gl[i])), len([]rune(wl[i])))
		}
	}
}

func TestNoColorInvariant_WordStreamCaret(t *testing.T) {
	th := noColorTheme()
	target := runesOf("the quick brown fox jumps over the lazy dog")
	typed := runesOf("the quick brown")
	states := statesTyped(15, len(target))

	static := RenderWordStream(states, target, typed, 40, th)
	for _, off := range []int64{0, 40, 80, 120, 160, 300, 600} {
		ca := caretAnim{nowMs: 100000 + off, lastKeyMs: 100000, blinkOn: off%80 < 40, cursorIdx: 15}
		got := renderWordStreamAnim(states, target, typed, 40, th, ca, &streamTokenCache{})
		assertLayoutIdentical(t, "caret@"+itoa(off), static, got)
	}
}

func TestNoColorInvariant_ResultRevealAndCelebration(t *testing.T) {
	th := noColorTheme()
	settled := func() ResultModel {
		m := newTestResult().WithBest(true)
		m.th = th
		return m
	}
	staticFrame := settled().View() // no reveal armed → static

	for _, off := range []int64{0, 100, 250, 400, 600, 800, 1000} {
		m := newTestResult().WithBest(true)
		m.th = th
		m = m.WithRevealStart(1000)
		m.nowMs = 1000 + off
		assertLayoutIdentical(t, "result@"+itoa(off), staticFrame, m.View())
	}
}

// itoa avoids importing strconv just for labels.
func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
