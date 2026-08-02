package ui

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// Sizes are scoped to what the product actually supports. The root model
// renders DegradedNotice below 60×20, so asserting fit at 40×15 would assert
// against a configuration the app deliberately refuses — permanently
// unfixable, and therefore a permanent exemption.
//
// 61 and 72 are not round numbers by accident: they are off-by-one boundaries
// where labels change form, and boundaries are where the layout breaks.
var (
	fitWidths  = []int{60, 61, 72, 80, 88, 120, 200}
	fitHeights = []int{20, 24, 30, 50}
)

// overflow records how a screen exceeds its terminal, so a fix that trades one
// dimension for the other cannot pass unnoticed. Zero means that dimension fits.
type overflow struct{ Lines, Width int }

func (o overflow) fits() bool { return o.Lines == 0 && o.Width == 0 }

// knownOverflow lives in frame_fits_known_overflow_test.go so the debt list can
// shrink without touching the assertions that police it.

func fitKey(name string, w, h int) string { return fmt.Sprintf("%s@%dx%d", name, w, h) }

// measureFrame returns how far a frame exceeds w×h. Width is measured with
// lipgloss.Width so a line of CJK is counted in cells, not runes.
func measureFrame(lines []string, w, h int) overflow {
	var o overflow
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

// TestFrameFits asserts every screen fits every supported terminal size.
//
// What the height assertion can and cannot prove: Home, Settings, History and
// CodePaste self-place via lipgloss.Place(w, h, ...), so len(lines) == h by
// construction and only the width measurement carries signal for them. Result
// and Typing place nothing themselves, so both dimensions are real there. Do
// not read a green run as proof that the four self-placing screens have room
// for their content — it proves only that they do not spill sideways.
func TestFrameFits(t *testing.T) {
	seen := map[string]bool{}
	measured := map[string]overflow{}

	for _, sc := range screenCases() {
		for _, w := range fitWidths {
			for _, h := range fitHeights {
				key := fitKey(sc.name, w, h)
				got := measureFrame(renderScreen(t, sc, w, h), w, h)
				if !got.fits() {
					measured[key] = got
				}
				want, listed := knownOverflow[key]
				if listed {
					seen[key] = true
				}

				switch {
				case !listed && !got.fits():
					t.Errorf("%s overflows %dx%d: %+v (unlisted — fix it, or record it in knownOverflow)",
						key, w, h, got)
				case listed && got.fits():
					t.Errorf("%s now fits %dx%d — delete its knownOverflow entry", key, w, h)
				case listed && got != want:
					t.Errorf("%s overflow changed: measured %+v, recorded %+v — update the entry",
						key, got, want)
				}
			}
		}
	}

	for key := range knownOverflow {
		if !seen[key] {
			t.Errorf("knownOverflow has %q, which no screen×size case produces — stale entry", key)
		}
	}

	// Report the measurements rather than making the next person transcribe
	// them from a wall of individual failures.
	if t.Failed() {
		t.Log("\nmeasured overflows:\n" + overflowLiteral(measured))
	}
}

// overflowLiteral renders measurements as the Go map literal knownOverflow
// expects, so a deliberate update is a paste rather than a transcription.
func overflowLiteral(m map[string]overflow) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("var knownOverflow = map[string]overflow{\n")
	for _, k := range keys {
		fmt.Fprintf(&sb, "\t%q: {Lines: %d, Width: %d},\n", k, m[k].Lines, m[k].Width)
	}
	sb.WriteString("}\n")
	return sb.String()
}

// TestFrameFits_LayoutIsThemeIndependent asserts mono and NO_COLOR change
// attributes only.
//
// The assertion is byte equality of the stripped frames, not equal widths.
// Equal widths would accept a theme that swapped █ for ▓ — same cell count,
// different picture — and "layout identical" is supposed to mean the user sees
// the same characters in the same places with the colour removed. It holds
// today for every case; the point is that it keeps holding while later phases
// rewrite these screens.
func TestFrameFits_LayoutIsThemeIndependent(t *testing.T) {
	for _, sc := range screenCases() {
		for _, w := range fitWidths {
			for _, h := range fitHeights {
				want := renderScreen(t, sc, w, h)
				for _, alt := range []struct {
					name string
					th   theme.Theme
				}{
					{"mono", theme.Load("mono", false)},
					{"no-color", theme.Load("default", true)},
				} {
					got := renderScreenTheme(t, sc, alt.th, w, h)
					if len(got) != len(want) {
						t.Errorf("%s: %s has %d lines, default has %d",
							fitKey(sc.name, w, h), alt.name, len(got), len(want))
						continue
					}
					for i := range want {
						if got[i] != want[i] {
							t.Errorf("%s line %d: %s renders %q, default renders %q",
								fitKey(sc.name, w, h), i, alt.name, got[i], want[i])
						}
					}
				}
			}
		}
	}
}
