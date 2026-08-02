package ui

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

func TestCountUpValue_EndpointsAndMonotonic(t *testing.T) {
	start := int64(1000)
	final := 94

	if got := countUpValue(final, start, start); got != 0 {
		t.Fatalf("at start: got %d want 0", got)
	}
	if got := countUpValue(final, start, start+countUpMs); got != final {
		t.Fatalf("at end: got %d want %d", got, final)
	}

	prev := -1
	for now := start; now <= start+countUpMs; now += 50 {
		got := countUpValue(final, start, now)
		if got < prev {
			t.Fatalf("count-up regressed at %d: %d < %d", now, got, prev)
		}
		prev = got
	}
}

func TestSparkVisibleBars_Endpoints(t *testing.T) {
	start := int64(1000)
	if got := sparkVisibleBars(5, start, start); got != 0 {
		t.Fatalf("at start: got %d want 0", got)
	}
	if got := sparkVisibleBars(5, start, start+drawInMs); got != 5 {
		t.Fatalf("at end: got %d want 5", got)
	}
}

func TestCardProgress_Stagger(t *testing.T) {
	start := int64(1000)
	if got := cardProgress(1, start, start+staggerMs-1); got != 0 {
		t.Fatalf("card 1 before start: got %.2f want 0", got)
	}
	if got := cardProgress(1, start, start+staggerMs+cardFadeMs); got != 1 {
		t.Fatalf("card 1 after fade: got %.2f want 1", got)
	}
}

func TestBigDigitsFixed_ConstantWidth(t *testing.T) {
	th := theme.Default()
	final := 104
	wantW := maxLineWidth(BigDigits(final, th))
	for _, n := range []int{0, 9, 10, 99, 104} {
		for _, line := range strings.Split(BigDigitsFixed(n, final, th), "\n") {
			if got := lipgloss.Width(line); got != wantW {
				t.Fatalf("BigDigitsFixed(%d) line width=%d want %d", n, got, wantW)
			}
		}
	}
}

func TestResultReveal_StaticUsesOriginalBigDigits(t *testing.T) {
	m := newTestResult()
	final := int(math.Round(m.res.NetWPM))
	assertHeroBigDigitsPrefix(t, m.heroBand(layoutFor(m.w, m.h)), BigDigits(final, m.th))

	revealed := m.WithRevealStart(1000)
	revealed.nowMs = 1000 + resultRevealTotalMs()
	assertHeroBigDigitsPrefix(t, revealed.heroBand(layoutFor(m.w, m.h)), BigDigits(final, m.th))
}

func TestResultReveal_SettledMatchesStatic(t *testing.T) {
	static := newTestResult().View()
	revealed := newTestResult().WithRevealStart(1000)
	revealed.nowMs = 1000 + resultRevealTotalMs()

	if got := revealed.View(); got != static {
		t.Fatalf("settled reveal differs from static frame")
	}
}

func TestResultReveal_InProgressKeepsLayout(t *testing.T) {
	settled := newTestResult().View()
	revealed := newTestResult().WithRevealStart(1000)
	revealed.nowMs = 1000

	assertSameLineWidths(t, settled, revealed.View())
}

func TestResultReveal_NoColorKeepsLayout(t *testing.T) {
	th := theme.Load("default", true)
	static := newTestResult()
	static.th = th

	revealed := static.WithRevealStart(1000)
	revealed.nowMs = 1120

	assertSameLineWidths(t, static.View(), revealed.View())
}

func TestResultHasActiveAnim_Window(t *testing.T) {
	m := newTestResult().WithRevealStart(1000)
	if !m.HasActiveAnim(1000) {
		t.Fatal("reveal should be active at start")
	}
	if m.HasActiveAnim(1000 + resultRevealTotalMs()) {
		t.Fatal("reveal should be inactive at total duration")
	}
	if newTestResult().HasActiveAnim(1000) {
		t.Fatal("static result should not report active animation")
	}
}

