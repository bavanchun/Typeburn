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

// NOTE: minMax and xAxisLabels now live in result_graph.go (moved there when the
// dual-axis graph became their primary consumer). sparklineVisible keeps calling
// them unchanged — same package. This whole file is removed in Phase 3 once the
// graph replaces the bar sparkline on the Result screen.
