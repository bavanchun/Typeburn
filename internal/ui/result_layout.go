package ui

// resultMaxContentW caps the Result panel's inner content width. Past roughly
// this width a fixed two-column layout stops reading as one object: the eye has
// to travel too far between related values, and the panel becomes a mostly
// empty box. Beyond the cap the panel is centred in the remaining space rather
// than stretched to fill it.
//
// Provisional: validated against real renders at 60/80/120/200 columns before
// the cap is switched on.
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

// resultLayout is the single source of truth for Result-screen geometry. The
// panel, hero, graph, and stats grid all size themselves from one of these, so
// they cannot disagree about how much room they have — four independent width
// policies is what left a 192-column panel holding 35 columns of content.
type resultLayout struct {
	PanelW  int // bordered panel width, including border columns
	InnerW  int // content width inside the border and padding
	LeftPad int // columns of margin that centre the panel
}

// layoutFor resolves the geometry for a terminal width.
func layoutFor(termW int) resultLayout {
	panelW := termW - resultPanelChrome
	if panelW < resultMinPanelW {
		panelW = resultMinPanelW
	}

	innerW := panelW - resultPanelInset
	if innerW < 1 {
		innerW = 1
	}
	return resultLayout{PanelW: panelW, InnerW: innerW}
}
