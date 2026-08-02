package app

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// The root frame is what the user actually sees: it is the only place the
// degraded gate, the quit overlay, a mid-flight transition and the persistence
// toast exist. A screen that fits when rendered alone can still overflow here.
//
// Sizes match the ui-package harness, plus one cell below the 60×20 gate to
// prove the degraded notice itself fits the terminal it apologises for.
var (
	appFitWidths  = []int{59, 60, 61, 72, 80, 88, 120, 200}
	appFitHeights = []int{19, 20, 24, 30, 50}
)

type appOverflow struct{ Lines, Width int }

func (o appOverflow) fits() bool { return o.Lines == 0 && o.Width == 0 }

func appFitKey(name string, w, h int) string { return fmt.Sprintf("%s@%dx%d", name, w, h) }

func measureAppFrame(frame string, w, h int) appOverflow {
	var o appOverflow
	lines := strings.Split(stripANSI(frame), "\n")
	if len(lines) > h {
		o.Lines = len(lines)
	}
	for _, ln := range lines {
		if got := lipgloss.Width(ln); got > w && got > o.Width {
			o.Width = got
		}
	}
	return o
}

// TestAppFrameFits asserts the composed root frame fits the terminal at every
// supported size, and below the degraded gate as well — DegradedNotice is the
// one thing that must never itself overflow.
//
// Below the gate every case renders the same notice, so those cells prove the
// notice, not the screen.
func TestAppFrameFits(t *testing.T) {
	measured := map[string]appOverflow{}

	for _, ac := range appCases() {
		for _, w := range appFitWidths {
			for _, h := range appFitHeights {
				key := appFitKey(ac.name, w, h)
				got := measureAppFrame(ac.build(w, h).View().Content, w, h)
				if !got.fits() {
					measured[key] = got
					t.Errorf("%s overflows %dx%d: %+v", key, w, h, got)
				}
			}
		}
	}

	if t.Failed() {
		keys := make([]string, 0, len(measured))
		for k := range measured {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		var sb strings.Builder
		sb.WriteString("\nmeasured overflows:\nmap[string]appOverflow{\n")
		for _, k := range keys {
			fmt.Fprintf(&sb, "\t%q: {Lines: %d, Width: %d},\n", k, measured[k].Lines, measured[k].Width)
		}
		sb.WriteString("}\n")
		t.Log(sb.String())
	}
}

// TestAppFrameFits_DegradedGate pins the boundary itself: one cell below the
// gate the notice must appear, and at the gate it must not.
func TestAppFrameFits_DegradedGate(t *testing.T) {
	for _, tc := range []struct {
		w, h     int
		degraded bool
	}{
		{59, 24, true}, {60, 19, true}, {59, 19, true}, {60, 20, false}, {80, 24, false},
	} {
		// Matched on the notice's own copy, not on a re-placed block: the root
		// pads every line to centre it, so the rendered notice is never a
		// substring of the frame it appears in.
		frame := stripANSI(baseModel(tc.w, tc.h).View().Content)
		got := strings.Contains(frame, "Terminal too small") &&
			strings.Contains(frame, fmt.Sprintf("current %d×%d", tc.w, tc.h))
		if got != tc.degraded {
			t.Errorf("%dx%d: degraded notice present=%v, want %v", tc.w, tc.h, got, tc.degraded)
		}
	}
}
