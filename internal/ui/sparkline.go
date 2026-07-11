package ui

import (
	"fmt"
	"math"
	"strings"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// sparkBars are the 8 unicode block elements from lowest to highest.
var sparkBars = []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// Sparkline renders a WPM-over-time chart from per-second raw-WPM values.
// Bars are drawn in RoleAccent; y-axis ticks and baseline in RoleTextFaint.
//
// chartH is the number of bar rows (default 4 if 0). width is the available
// content width for the entire chart block (axis label + bars).
//
// Edge cases:
//   - len(vals) == 0: returns empty string
//   - len(vals) == 1: renders a single full bar
//   - all values equal: renders all mid-height bars
func Sparkline(vals []float64, width, chartH int, th theme.Theme) string {
	return sparklineVisible(vals, width, chartH, len(vals), th)
}

// sparklineVisible renders the chart with only the first `visible` bars drawn;
// positions at/after `visible` are blanked to equal-width spaces so the layout
// never reflows. The Result reveal animates `visible` from 0 → len(vals); the
// public Sparkline passes len(vals), so the fully-revealed frame is byte-
// identical to the static render — one code path, no duplicated layout to drift.
func sparklineVisible(vals []float64, width, chartH, visible int, th theme.Theme) string {
	if len(vals) == 0 {
		return ""
	}
	if chartH <= 0 {
		chartH = 4
	}
	if visible < 0 {
		visible = 0
	}

	accentStyle := th.Style(theme.RoleAccent)
	faintStyle := th.Style(theme.RoleTextFaint)

	minV, maxV := minMax(vals)
	// When all values are equal, spread to show mid-height bar.
	if maxV == minV {
		minV = 0
		if maxV == 0 {
			maxV = 1
		}
	}

	// Build the bar string (one char per sample, single-row sparkline style).
	// Positions at/after `visible` render as a space (not yet drawn in).
	bars := make([]rune, len(vals))
	for i, v := range vals {
		if i >= visible {
			bars[i] = ' '
			continue
		}
		ratio := (v - minV) / (maxV - minV)
		idx := int(math.Round(ratio * float64(len(sparkBars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(sparkBars) {
			idx = len(sparkBars) - 1
		}
		bars[i] = sparkBars[idx]
	}

	// Y-axis labels: top = maxV, bottom = minV. Use 4-char wide labels.
	topLabel := fmt.Sprintf("%4.0f", maxV)
	midLabel := fmt.Sprintf("%4.0f", (minV+maxV)/2)
	botLabel := fmt.Sprintf("%4.0f", minV)

	barStr := accentStyle.Render(string(bars))
	pipe := faintStyle.Render("┤")
	baseline := faintStyle.Render("┼" + strings.Repeat("─", len(vals)) + " s")

	// Build the multi-row chart.
	// Row 0: top tick
	// Row 1: mid tick  (only if chartH >= 3)
	// Row 2: bar row
	// Row 3: baseline with x-axis ticks
	var sb strings.Builder

	// Top row: max label + bar
	sb.WriteString(faintStyle.Render(topLabel))
	sb.WriteString(pipe)
	sb.WriteString(strings.Repeat(" ", len(vals))) // empty top row
	sb.WriteString("\n")

	// Middle row (optional)
	if chartH >= 3 {
		sb.WriteString(faintStyle.Render(midLabel))
		sb.WriteString(pipe)
		sb.WriteString(strings.Repeat(" ", len(vals)))
		sb.WriteString("\n")
	}

	// Bar row
	sb.WriteString(faintStyle.Render(botLabel))
	sb.WriteString(pipe)
	sb.WriteString(barStr)
	sb.WriteString("\n")

	// Baseline
	sb.WriteString(faintStyle.Render("    "))
	sb.WriteString(baseline)

	// X-axis second labels
	sb.WriteString("\n")
	sb.WriteString(faintStyle.Render("    "))
	sb.WriteString(faintStyle.Render(xAxisLabels(len(vals))))

	return sb.String()
}

// xAxisLabels builds a compact x-axis label row. It places second markers at
// intervals of 4 (or fewer for short sequences) to keep the row readable.
// Returns a plain string; the caller applies styling.
func xAxisLabels(n int) string {
	if n == 0 {
		return ""
	}

	// Place markers at 0, 4, 8, ... seconds.
	step := 4
	if n <= 4 {
		step = 1
	}
	var parts []string
	lastEnd := 0
	for i := 0; i < n; i += step {
		label := fmt.Sprintf("%d", i)
		// Only append if there's room.
		if i+len(label) <= n {
			// Pad from lastEnd to i with spaces, then add label.
			for lastEnd < i {
				parts = append(parts, " ")
				lastEnd++
			}
			parts = append(parts, label)
			lastEnd += len(label)
		}
	}
	// Fill remaining.
	for lastEnd < n {
		parts = append(parts, " ")
		lastEnd++
	}

	return strings.Join(parts, "")
}

// minMax returns the minimum and maximum values in vals.
// Assumes len(vals) > 0.
func minMax(vals []float64) (min, max float64) {
	min, max = vals[0], vals[0]
	for _, v := range vals[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return min, max
}
