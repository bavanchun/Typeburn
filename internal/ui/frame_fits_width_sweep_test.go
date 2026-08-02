package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
)

// TestFrameFits_WidthSweep walks every column from the supported minimum to 90,
// one at a time, for every screen.
//
// TestFrameFits samples seven widths, which is enough to catch a layout that is
// wrong everywhere and useless against one that is wrong at a single column.
// Both defects this replaces were of the second kind: the six-hint Home footer
// switched to its full 73-cell form at exactly 72, and the History rules were a
// constant 62 that only 60 and 61 could not hold. A sampled matrix walks
// straight past both.
//
// Width only, at a single height. Heights are TestFrameFits's business; this
// test exists for the off-by-one a sampled width matrix cannot see.
func TestFrameFits_WidthSweep(t *testing.T) {
	const h = 24

	for _, sc := range screenCases() {
		for w := 60; w <= 90; w++ {
			for i, ln := range renderScreen(t, sc, w, h) {
				if got := lipgloss.Width(ln); got > w {
					t.Errorf("%s at width %d: line %d is %d cells: %q", sc.name, w, i, got, ln)
				}
			}
		}
	}
}
