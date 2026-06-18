package app

import (
	"regexp"
	"strings"
	"testing"
)

var transAnsiRE = regexp.MustCompile("\x1b\\[[0-9;]*m")

func transStrip(s string) string { return transAnsiRE.ReplaceAllString(s, "") }

// frame builds a 4-line test frame with the given marker rune filling each line.
func transFrame(marker string, lines, width int) string {
	row := strings.Repeat(marker, width)
	out := make([]string, lines)
	for i := range out {
		out[i] = row
	}
	return strings.Join(out, "\n")
}

// NO_COLOR wipe reveals exactly floor(rows*p) incoming rows from the top.
func TestRenderTransition_WipeRowMath(t *testing.T) {
	from := transFrame("F", 4, 6)
	to := transFrame("T", 4, 6)

	cases := []struct {
		p       float64
		topIsTo int // number of leading rows that should be the incoming frame
	}{
		{0.0, 0}, {0.25, 1}, {0.5, 2}, {0.75, 3}, {1.0, 4},
	}
	for _, c := range cases {
		out := renderTransition(from, to, c.p, true)
		lines := strings.Split(out, "\n")
		if len(lines) != 4 {
			t.Fatalf("p=%v line count=%d want 4", c.p, len(lines))
		}
		for i, ln := range lines {
			wantTo := i < c.topIsTo
			isTo := strings.Contains(ln, "T")
			if wantTo != isTo {
				t.Errorf("p=%v line %d: isTo=%v want %v", c.p, i, isTo, wantTo)
			}
		}
	}
}

// Both wipe and crossfade preserve line count and per-line rune width.
func TestRenderTransition_PreservesLayout(t *testing.T) {
	from := transFrame("F", 5, 8)
	to := transFrame("T", 5, 8)

	for _, noColor := range []bool{true, false} {
		for _, p := range []float64{0, 0.3, 0.5, 0.7, 1} {
			out := renderTransition(from, to, p, noColor)
			lines := strings.Split(transStrip(out), "\n")
			if len(lines) != 5 {
				t.Fatalf("noColor=%v p=%v line count=%d want 5", noColor, p, len(lines))
			}
			for i, ln := range lines {
				if w := len([]rune(ln)); w != 8 {
					t.Errorf("noColor=%v p=%v line %d width=%d want 8", noColor, p, i, w)
				}
			}
		}
	}
}

// The color crossfade swaps which frame shows at the midpoint.
func TestRenderTransition_CrossfadeSwapsAtMidpoint(t *testing.T) {
	from := transFrame("F", 3, 5)
	to := transFrame("T", 3, 5)

	if got := transStrip(renderTransition(from, to, 0.49, false)); !strings.Contains(got, "F") {
		t.Error("p<0.5 should show the outgoing frame")
	}
	if got := transStrip(renderTransition(from, to, 0.51, false)); !strings.Contains(got, "T") {
		t.Error("p>=0.5 should show the incoming frame")
	}
}

// transitionActive is the frame-loop liveness signal: true only within the window.
func TestTransitionActive_Window(t *testing.T) {
	m := newTestModel()
	if m.transitionActive(0) {
		t.Error("nil transition should be inactive")
	}
	m.transition = &transitionState{startMs: 1000, durMs: transitionDurMs}
	if !m.transitionActive(1000) {
		t.Error("at start should be active")
	}
	if !m.transitionActive(1000 + transitionDurMs - 1) {
		t.Error("inside window should be active")
	}
	if m.transitionActive(1000 + transitionDurMs) {
		t.Error("at end should be inactive (self-stop)")
	}
}

// An expired transition is ignored by View (derived expiry) and nil-ed out on
// the next message in Update.
func TestTransition_ExpiresCleanly(t *testing.T) {
	m := newTestModel()
	m.w, m.h = 80, 24
	m.transition = &transitionState{
		fromFrame: "stale", toScreen: ScreenResult, startMs: 1000, durMs: transitionDurMs,
	}
	m.animNowMs = 1000 + transitionDurMs + 5 // past the window

	// View must not surface the stale frame.
	if strings.Contains(transStrip(m.View().Content), "stale") {
		t.Error("expired transition should be ignored by View")
	}
	// Next non-frame message nil-outs it.
	next, _ := m.Update(press('1', 0))
	if next.(Model).transition != nil {
		t.Error("expired transition should be nil-ed out on the next message")
	}
}
