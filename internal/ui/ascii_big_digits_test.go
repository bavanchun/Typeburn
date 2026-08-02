package ui

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// The hero block is the largest thing on the Result screen, and for three
// releases it rendered the wrong number: slot 0 held the "2" artwork, so
// BigDigits(0) drew a 2 and BigDigits(100) drew "122". The golden files
// recorded that as the baseline, because a golden test proves unchanged, never
// correct.
//
// These invariants are what a golden cannot give: properties that are true of
// the artwork itself.

// TestDigitGlyphs_Rectangular pins the contract the hero's column arithmetic
// depends on. Widths differ between digits — 1 is narrow, 0 and 6 are wide —
// but every row within one digit must be the same number of cells, or the block
// is ragged and anything positioned beside it shifts on 3, 6 and 9.
func TestDigitGlyphs_Rectangular(t *testing.T) {
	for d, glyph := range digitGlyphs {
		if len(glyph) != numRows {
			t.Errorf("digit %d has %d rows, want %d", d, len(glyph), numRows)
			continue
		}
		want := lipgloss.Width(glyph[0])
		for r, row := range glyph {
			if got := lipgloss.Width(row); got != want {
				t.Errorf("digit %d row %d is %d cells, row 0 is %d — glyph is not rectangular",
					d, r, got, want)
			}
		}
	}
}

// TestDigitGlyphs_Distinct is the assertion that would have caught the defect
// on the day it was introduced: two digits sharing artwork means one of them
// renders as the other.
func TestDigitGlyphs_Distinct(t *testing.T) {
	seen := map[string]int{}
	for d, glyph := range digitGlyphs {
		key := strings.Join(glyph, "\n")
		if prev, dup := seen[key]; dup {
			t.Errorf("digit %d is byte-identical to digit %d — one of them renders as the other", d, prev)
			continue
		}
		seen[key] = d
	}
}

// TestBigDigits_WidthIsSumOfGlyphs asserts the rendered width is exactly the
// glyphs plus one separator column between each pair. The hero and the reveal's
// fixed-width count-up both do this arithmetic; if it is wrong here it is wrong
// in both.
func TestBigDigits_WidthIsSumOfGlyphs(t *testing.T) {
	th := theme.Load("default", true)
	for _, n := range []int{0, 7, 60, 90, 100, 106, 200, 999} {
		want := 0
		for i, d := range decimalDigits(n) {
			if i > 0 {
				want++ // separator column
			}
			want += lipgloss.Width(digitGlyphs[d][0])
		}
		for r, row := range strings.Split(stripANSI(BigDigits(n, th)), "\n") {
			if got := lipgloss.Width(row); got != want {
				t.Errorf("BigDigits(%d) row %d is %d cells, want %d", n, r, got, want)
			}
		}
	}
}

// TestBigDigits_RendersItsOwnDigits asserts BigDigits(n) selects the right
// slots and joins them with exactly one separator column — that it drops no
// digit and invents none.
//
// It reads the same table it checks, so it cannot see wrong artwork in a slot:
// it passed while BigDigits(0) drew a 2. Distinctness and the recorded strip
// are what cover that; this covers the decompose-and-join logic around them.
func TestBigDigits_RendersItsOwnDigits(t *testing.T) {
	th := theme.Load("default", true)
	for _, n := range []int{0, 5, 60, 90, 100, 106, 302} {
		rows := make([]string, numRows)
		for r := range rows {
			var sb strings.Builder
			for i, d := range decimalDigits(n) {
				if i > 0 {
					sb.WriteString(" ")
				}
				sb.WriteString(digitGlyphs[d][r])
			}
			rows[r] = sb.String()
		}
		if got, want := stripANSI(BigDigits(n, th)), strings.Join(rows, "\n"); got != want {
			t.Errorf("BigDigits(%d) is not the concatenation of its digits\ngot:\n%s\nwant:\n%s", n, got, want)
		}
	}
}

// updateStrip is deliberately separate from the layout baselines' -update flag.
// This file is the only guard on slot identity — swap the 3 and 7 artwork and
// every other assertion here still passes — so a routine golden refresh after a
// layout change must not be able to silently re-bless it.
var updateStrip = flag.Bool("update-strip", false, "rewrite the big-digit artwork strip")

// TestBigDigits_Strip records all ten digits side by side in one reviewable
// file. Distinctness proves no two slots are equal; it cannot prove slot 7
// holds a seven. Only a human reading the artwork can, so the artwork is put
// somewhere a human will actually read it during review.
func TestBigDigits_Strip(t *testing.T) {
	got := stripANSI(BigDigits(1234567890, theme.Load("default", true))) + "\n"
	path := filepath.Join("testdata", "big_digits_strip.txt")

	if *updateStrip {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("write %s: %v", path, err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s (run with -update-strip to record): %v", path, err)
	}
	if got != string(want) {
		t.Errorf("%s changed; rerun with -update-strip and READ the artwork\n--- got ---\n%s", path, got)
	}
}
