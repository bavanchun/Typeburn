package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/theme"
	"github.com/bavanchun/Typeburn/internal/words"
)

// resultWithMisses builds an 80×24 ResultModel carrying a known heatmap, using
// the given theme so NO_COLOR/mono invariants can be asserted.
func resultWithMisses(th theme.Theme, misses []metrics.KeyMiss) ResultModel {
	res := metrics.Result{
		NetWPM:       80,
		Accuracy:     95,
		DurationMs:   30000,
		CorrectChars: 100,
		KeyMisses:    misses,
	}
	msg := ResultMsg{Result: res, Mode: config.ModeTime, Length: 30, QuoteLen: words.QuoteShort}
	return NewResult(msg, th, config.DefaultKeymap()).SetSize(80, 24)
}

var sampleMisses = []metrics.KeyMiss{
	{Key: 'e', Label: "e", Misses: 4, Attempts: 31},
	{Key: 't', Label: "t", Misses: 3, Attempts: 20},
	{Key: 'a', Label: "a", Misses: 2, Attempts: 18},
	{Key: ' ', Label: "␣", Misses: 2, Attempts: 15},
}

// TestResultView_ShowsMostMissed checks the heatmap line renders the label and
// the top key with a ×count.
func TestResultView_ShowsMostMissed(t *testing.T) {
	view := stripANSI(resultWithMisses(theme.Default(), sampleMisses).View())
	if !strings.Contains(view, "most missed") {
		t.Errorf("expected 'most missed' label in view:\n%s", view)
	}
	if !strings.Contains(view, "e ×4") {
		t.Errorf("expected top key 'e ×4' in view:\n%s", view)
	}
	if !strings.Contains(view, "␣") {
		t.Errorf("expected space glyph for missed space in view:\n%s", view)
	}
}

// TestResultView_CleanRunShowsNoMissedKeys checks the faint fallback for clean runs.
func TestResultView_CleanRunShowsNoMissedKeys(t *testing.T) {
	view := stripANSI(resultWithMisses(theme.Default(), nil).View())
	if !strings.Contains(view, "no missed keys") {
		t.Errorf("expected 'no missed keys' on clean run:\n%s", view)
	}
}

// TestRenderKeyHeatmap_MonoLayoutIdentical asserts the heatmap text content and
// width are identical between the colored and NO_COLOR renders (attribute-only).
func TestRenderKeyHeatmap_MonoLayoutIdentical(t *testing.T) {
	colored := resultWithMisses(theme.Default(), sampleMisses).renderKeyHeatmap(60)
	noColor := resultWithMisses(theme.Load("default", true), sampleMisses).renderKeyHeatmap(60)
	if stripANSI(colored) != stripANSI(noColor) {
		t.Errorf("layout differs between colored and NO_COLOR:\n colored=%q\n nocolor=%q",
			stripANSI(colored), stripANSI(noColor))
	}
}

// TestRenderKeyHeatmap_WidthCap ensures the visible width never exceeds innerW
// even at a narrow terminal.
func TestRenderKeyHeatmap_WidthCap(t *testing.T) {
	m := resultWithMisses(theme.Default(), sampleMisses)
	for _, innerW := range []int{20, 30, 52} {
		line := m.renderKeyHeatmap(innerW)
		if w := len([]rune(stripANSI(line))); w > innerW {
			t.Errorf("innerW=%d: visible width %d exceeds cap:\n%q", innerW, w, stripANSI(line))
		}
	}
}
