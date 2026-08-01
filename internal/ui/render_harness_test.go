package ui

import (
	"os"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// screenCase is one renderable screen plus the terminal-independent state it
// needs. render must build the sub-model from scratch at every size: several
// sub-models compute layout in SetSize, so reusing one across sizes would
// measure a stale arrangement rather than the one a user sees.
type screenCase struct {
	name   string
	render func(th theme.Theme, w, h int) string
}

// renderFrameTheme returns the raw frame a screen produces at w×h in the given
// theme, escapes intact. Every other accessor is built on it, so there is one
// place where a case is turned into output and one place a fidelity gate has to
// cover.
func renderFrameTheme(s screenCase, th theme.Theme, w, h int) string {
	return s.render(th, w, h)
}

// renderFrame is renderFrameTheme in the default theme — the raw bytes the
// recorded baselines hold.
func renderFrame(s screenCase, w, h int) string {
	return renderFrameTheme(s, theme.Load("default", false), w, h)
}

// renderScreenTheme returns the frame with ANSI stripped and split into lines.
//
// Callers measure width with lipgloss.Width, not len: stripping ANSI leaves
// multi-cell runes intact, and a CJK screen that counted runes would look like
// it fits when it does not.
func renderScreenTheme(t *testing.T, s screenCase, th theme.Theme, w, h int) []string {
	t.Helper()
	return strings.Split(stripANSI(renderFrameTheme(s, th, w, h)), "\n")
}

// renderScreen is renderScreenTheme in the default theme. A new screen is
// covered by every invariant the moment it is added to screenCases.
func renderScreen(t *testing.T, s screenCase, w, h int) []string {
	t.Helper()
	return renderScreenTheme(t, s, theme.Load("default", false), w, h)
}

// TestRenderHarness_ReproducesRecordedBaseline gates every other assertion built
// on the harness. If renderScreen rendered something subtly different from the
// real program, every invariant below would be measured against a fiction.
//
// The comparison is against the bytes on disk, escapes and all. Two weaker
// versions were tried and both are worthless: comparing the harness to a live
// re-render of the same expression is f(x) == f(x), and comparing ANSI-stripped
// text passes even when the harness loads the wrong theme, because stripping is
// exactly what erases the difference a theme makes.
func TestRenderHarness_ReproducesRecordedBaseline(t *testing.T) {
	const w = 80
	recorded, err := os.ReadFile(baselinePath(w, false))
	if err != nil {
		t.Fatalf("read recorded panel: %v", err)
	}

	panelCase := screenCase{"result/panel", func(th theme.Theme, w, h int) string {
		m := NewResult(shortRunResult(), th, config.DefaultKeymap()).SetSize(w, h)
		m.revealStartMs, m.nowMs = 0, 1<<40
		return m.renderPanel()
	}}

	// Byte-exact, unstripped: this is the assertion that catches a harness
	// rendering in a theme the recording was not made in.
	if got := renderFrame(panelCase, w, 40); got != string(recorded) {
		t.Fatalf("harness frame differs from the recorded bytes\n got %q\nwant %q", got, string(recorded))
	}

	// And the stripped/split view the invariants actually consume must line up
	// with the same recording.
	want := strings.Split(stripANSI(string(recorded)), "\n")
	panel := renderScreen(t, panelCase, w, 40)
	if len(panel) != len(want) {
		t.Fatalf("harness produced %d lines, recorded panel has %d", len(panel), len(want))
	}
	for i := range want {
		if panel[i] != want[i] {
			t.Fatalf("line %d differs\n harness: %q\nrecorded: %q", i, panel[i], want[i])
		}
	}

	// The recording is of renderPanel, but every entry in screenCases goes
	// through View. Assert the recorded panel actually appears in the View
	// frame, or the gate would vouch for a path no case uses.
	frame := renderScreen(t, screenCase{"result/view", func(th theme.Theme, w, h int) string {
		m := NewResult(shortRunResult(), th, config.DefaultKeymap()).SetSize(w, h)
		m.revealStartMs, m.nowMs = 0, 1<<40
		return m.View()
	}}, w, 40)

	for _, line := range want {
		if !containsLine(frame, line) {
			t.Fatalf("recorded panel line %q does not appear in the View frame", line)
		}
	}
}

// containsLine reports whether any frame line contains want once surrounding
// placement padding is discounted.
func containsLine(frame []string, want string) bool {
	want = strings.TrimSpace(want)
	if want == "" {
		return true
	}
	for _, ln := range frame {
		if strings.Contains(ln, want) {
			return true
		}
	}
	return false
}
