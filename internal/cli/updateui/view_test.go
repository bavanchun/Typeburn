package updateui

import (
	"regexp"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// modelAt builds a model parked at a given progress state, with no running
// program behind it.
func modelAt(noColor bool, p update.Progress) Model {
	th := theme.Load("default", noColor)
	m := New("v2.5.1", "v2.6.0", th, func() update.Progress { return p }, nil)
	m.cur = p
	return m
}

func frameLines(s string) []string {
	return strings.Split(strings.Trim(s, "\n"), "\n")
}

// The frame is a fixed-size box: every line must measure the same width, or the
// border visibly breaks.
func TestFrame_AllLinesEqualWidth(t *testing.T) {
	for _, noColor := range []bool{false, true} {
		m := modelAt(noColor, update.Progress{Stage: update.StageDownloading, Done: 2_400_000, Total: 4_513_792})
		lines := frameLines(m.Frame())

		want := lipgloss.Width(lines[0])
		for i, ln := range lines {
			if got := lipgloss.Width(ln); got != want {
				t.Errorf("noColor=%v line %d width = %d, want %d\n%q", noColor, i, got, want, ln)
			}
		}
	}
}

// The specific defect this guards: rebuilding the top border to inject a title
// is easy to get off by one, which shifts the top-right corner out of line.
func TestFrame_TitleInjectionPreservesBorderWidth(t *testing.T) {
	m := modelAt(false, update.Progress{Stage: update.StageVerifying})
	lines := frameLines(m.Frame())

	top, bottom := lines[0], lines[len(lines)-1]
	if lipgloss.Width(top) != lipgloss.Width(bottom) {
		t.Errorf("top border width %d != bottom border width %d", lipgloss.Width(top), lipgloss.Width(bottom))
	}
	if !strings.Contains(stripANSI(top), "typeburn update") {
		t.Errorf("top border lost its title: %q", stripANSI(top))
	}
}

// Colored and NO_COLOR frames must be layout-identical — only attributes differ.
func TestFrame_NoColorLayoutIdentical(t *testing.T) {
	p := update.Progress{Stage: update.StageDownloading, Done: 1_000_000, Total: 4_513_792}
	colored := frameLines(modelAt(false, p).Frame())
	plain := frameLines(modelAt(true, p).Frame())

	if len(colored) != len(plain) {
		t.Fatalf("line counts differ: colored %d, no-color %d", len(colored), len(plain))
	}
	for i := range colored {
		if a, b := lipgloss.Width(colored[i]), lipgloss.Width(plain[i]); a != b {
			t.Errorf("line %d width: colored %d, no-color %d", i, a, b)
		}
	}
}

// sgrColor matches the SGR sequences that set a foreground or background color.
// Attribute-only codes (bold, faint, reverse) are permitted under NO_COLOR.
var sgrColor = regexp.MustCompile(`\x1b\[[0-9;]*(?:3[0-79]|4[0-79]|9[0-7]|10[0-7])[0-9;]*m`)

// bubbles seeds a default purple gradient in progress.New, so this asserts the
// absence of color on the rendered frame rather than trusting any single knob.
func TestFrame_NoColorEmitsNoColorSGR(t *testing.T) {
	stages := []update.Progress{
		{Stage: update.StageChecksums},
		{Stage: update.StageDownloading, Done: 2_000_000, Total: 4_513_792},
		{Stage: update.StageVerifying},
		{Stage: update.StageInstalling},
	}
	for _, p := range stages {
		frame := modelAt(true, p).Frame()
		if loc := sgrColor.FindString(frame); loc != "" {
			t.Errorf("stage %v leaked color SGR %q", p.Stage, loc)
		}
	}
}

// Stages are declared in run order, so everything before the current stage
// renders settled and everything after renders pending.
func TestFrame_GlyphProgression(t *testing.T) {
	m := modelAt(true, update.Progress{Stage: update.StageVerifying})
	lines := frameLines(m.Frame())

	var rows []string
	for _, ln := range lines {
		plain := stripANSI(ln)
		for _, name := range []string{"checksums", "downloading", "verifying", "installing"} {
			if strings.Contains(plain, name) {
				rows = append(rows, plain)
			}
		}
	}
	if len(rows) != 4 {
		t.Fatalf("found %d stage rows, want 4:\n%s", len(rows), strings.Join(rows, "\n"))
	}
	if !strings.Contains(rows[0], "✓") || !strings.Contains(rows[1], "✓") {
		t.Errorf("finished stages not checked:\n%s\n%s", rows[0], rows[1])
	}
	if !strings.Contains(rows[3], "·") {
		t.Errorf("pending stage not dotted: %s", rows[3])
	}
}

// A response with no Content-Length has no honest percentage to show.
func TestFrame_UnknownTotalRendersIndeterminate(t *testing.T) {
	m := modelAt(true, update.Progress{Stage: update.StageDownloading, Done: 900, Total: 0})
	frame := stripANSI(m.Frame())

	if strings.Contains(frame, "%") {
		t.Errorf("rendered a percentage with unknown total:\n%s", frame)
	}
	if strings.Contains(frame, "NaN") || strings.Contains(frame, "Inf") {
		t.Errorf("rendered a degenerate number:\n%s", frame)
	}
	if !strings.Contains(frame, "· · ·") {
		t.Errorf("missing indeterminate marker:\n%s", frame)
	}
}

func TestPercent_ClampsAndHolds(t *testing.T) {
	tests := []struct {
		name string
		p    update.Progress
		want float64
	}{
		{"before download", update.Progress{Stage: update.StageChecksums}, 0},
		{"mid download", update.Progress{Stage: update.StageDownloading, Done: 50, Total: 200}, 0.25},
		{"unknown total", update.Progress{Stage: update.StageDownloading, Done: 50, Total: 0}, 0},
		{"after download", update.Progress{Stage: update.StageVerifying}, 1},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := modelAt(true, tc.p).percent(); got != tc.want {
				t.Errorf("percent() = %v, want %v", got, tc.want)
			}
		})
	}
}
