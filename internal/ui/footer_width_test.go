package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// footerHintSets is every hint set the app renders, so the sweep below covers
// the six-hint Home bar rather than only the short typing one.
func footerHintSets() map[string][]Hint {
	return map[string][]Hint{
		"typing":   TypingHints(),
		"history":  historyFooterHints(),
		"settings": settingsHints(),
		"home":     homeHints(),
		"result":   resultHints(),
	}
}

// TestRenderFooter_FitsEveryWidth sweeps the whole supported range one column
// at a time. The defect this replaces was invisible at every round number: the
// full six-hint Home footer is 73 cells and the tier switched to it at 72, so
// exactly one width in the range overflowed.
func TestRenderFooter_FitsEveryWidth(t *testing.T) {
	th := theme.Load("default", false)

	for name, hints := range footerHintSets() {
		for w := 60; w <= 90; w++ {
			if got := lipgloss.Width(RenderFooter(hints, w, th)); got > w {
				t.Errorf("%s footer at width %d renders %d cells", name, w, got)
			}
		}
	}
}

// TestRenderFooter_UsesTheLabelsWhenTheyFit guards against the fix being a
// blanket downgrade: the full form is the design above the narrow tier and must
// survive wherever it fits.
func TestRenderFooter_UsesTheLabelsWhenTheyFit(t *testing.T) {
	th := theme.Load("default", false)
	hints := footerHintSets()["home"]

	full := RenderFooter(hints, 200, th)
	if !strings.Contains(full, "settings") {
		t.Fatalf("a 200-column footer dropped its labels: %q", full)
	}

	// One cell wider than the measured full form is the first width that can
	// carry it; one cell narrower must not.
	fits := lipgloss.Width(full)
	if got := RenderFooter(hints, fits, th); !strings.Contains(got, "settings") {
		t.Errorf("labels dropped at width %d, the exact width they occupy", fits)
	}
	if got := RenderFooter(hints, fits-1, th); strings.Contains(got, "settings") {
		t.Errorf("labels kept at width %d, one cell too narrow for them", fits-1)
	}
}

// TestRenderFooter_NarrowTierStaysGlyphs pins the design band: below 72 columns
// the glyph form is deliberate, not a consequence of measurement.
func TestRenderFooter_NarrowTierStaysGlyphs(t *testing.T) {
	th := theme.Load("default", false)

	got := RenderFooter(TypingHints(), narrowFooterW-1, th)

	if strings.Contains(got, "restart") {
		t.Errorf("footer at width %d shows action words: %q", narrowFooterW-1, got)
	}
	if !strings.Contains(got, "tab") {
		t.Errorf("glyph footer lost its keys: %q", got)
	}
}

// TestRenderFooter_UnknownWidthRendersFull: width 0 means the terminal size has
// not arrived, and a footer with no labels is the worse guess.
func TestRenderFooter_UnknownWidthRendersFull(t *testing.T) {
	th := theme.Load("default", false)

	if got := RenderFooter(TypingHints(), 0, th); !strings.Contains(got, "restart") {
		t.Errorf("unsized footer dropped its labels: %q", got)
	}
}
