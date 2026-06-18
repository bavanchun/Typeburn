package ui

import (
	"math"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/internal/theme"
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
	assertHeroBigDigitsPrefix(t, m.renderHero(80), BigDigits(final, m.th))

	revealed := m.WithRevealStart(1000)
	revealed.nowMs = 1000 + resultRevealTotalMs()
	assertHeroBigDigitsPrefix(t, revealed.renderHero(80), BigDigits(final, m.th))
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

func assertHeroBigDigitsPrefix(t *testing.T, hero, big string) {
	t.Helper()
	heroLines := strings.Split(hero, "\n")
	bigLines := strings.Split(big, "\n")
	if len(heroLines) < len(bigLines) {
		t.Fatalf("hero line count=%d want at least %d", len(heroLines), len(bigLines))
	}
	for i, want := range bigLines {
		if !strings.HasPrefix(heroLines[i], want) {
			t.Fatalf("hero line %d should start with original big digit row\nwant prefix=%q\ngot=%q",
				i, want, heroLines[i])
		}
	}
}
