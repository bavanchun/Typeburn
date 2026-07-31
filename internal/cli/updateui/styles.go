package updateui

import (
	"image/color"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// barWidth is the progress bar's cell width. Chosen so the download row fits
// inside boxWidth with its glyph, label and percentage — see view.go.
const barWidth = 20

// spring tuning for the bar. The bar chases the reported byte count rather than
// snapping to it, so the jittery throughput of a real download reads as smooth
// motion instead of stutter.
const (
	springFrequency = 12.0
	springDamping   = 1.0
)

// isNoColor reports whether the active theme resolves to no color at all. This
// is the same predicate the TUI uses to pick its NO_COLOR variants, so both
// surfaces agree on when color is unavailable.
func isNoColor(th theme.Theme) bool { return th.Color(theme.RoleBg) == nil }

// newBar builds the progress bar for th.
//
// Two NO_COLOR hazards, both specific to bubbles v2 and both verified against
// v2.1.1 rather than assumed:
//
//  1. progress.New seeds a default purple blend, so simply not passing
//     WithColors still emits color. It has to be overridden with a color func
//     that yields nothing.
//  2. The fill, the empty track and the percentage text are three independent
//     knobs. Clearing only the fill leaves the other two colored.
func newBar(th theme.Theme) progress.Model {
	opts := []progress.Option{
		progress.WithWidth(barWidth),
		progress.WithFillCharacters('█', '░'),
		progress.WithSpringOptions(springFrequency, springDamping),
	}

	if isNoColor(th) {
		opts = append(opts, progress.WithColorFunc(func(_, _ float64) color.Color { return nil }))
		bar := progress.New(opts...)
		bar.EmptyColor = nil
		bar.PercentageStyle = lipgloss.NewStyle()
		return bar
	}

	opts = append(opts, progress.WithColors(
		th.Color(theme.RoleAccentDim),
		th.Color(theme.RoleAccent),
		th.Color(theme.RoleSuccess),
	))
	bar := progress.New(opts...)
	bar.EmptyColor = th.Color(theme.RoleTextFaint)
	bar.PercentageStyle = th.Style(theme.RoleTextMuted)
	return bar
}

// newSpinner builds the activity spinner. It is deliberately left unstyled:
// view.go colors the glyph by row state, so a settled row can swap in the
// success check without a competing style.
func newSpinner() spinner.Model {
	return spinner.New(spinner.WithSpinner(spinner.MiniDot))
}
