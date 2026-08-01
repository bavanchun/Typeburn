package ui

import "math"

// Braille cell rendering for the WPM line. A braille cell packs 2×4 dots, so a
// chart row of N cells addresses a 2N×4 dot grid — four times the vertical
// resolution a plain character grid would give.

// brailleBits maps a cell-local dot coordinate [column 0..1][row 0..3] to the
// braille dot bit, per the standard 2×4 layout (dots 1..8).
var brailleBits = [2][4]byte{
	{0x01, 0x02, 0x04, 0x40}, // left column → dots 1,2,3,7
	{0x08, 0x10, 0x20, 0x80}, // right column → dots 4,5,6,8
}

// drawSeg connects (x0,y0)-(x1,y1) on the dot grid by sampling densely enough to
// fill both the vertical and horizontal spans, so flat segments render solid and
// steep segments have no vertical gaps.
func drawSeg(dots [][]bool, x0, y0, x1, y1 int) {
	span := y1 - y0
	if span < 0 {
		span = -span
	}
	if dx := x1 - x0; dx > span {
		span = dx
	}
	if span < 2 {
		span = 2
	}
	for s := 0; s <= span; s++ {
		t := float64(s) / float64(span)
		x := int(math.Round(float64(x0) + t*float64(x1-x0)))
		y := int(math.Round(float64(y0) + t*float64(y1-y0)))
		if y >= 0 && y < len(dots) && x >= 0 && x < len(dots[y]) {
			dots[y][x] = true
		}
	}
}

// brailleAt packs the 2×4 dot block at cell (cr,cc) into a braille rune; returns
// 0 when the block is empty so the caller emits a plain space of equal width.
func brailleAt(dots [][]bool, cr, cc int) rune {
	var b byte
	for dx := 0; dx < 2; dx++ {
		for dy := 0; dy < 4; dy++ {
			r, c := cr*4+dy, cc*2+dx
			if r < len(dots) && c < len(dots[r]) && dots[r][c] {
				b |= brailleBits[dx][dy]
			}
		}
	}
	if b == 0 {
		return 0
	}
	return rune(brailleBase) + rune(b)
}

// Axis labels, tick joins, and scale helpers live in result_graph_axes.go.