func assertSameLineWidths(t *testing.T, a, b string) {
	t.Helper()
	aa := strings.Split(strip(a), "\n")
	bb := strings.Split(strip(b), "\n")
	if len(aa) != len(bb) {
		t.Fatalf("line count: got %d want %d", len(bb), len(aa))
	}
	for i := range aa {
		if len([]rune(aa[i])) != len([]rune(bb[i])) {
			t.Fatalf("line %d width: got %d want %d\nwant=%q\ngot =%q",
				i, len([]rune(bb[i])), len([]rune(aa[i])), aa[i], bb[i])
		}
	}
}

// revealConfigs are the Result states whose geometry differs: the digit counts
// that move the hero's zones, the rail states, the celebration, and the two
// chart shapes. The band positions the rail by column arithmetic against the
// hero, so every one of these has to hold its shape for the whole animation.
func revealConfigs() []struct {
	name  string
	model func(w, h int) ResultModel
} {
	ranked := ResultContext{HasHistory: true, PB: 111, Avg10: 96, Rank: 4, Total: 47}
	longRun := make([]metrics.PerSecond, 120)
	for i := range longRun {
		longRun[i] = metrics.PerSecond{Sec: i, RawWPM: float64(70 + i%17)}
		if i%9 == 0 {
			longRun[i].Errors = i%3 + 1
		}
	}
	return []struct {
		name  string
		model func(w, h int) ResultModel
	}{
		{"two digits, first run", func(w, h int) ResultModel {
			return ladderRun(87, 96, w, h).WithContext(ResultContext{})
		}},
		{"three digits at full accuracy, ranked", func(w, h int) ResultModel {
			return ladderRun(100, 100, w, h).WithContext(ranked)
		}},
		{"new best", func(w, h int) ResultModel {
			return ladderRun(120, 98, w, h).WithContext(ranked).WithBest(true)
		}},
		{"letter-strict", func(w, h int) ResultModel {
			m := ladderRun(74, 91, w, h).WithContext(ranked)
			m.strict = true
			m.res.KeystrokeAccuracy = 88
			return m
		}},
		{"no per-second samples", func(w, h int) ResultModel {
			m := ladderRun(64, 93, w, h).WithContext(ranked)
			m.res.PerSecond = nil
			return m
		}},
		{"withheld from history", func(w, h int) ResultModel {
			return ladderRun(106, 97, w, h).WithContext(ranked.Unranked())
		}},
		{"long run, downsampled chart", func(w, h int) ResultModel {
			m := ladderRun(93, 95, w, h).WithContext(ranked)
			m.res.PerSecond = longRun
			return m
		}},
	}
}

// Every frame of the reveal keeps the settled line count and per-line width, and
// the settled frame is byte-identical to the static one. Sweeping in 17 ms steps
// walks the whole window rather than the handful of instants a spot check hits —
// the reveal is where a column-positioned layout breaks first.
func TestResultReveal_ExhaustiveSweepHoldsGeometry(t *testing.T) {
	sizes := [][2]int{{60, 20}, {72, 24}, {80, 24}, {120, 24}, {200, 50}}
	const start = int64(1000)
	end := start + celebrateMs + 200

	for _, cfg := range revealConfigs() {
		for _, size := range sizes {
			w, h := size[0], size[1]
			settled := cfg.model(w, h)
			settled.revealStartMs, settled.nowMs = 0, 0
			want := strings.Split(stripANSI(settled.View()), "\n")

			for now := start; now <= end; now += 17 {
				m := cfg.model(w, h).WithRevealStart(start)
				m.nowMs = now
				got := strings.Split(stripANSI(m.View()), "\n")
				if len(got) != len(want) {
					t.Fatalf("%s @%dx%d t=%d: %d lines, settled has %d",
						cfg.name, w, h, now-start, len(got), len(want))
				}
				for i := range want {
					if len([]rune(got[i])) != len([]rune(want[i])) {
						t.Fatalf("%s @%dx%d t=%d line %d: width %d, settled %d\nwant=%q\ngot =%q",
							cfg.name, w, h, now-start, i, len([]rune(got[i])), len([]rune(want[i])),
							want[i], got[i])
					}
				}
			}

			// Past every window the animated frame is the static frame, bytes
			// and escapes included.
			done := cfg.model(w, h).WithRevealStart(start)
			done.nowMs = end
			if done.View() != settled.View() {
				t.Errorf("%s @%dx%d: settled reveal is not byte-identical to the static frame",
					cfg.name, w, h)
			}
		}
	}
}

