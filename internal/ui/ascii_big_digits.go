package ui

import (
	"strings"

	"github.com/bavanchun/Typeburn/internal/theme"
)

// digitGlyphs maps each decimal digit 0-9 to a 5-row ASCII block-art glyph.
// Each glyph is 6 characters wide (including trailing space) for uniform column
// joining. The technique mirrors ascii-logo.go: render per-row, join columns.
var digitGlyphs = [10][]string{
	{ // 0
		"██████╗ ",
		"╚════██╗",
		" █████╔╝",
		"██╔═══╝ ",
		"███████╗",
		"╚══════╝",
	},
	{ // 1
		" ██╗",
		"███║",
		"╚██║",
		" ██║",
		" ██║",
		" ╚═╝",
	},
	{ // 2
		"██████╗ ",
		"╚════██╗",
		" █████╔╝",
		"██╔═══╝ ",
		"███████╗",
		"╚══════╝",
	},
	{ // 3
		"██████╗ ",
		"╚════██╗",
		" █████╔╝",
		" ╚════██╗",
		"██████╔╝",
		"╚═════╝ ",
	},
	{ // 4
		"██╗  ██╗",
		"██║  ██║",
		"███████║",
		"╚════██║",
		"     ██║",
		"     ╚═╝",
	},
	{ // 5
		"███████╗",
		"██╔════╝",
		"███████╗",
		"╚════██║",
		"███████║",
		"╚══════╝",
	},
	{ // 6
		" ██████╗",
		"██╔════╝",
		"███████╗",
		"██╔═══██╗",
		"╚██████╔╝",
		" ╚═════╝ ",
	},
	{ // 7
		"███████╗",
		"╚════██║",
		"    ██╔╝",
		"   ██╔╝ ",
		"   ██║  ",
		"   ╚═╝  ",
	},
	{ // 8
		" █████╗ ",
		"██╔══██╗",
		"╚█████╔╝",
		"██╔══██╗",
		"╚█████╔╝",
		" ╚════╝ ",
	},
	{ // 9
		" █████╗ ",
		"██╔══██╗",
		"╚██████║",
		" ╚════██║",
		" █████╔╝",
		" ╚════╝ ",
	},
}

// numRows is the height of each digit glyph (all glyphs share this height).
const numRows = 6

// BigDigits renders n as concatenated ASCII block-art digits styled with
// RoleAccent Bold. n must be >= 0; negative values render as "0".
// Returns a multi-line string suitable for embedding in a panel.
func BigDigits(n int, th theme.Theme) string {
	if n < 0 {
		n = 0
	}
	style := th.Style(theme.RoleAccent).Bold(true)

	// Decompose n into decimal digits (left-to-right).
	digits := decimalDigits(n)

	// Build each row by concatenating the corresponding glyph row of each digit.
	rows := make([]string, numRows)
	for row := range rows {
		var sb strings.Builder
		for i, d := range digits {
			glyph := digitGlyphs[d]
			if row < len(glyph) {
				sb.WriteString(glyph[row])
			}
			// Add a single column of space between digits (except after last).
			if i < len(digits)-1 {
				sb.WriteString(" ")
			}
		}
		rows[row] = style.Render(sb.String())
	}

	return strings.Join(rows, "\n")
}

// decimalDigits returns the decimal digits of n from most-significant to
// least-significant. n == 0 returns []int{0}.
func decimalDigits(n int) []int {
	if n == 0 {
		return []int{0}
	}
	var ds []int
	for n > 0 {
		ds = append([]int{n % 10}, ds...)
		n /= 10
	}
	return ds
}
