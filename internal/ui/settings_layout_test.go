package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// TestSettingsRows_EndWhereTheRuleEnds is the assertion the byte-length gap
// could never satisfy. Every value is wrapped in "‹ ›", whose guillemets cost
// three bytes and one cell each, so len overcounted the value by four and every
// row stopped two cells short of the rule above it.
func TestSettingsRows_EndWhereTheRuleEnds(t *testing.T) {
	th := theme.Load("default", false)
	m := NewSettings(config.Defaults(), th, config.DefaultKeymap()).SetSize(80, 24)

	for i, row := range m.rows {
		for _, sel := range []int{-1, i} {
			m.sel = sel
			got := lipgloss.Width(stripANSI(m.renderRow(i, row)))
			if got != settingsBlockW {
				t.Errorf("row %q (selected=%v) is %d cells, the rule is %d",
					row.label, sel == i, got, settingsBlockW)
			}
		}
	}
}

// TestSettingsRows_MeasureValuesInCellsNotBytes pins the unit directly, so a
// value carrying multi-byte runes cannot silently reintroduce the drift.
func TestSettingsRows_MeasureValuesInCellsNotBytes(t *testing.T) {
	const label = "Default mode"
	ascii := settingsGap(label, "‹ words ›")
	if want := settingsBlockW - settingsRowIndentW - len(label) - 9; ascii != want {
		t.Errorf("gap %d for a 9-cell value, want %d", ascii, want)
	}
	// "中文字" is 6 bytes over the ASCII value's 5 and 6 cells over its 5, so a
	// byte-based gap and a cell-based one disagree by three here.
	const wideVal = "‹ 中文字 ›"
	want := settingsBlockW - settingsRowIndentW - len(label) - lipgloss.Width(wideVal)
	if got := settingsGap(label, wideVal); got != want {
		t.Errorf("gap %d for a %d-cell value, want %d", got, lipgloss.Width(wideVal), want)
	}
}

// TestSettingsRows_NeverCollapseTheGap: a long enough label and value must
// still leave the two cells that keep them from touching.
func TestSettingsRows_NeverCollapseTheGap(t *testing.T) {
	if got := settingsGap(strings.Repeat("x", 40), "‹ something long ›"); got != 2 {
		t.Errorf("gap %d, want the 2-cell minimum", got)
	}
}

// TestSettingsView_FitsEverySupportedWidth sweeps the range: the block is fixed
// at settingsBlockW, but the footer beneath it is not.
func TestSettingsView_FitsEverySupportedWidth(t *testing.T) {
	th := theme.Load("default", false)
	km := config.DefaultKeymap()

	for w := 60; w <= 90; w++ {
		view := NewSettings(config.Defaults(), th, km).SetSize(w, 24).View()
		for i, ln := range strings.Split(stripANSI(view), "\n") {
			if got := lipgloss.Width(ln); got > w {
				t.Errorf("width %d: line %d is %d cells", w, i, got)
			}
		}
	}
}
