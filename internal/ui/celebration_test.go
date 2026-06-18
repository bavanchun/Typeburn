package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
)

func TestCelebrationGlyphs_Width1(t *testing.T) {
	for _, g := range celebrationGlyphs {
		if w := lipgloss.Width(string(g)); w != 1 {
			t.Errorf("glyph %q display width=%d want 1", string(g), w)
		}
	}
}

// celebrationCells must be empty outside the active window and non-empty inside.
func TestCelebrationCells_Lifetime(t *testing.T) {
	th := theme.Default()
	band := []int{2, 3, 4}
	start := int64(1000)

	// Mid-window (after staggered births, before deaths) particles are live.
	if got := celebrationCells(start, start+150, band, 40, th); len(got) == 0 {
		t.Error("expected live particles mid-window")
	}
	// Past the longest particle life (stagger ≤200 + life 300 = 500ms), none remain.
	if got := celebrationCells(start, start+600, band, 40, th); len(got) != 0 {
		t.Errorf("expected no particles after all lifetimes, got %d", len(got))
	}
}

// Deterministic jitter: identical (i, startMs) → identical placement, so goldens
// are stable.
func TestCelebrationCells_Deterministic(t *testing.T) {
	th := theme.Default()
	band := []int{2, 3, 4}
	a := celebrationCells(1000, 1050, band, 40, th)
	b := celebrationCells(1000, 1050, band, 40, th)
	if len(a) != len(b) {
		t.Fatalf("nondeterministic count: %d vs %d", len(a), len(b))
	}
	for i := range a {
		if a[i].row != b[i].row || a[i].col != b[i].col {
			t.Errorf("particle %d moved between calls: %v vs %v", i, a[i], b[i])
		}
	}
}

// blankBand selects only blank rows adjacent to content, never far-flung edges.
func TestBlankBand_AdjacentToContent(t *testing.T) {
	lines := []string{
		"          ", // 0 blank, far from content (edge)
		"          ", // 1 blank
		"          ", // 2 blank
		"          ", // 3 blank (within 3 of content at 4? 4 is content) → included
		"panel here", // 4 content
		"          ", // 5 blank, adjacent → included
		"          ", // 6 blank, within 3 → included
	}
	band := blankBand(lines)
	got := map[int]bool{}
	for _, i := range band {
		got[i] = true
	}
	if got[0] {
		t.Error("row 0 (far edge) should not be in band")
	}
	if !got[3] || !got[5] {
		t.Error("rows adjacent to content should be in band")
	}
}

// applyCelebration must preserve line count and per-line rune width, and only
// touch blank rows.
func TestApplyCelebration_PreservesLayout(t *testing.T) {
	th := theme.Default()
	content := strings.Join([]string{
		strings.Repeat(" ", 40),
		"  +----------------------------------+  ",
		strings.Repeat(" ", 40),
		strings.Repeat(" ", 40),
	}, "\n")

	out := applyCelebration(content, 1000, 1050, th)
	in := strings.Split(content, "\n")
	got := strings.Split(out, "\n")
	if len(in) != len(got) {
		t.Fatalf("line count changed: %d → %d", len(in), len(got))
	}
	for i := range in {
		iw := len([]rune(stripANSI(in[i])))
		gw := len([]rune(stripANSI(got[i])))
		if iw != gw {
			t.Errorf("line %d width changed: %d → %d", i, iw, gw)
		}
	}
	// The content (panel) row must be untouched.
	if got[1] != in[1] {
		t.Error("celebration touched a content row")
	}
}

// Outside the active window applyCelebration is a no-op (settled == input).
func TestApplyCelebration_InactiveNoop(t *testing.T) {
	content := strings.Repeat(" ", 20) + "\nx\n" + strings.Repeat(" ", 20)
	if got := applyCelebration(content, 0, 50, theme.Default()); got != content {
		t.Error("startMs<=0 should be a no-op")
	}
	if got := applyCelebration(content, 1000, 1000+celebrateMs, theme.Default()); got != content {
		t.Error("after celebrateMs should be a no-op")
	}
}

// An isBest result keeps the loop alive through the celebration window; an
// ordinary result self-stops with the reveal.
func TestResultCelebration_ExtendsActiveWindow(t *testing.T) {
	best := newTestResult().WithBest(true).WithRevealStart(1000)
	afterReveal := int64(1000) + resultRevealTotalMs()
	if !best.HasActiveAnim(afterReveal) {
		t.Error("new-best result should stay active through the celebration window")
	}
	if best.HasActiveAnim(1000 + celebrateMs) {
		t.Error("celebration should self-stop after celebrateMs")
	}

	ordinary := newTestResult().WithBest(false).WithRevealStart(1000)
	if ordinary.HasActiveAnim(afterReveal) {
		t.Error("ordinary result should self-stop when the reveal finishes")
	}
}

// An ordinary (non-best) result never sparkles: well past every window it
// renders exactly the static frame.
func TestResultView_OrdinaryNoCelebration(t *testing.T) {
	static := newTestResult().View() // best=false, no reveal
	ordinary := newTestResult().WithBest(false).WithRevealStart(1000)
	ordinary.nowMs = 1000 + celebrateMs + 100 // past reveal + any burst
	if ordinary.View() != static {
		t.Error("ordinary result must render the static frame (zero sparkle)")
	}
}

// A new-best frame mid-burst differs from its settled self (sparkle present)
// but keeps identical line count + width; after celebrateMs the burst is gone.
func TestResultView_BestBurstThenSettles(t *testing.T) {
	mid := newTestResult().WithBest(true).WithRevealStart(1000)
	mid.nowMs = 1150 // particles reliably alive
	settled := newTestResult().WithBest(true).WithRevealStart(1000)
	settled.nowMs = 1000 + celebrateMs + 100

	if mid.View() == settled.View() {
		t.Error("mid-celebration frame should differ from settled (sparkle present)")
	}
	assertSameLineWidths(t, settled.View(), mid.View())
}