// The counting number is right-aligned in its zone for the whole reveal, so the
// digits grow leftwards into space that is already reserved instead of shunting
// the block sideways as 9 becomes 10 becomes 100. Per-line width alone cannot
// see this: the zone is padded either way, and only the position of the ink
// moves.
func TestResultReveal_CountUpKeepsTheDigitsAnchored(t *testing.T) {
	const start = int64(1000)
	for _, wpm := range []float64{9, 94, 106} {
		m := ladderRun(wpm, 96, 120, 24)
		lay := layoutFor(120, 24)
		zoneW := m.heroZonesFor(lay.InnerW).WPMW
		settled := lastInkColumn(m.heroBand(lay)[1:], zoneW)

		for now := start; now <= start+countUpMs; now += 17 {
			r := ladderRun(wpm, 96, 120, 24).WithRevealStart(start)
			r.nowMs = now
			if got := lastInkColumn(r.heroBand(lay)[1:], zoneW); got != settled {
				t.Fatalf("wpm=%.0f t=%d: digits end at column %d, settled ends at %d",
					wpm, now-start, got, settled)
			}
		}
	}
}

// The band's columns are resolved from the final values, so the accuracy zone
// and the rail sit in the same place on the first frame of the reveal as on the
// last. Per-line width cannot see this — every row is padded to the panel's
// inner width either way — but the user would watch two columns slide sideways
// while the number counted up.
func TestResultReveal_ZonesDoNotMoveDuringTheCountUp(t *testing.T) {
	const start = int64(1000)
	for _, wpm := range []float64{9, 94, 106} {
		for _, w := range []int{60, 80, 120} {
			lay := layoutFor(w, 24)
			want := ladderRun(wpm, 96, w, 24).heroZonesFor(lay.InnerW)
			for now := start; now <= start+resultRevealTotalMs(); now += 17 {
				m := ladderRun(wpm, 96, w, 24).WithRevealStart(start)
				m.nowMs = now
				if got := m.heroZonesFor(lay.InnerW); got != want {
					t.Fatalf("wpm=%.0f w=%d t=%d: zones moved\n got %+v\nwant %+v",
						wpm, w, now-start, got, want)
				}
			}
		}
	}
}

// lastInkColumn returns the rightmost non-blank column within the band's first
// zoneW columns.
func lastInkColumn(band []string, zoneW int) int {
	last := -1
	for _, line := range band {
		r := []rune(stripANSI(line))
		if len(r) > zoneW {
			r = r[:zoneW]
		}
		for i, c := range r {
			if c != ' ' && i > last {
				last = i
			}
		}
	}
	return last
}

// The band's first line carries the zone labels; the block art starts under it.
func assertHeroBigDigitsPrefix(t *testing.T, band []string, big string) {
	t.Helper()
	bigLines := strings.Split(big, "\n")
	if len(band) != len(bigLines)+1 {
		t.Fatalf("hero band line count=%d want %d", len(band), len(bigLines)+1)
	}
	for i, want := range bigLines {
		if !strings.HasPrefix(band[i+1], want) {
			t.Fatalf("band line %d should start with the original big digit row\nwant prefix=%q\ngot=%q",
				i+1, want, band[i+1])
		}
	}
}
