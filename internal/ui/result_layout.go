package ui

// resultMaxContentW caps the Result panel's inner content width. Past roughly
// this width a fixed two-column layout stops reading as one object: the eye has
// to travel too far between related values, and the panel becomes a mostly
// empty box.
//
// Measured reason the cap stays: stretching the inner area to 116 columns (a
// 200-column terminal) leaves the comparison rail's label and its value about
// 50 columns apart, at which point the pair stops reading as one fact. The rail
// therefore also renders at its own natural width inside whatever column it is
// given, and railLabelValueSpan is asserted in the tests. Beyond the cap the
// panel is centred in the remaining space rather than stretched to fill it.
const resultMaxContentW = 88

// resultPanelChrome is the outer margin the panel leaves around itself.
const resultPanelChrome = 8

// resultPanelInset is what the panel itself consumes inside PanelW: two border
// columns plus two padding columns on each side. The previous arithmetic used
// 4 here, which under-counted the borders; nothing noticed because no section
// ever tried to use its full width.
const resultPanelInset = 6

// resultMinPanelW is the narrowest panel worth drawing. Below the degraded gate
// (60×20) the Result screen is replaced entirely, so this floor only guards
// against absurd inputs.
const resultMinPanelW = 40

// resultCompactH is the terminal height below which the panel drops its
// vertical padding and shortens the chart. The Result frame has to leave room
// for a blank spacer row, an optional update hint, the footer, and a
// persistence notice overlaid on the terminal's last row — so the panel gets
// h-4 rows, not h.
const resultCompactH = 24

// Chart plot rows per height tier, and the panel's vertical padding per tier.
const (
	resultChartRows        = 4
	resultChartRowsCompact = 3
)

// heroGutterWide and heroGutterTight are the two column gaps between the hero
// band's zones. The tight value is the first rung of the fallback ladder.
const (
	heroGutterWide  = 6
	heroGutterTight = 4
)

// resultLayout is the single source of truth for Result-screen geometry. The
// panel, hero, graph, and meta rows all size themselves from one of these, so
// they cannot disagree about how much room they have — four independent width
// policies is what left a 192-column panel holding 35 columns of content.
type resultLayout struct {
	PanelW  int  // bordered panel width, including border columns
	InnerW  int  // content width inside the border and padding
	ChartH  int  // chart plot rows
	VPad    int  // panel vertical padding rows, top and bottom
	Compact bool // short terminal: no vertical padding, shorter chart, one meta row
}

// ContentRows returns the number of rows the panel's inner content occupies.
// It is fixed per tier by construction: label row, six hero rows, the chart
// header, the plot, its baseline and second ticks, then the meta rows.
func (l resultLayout) ContentRows() int {
	metaRows := 2
	if l.Compact {
		metaRows = 1
	}
	return 1 + numRows + 1 + l.ChartH + 2 + metaRows
}

// PanelRows returns the rendered height of the whole bordered panel.
func (l resultLayout) PanelRows() int { return l.ContentRows() + 2*l.VPad + 2 }

// Horizontal centring is deliberately absent. ResultModel.View already places
// the whole frame with lipgloss.Place(Center, Center); a layout that also padded
// would centre the panel twice and push it right. Capping the width is the
// entire job here — the panel looked left-hugging before because it was nearly
// as wide as the terminal with its content crammed inside, not because it was
// misplaced.

// layoutFor resolves the geometry for a terminal size. A non-positive height
// means "unconstrained" (direct constructions and recorded baselines), which
// takes the roomier tier.
func layoutFor(termW, termH int) resultLayout {
	panelW := termW - resultPanelChrome
	if panelW < resultMinPanelW {
		panelW = resultMinPanelW
	}

	innerW := panelW - resultPanelInset
	if innerW > resultMaxContentW {
		innerW = resultMaxContentW
		panelW = innerW + resultPanelInset
	}
	if innerW < 1 {
		innerW = 1
	}

	lay := resultLayout{PanelW: panelW, InnerW: innerW, ChartH: resultChartRows, VPad: 1}
	if termH > 0 && termH < resultCompactH {
		lay.Compact = true
		lay.ChartH = resultChartRowsCompact
		lay.VPad = 0
	}
	return lay
}

