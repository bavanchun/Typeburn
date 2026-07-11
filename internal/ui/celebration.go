package ui

import (
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// New-best celebration timing/shape. The burst rides the same frame loop as the
// result reveal and is one-shot. celebrateMs bounds the active window; particles
// twinkle on a short sub-cycle within their individual lifetimes.
const (
	celebrateMs    int64 = 1000
	particleCount        = 6
	particleLifeMs int64 = 300
	bandRadius           = 3 // a blank row counts if within this many rows of content
)

// celebrationGlyphs are display-width-1 sparkle runes (asserted in tests). They
// are spliced onto blank padding cells, so width must stay 1 to preserve layout.
var celebrationGlyphs = []rune{'*', '+', '·', '.'}

// overlayCell is a styled width-1 glyph to splat at (row, col) in the frame.
type overlayCell struct {
	row, col int
	glyph    string
}

// applyCelebration overlays a sparkle burst onto blank margin rows of an already
// composed, placed frame when the burst is active. It only ever rewrites
// all-blank rows (never styled content), and rebuilds each touched row to the
// exact same rune width — so the overlay is layout-identical (line count + width
// preserved; only blank cells gain a glyph). Returns the frame unchanged when no
// particle is live.
func applyCelebration(frame string, startMs, nowMs int64, th theme.Theme) string {
	if startMs <= 0 || nowMs < startMs || nowMs >= startMs+celebrateMs {
		return frame
	}
	lines := strings.Split(frame, "\n")
	band := blankBand(lines)
	if len(band) == 0 {
		return frame
	}
	w := len([]rune(lines[band[0]])) // blank rows are pure spaces → rune count = width
	cells := celebrationCells(startMs, nowMs, band, w, th)
	if len(cells) == 0 {
		return frame
	}

	byRow := make(map[int]map[int]string, len(cells))
	for _, c := range cells {
		if byRow[c.row] == nil {
			byRow[c.row] = make(map[int]string)
		}
		byRow[c.row][c.col] = c.glyph
	}
	for row, cols := range byRow {
		lines[row] = overlayBlankRow(len([]rune(lines[row])), cols)
	}
	return strings.Join(lines, "\n")
}

// celebrationCells places up to particleCount sparkles deterministically from
// (index, startMs) — no math/rand global — so renders are reproducible for
// goldens. Each particle has a staggered birth and a short life; it twinkles
// through the glyph set while alive. Columns cluster in the centre band so the
// burst reads as celebrating the result card rather than scattering edge-to-edge.
func celebrationCells(startMs, nowMs int64, band []int, w int, th theme.Theme) []overlayCell {
	style := celebrationStyle(th)
	colLo := w / 4
	colSpan := w / 2
	if colSpan < 1 {
		colSpan = 1
	}

	cells := make([]overlayCell, 0, particleCount)
	for i := 0; i < particleCount; i++ {
		h := particleHash(i, startMs)
		born := int64(h % 200) // 0–200ms stagger
		age := nowMs - startMs - born
		if age < 0 || age >= particleLifeMs {
			continue
		}
		row := band[int(h>>8)%len(band)]
		col := colLo + int(h>>16)%colSpan
		gi := int((nowMs/80 + int64(i)) % int64(len(celebrationGlyphs)))
		cells = append(cells, overlayCell{row: row, col: col, glyph: style.Render(string(celebrationGlyphs[gi]))})
	}
	return cells
}

// celebrationStyle is the sparkle style: success color, or an attribute-only
// twinkle (bold + reverse) under NO_COLOR so the burst stays layout-identical.
func celebrationStyle(th theme.Theme) lipgloss.Style {
	if th.Color(theme.RoleSuccess) == nil {
		return lipgloss.NewStyle().Bold(true).Reverse(true)
	}
	return th.Style(theme.RoleSuccess)
}

// particleHash is a small deterministic mixer over (index, startMs).
func particleHash(i int, startMs int64) uint64 {
	h := uint64(startMs)*1099511628211 ^ uint64(i+1)*2654435761
	h ^= h >> 17
	h *= 0xff51afd7ed558ccd
	h ^= h >> 33
	return h
}

// blankBand returns indices of all-blank rows that sit within bandRadius of a
// content row — i.e. the padding hugging the result card, not far-flung edges.
func blankBand(lines []string) []int {
	blank := make([]bool, len(lines))
	for i, ln := range lines {
		blank[i] = ln != "" && strings.Trim(ln, " ") == ""
	}
	var band []int
	for i := range lines {
		if !blank[i] {
			continue
		}
		for d := 1; d <= bandRadius; d++ {
			if (i-d >= 0 && !blank[i-d]) || (i+d < len(lines) && !blank[i+d]) {
				band = append(band, i)
				break
			}
		}
	}
	return band
}

// overlayBlankRow rebuilds a blank row of the given rune width with styled glyphs
// at the requested columns and spaces elsewhere — exact width, no reflow.
func overlayBlankRow(width int, cols map[int]string) string {
	var b strings.Builder
	for col := 0; col < width; col++ {
		if g, ok := cols[col]; ok {
			b.WriteString(g)
		} else {
			b.WriteByte(' ')
		}
	}
	return b.String()
}
