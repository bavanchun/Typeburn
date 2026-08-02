package ui

import (
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// sparkBars are the 8 unicode block elements from lowest to highest, used by
// the inline History trend sparkline (the Result screen draws a braille line
// graph instead — see result_graph.go).
var sparkBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// trendPrefix and the "last N tests" suffix are the fixed text either side of
// the sparkline; the bars get whatever cells are left.
const trendPrefix = "trend  "

// trendLine renders the History trend row, sized to the terminal.
//
// The sparkline used to emit one cell per record, so a full 200-record history
// produced a 223-cell line. That is not merely a line that spills: History is
// centred with lipgloss.Place, which sizes the block from its widest line, so a
// single over-wide row pushes every other row off centre and cuts the label at
// the end of this one. The bars are downsampled to the space that is actually
// available instead.
//
// termW <= 0 means the model has not been sized yet; there is no budget to fit,
// so every record is drawn.
func trendLine(vals []float64, total, termW int, th theme.Theme) string {
	suffix := "  last " + histItoa(total) + " tests"

	cells := len(vals)
	if termW > 0 {
		cells = termW - lipgloss.Width(trendPrefix) - lipgloss.Width(suffix)
		if cells < 0 {
			cells = 0
		}
	}

	bars := sparklineInline(downsampleSpark(vals, cells), th)
	return th.Style(theme.RoleTextMuted).Render(trendPrefix + bars + suffix)
}

// downsampleSpark reduces vals to at most cells values by averaging contiguous
// groups, so a long history is compressed rather than clipped: the trend keeps
// its shape and the most recent record still owns the rightmost bar.
func downsampleSpark(vals []float64, cells int) []float64 {
	if cells <= 0 || len(vals) == 0 {
		return nil
	}
	if len(vals) <= cells {
		return vals
	}

	out := make([]float64, cells)
	for i := range out {
		lo := i * len(vals) / cells
		hi := (i + 1) * len(vals) / cells
		if hi <= lo {
			hi = lo + 1
		}
		var sum float64
		for _, v := range vals[lo:hi] {
			sum += v
		}
		out[i] = sum / float64(hi-lo)
	}
	return out
}

// sparklineInline renders a compact single-row sparkline string for the trend
// label. It uses only the bar characters (no axis) for inline display.
func sparklineInline(vals []float64, th theme.Theme) string {
	if len(vals) == 0 {
		return ""
	}
	minV, maxV := vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		minV = 0
		if maxV == 0 {
			maxV = 1
		}
	}
	bars := make([]rune, len(vals))
	for i, v := range vals {
		ratio := (v - minV) / (maxV - minV)
		idx := int(ratio*float64(len(sparkBars)-1) + 0.5)
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBars) {
			idx = len(sparkBars) - 1
		}
		bars[i] = sparkBars[idx]
	}
	return th.Style(theme.RoleAccent).Render(string(bars))
}

// histItoa converts a non-negative int to a decimal string without importing
// fmt, which the History view otherwise has no use for.
func histItoa(n int) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[pos:])
}