// heroRung identifies which arrangement of the hero band survived the width
// budget. Gate measurements on the shipped glyph table: a three-digit WPM is 24
// cells and a big "100 %" is 26, so 24 + 26 plus two gutters already spends 62
// of the 88 columns the cap allows. Two big-digit zones therefore cannot
// coexist with a useful comparison rail at any supported width — the ladder is
// the ordinary path, not an edge case.
type heroRung int

const (
	heroRungBigAcc      heroRung = iota // big accuracy, wide gutters, full-label rail
	heroRungTightGutter                 // big accuracy, tight gutters, full-label rail
	heroRungTextAcc                     // accuracy demoted to a text block
	heroRungShortRail                   // rail switches to abbreviated labels
	heroRungNoRail                      // no room beside the hero; rail is dropped
)

// heroDemand is the width every candidate arrangement is measured against. All
// five numbers are measured from the strings actually about to be rendered —
// hardcoding a digit width is what shipped a hero that broke the moment a run
// reached 100 wpm.
type heroDemand struct {
	WPMW       int // big-digit WPM block
	WPMTextW   int // text fallback for an absurdly wide value
	AccBigW    int // big-digit accuracy block including its trailing " %"
	AccTextW   int // accuracy as a two-line label/value block
	RailFullW  int // rail at its natural width with full labels
	RailShortW int // rail at its natural width with abbreviated labels
}

// heroZones is the resolved hero band geometry for one render.
type heroZones struct {
	Rung      heroRung
	WPMW      int
	AccW      int
	Gutter    int
	RailW     int  // columns reserved for the rail; 0 when it is dropped
	WPMBig    bool // false only when even the big digits alone overflow innerW
	AccBig    bool
	RailShort bool
}

// resolveHeroZones walks the fallback ladder and returns the first arrangement
// whose leftover columns can hold the rail at its natural width.
func resolveHeroZones(innerW int, d heroDemand) heroZones {
	ladder := []struct {
		zones heroZones
		need  int
	}{
		{heroZones{Rung: heroRungBigAcc, AccW: d.AccBigW, Gutter: heroGutterWide, AccBig: true}, d.RailFullW},
		{heroZones{Rung: heroRungTightGutter, AccW: d.AccBigW, Gutter: heroGutterTight, AccBig: true}, d.RailFullW},
		{heroZones{Rung: heroRungTextAcc, AccW: d.AccTextW, Gutter: heroGutterWide}, d.RailFullW},
		{heroZones{Rung: heroRungShortRail, AccW: d.AccTextW, Gutter: heroGutterWide, RailShort: true}, d.RailShortW},
	}
	for _, step := range ladder {
		z := step.zones
		z.WPMW, z.WPMBig = d.WPMW, true
		z.RailW = innerW - z.WPMW - z.AccW - 2*z.Gutter
		if step.need > 0 && z.RailW >= step.need {
			return z
		}
	}
	return heroWithoutRail(innerW, d)
}

// heroWithoutRail is the bottom of the ladder: the hero keeps the whole inner
// width. The gutter shrinks and, for a value too wide to render as block art at
// all, the WPM falls back to text so the band can never wrap the border.
func heroWithoutRail(innerW int, d heroDemand) heroZones {
	z := heroZones{Rung: heroRungNoRail, WPMW: d.WPMW, AccW: d.AccTextW, Gutter: heroGutterWide, WPMBig: true}
	for _, g := range []int{heroGutterWide, heroGutterTight, 1} {
		z.Gutter = g
		if z.WPMW+z.AccW+z.Gutter <= innerW {
			return z
		}
	}
	z.Gutter, z.AccW = 1, 0
	if z.WPMW <= innerW {
		return z
	}
	z.WPMBig, z.WPMW = false, d.WPMTextW
	return z
}
