package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
)

// ladderRun builds a Result at a chosen speed and accuracy with a populated
// rail, which is the configuration that decides how much room the rail needs.
func ladderRun(netWPM, acc float64, termW, termH int) ResultModel {
	res := metrics.Result{
		NetWPM: netWPM, RawWPM: netWPM + 6, Accuracy: acc, Consistency: 95,
		CorrectChars: 220, IncorrectChars: 8, ExtraChars: 1, DurationMs: 30000,
		PerSecond: shortRunResult().Result.PerSecond,
	}
	msg := ResultMsg{Result: res, Mode: config.ModeTime, Length: 30}
	return NewResult(msg, theme.Load("default", true), config.DefaultKeymap()).
		WithContext(ResultContext{HasHistory: true, PB: 99, Avg10: 88, Rank: 2, Total: 6}).
		SetSize(termW, termH)
}

// Every rung of the hero's fallback ladder is reached by an ordinary run, not a
// contrived one. Two big-digit zones and a comparison rail do not fit together
// at any supported width once the WPM reaches three digits or the accuracy
// reaches 100%, so the ladder is the normal path and each step of it has to be
// exercised.
func TestResolveHeroZones_EveryRungIsReachable(t *testing.T) {
	cases := []struct {
		name           string
		wpm, acc       float64
		termW          int
		want           heroRung
		wantRail       bool
		wantBigAcc     bool
		wantShortLabel bool
	}{
		{"87 wpm at 96% on 120 cols", 87, 96, 120, heroRungBigAcc, true, true, false},
		{"87 wpm at 100% on 88 cols", 87, 100, 88, heroRungTightGutter, true, true, false},
		{"106 wpm at 97% on 80 cols", 106, 97, 80, heroRungTextAcc, true, false, false},
		{"106 wpm at 97% on 72 cols", 106, 97, 72, heroRungShortRail, true, false, true},
		{"106 wpm at 97% on 60 cols", 106, 97, 60, heroRungNoRail, false, false, false},
	}
	seen := map[heroRung]bool{}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lay := layoutFor(c.termW, 24)
			z := ladderRun(c.wpm, c.acc, c.termW, 24).heroZonesFor(lay.InnerW)
			seen[z.Rung] = true
			if z.Rung != c.want {
				t.Fatalf("rung %d, want %d (innerW=%d, zones=%+v)", z.Rung, c.want, lay.InnerW, z)
			}
			if (z.RailW > 0) != c.wantRail {
				t.Errorf("railW=%d, want rail=%v", z.RailW, c.wantRail)
			}
			if z.AccBig != c.wantBigAcc {
				t.Errorf("accBig=%v, want %v", z.AccBig, c.wantBigAcc)
			}
			if z.RailShort != c.wantShortLabel {
				t.Errorf("railShort=%v, want %v", z.RailShort, c.wantShortLabel)
			}
		})
	}
	for rung := heroRungBigAcc; rung <= heroRungNoRail; rung++ {
		if !seen[rung] {
			t.Errorf("rung %d is never reached by the table — it is unreachable code", rung)
		}
	}
}

// Zone widths are measured from the digits being rendered. A hardcoded width is
// what shipped a hero that broke the first time a run reached three digits.
func TestHeroZones_WidthsComeFromTheDigits(t *testing.T) {
	th := theme.Load("default", true)
	for _, wpm := range []float64{7, 42, 87, 100, 106, 200, 999} {
		m := ladderRun(wpm, 96, 120, 24)
		z := m.heroZonesFor(layoutFor(120, 24).InnerW)
		want := maxLineWidth(BigDigits(int(wpm), th))
		if z.WPMW != want {
			t.Errorf("wpm=%.0f: zone width %d, want the rendered digit width %d", wpm, z.WPMW, want)
		}
	}
}

// Whatever the run and whatever the terminal, the zones must exactly consume
// the inner width — never more, and never leave the rail hanging off the edge.
func TestHeroZones_NeverExceedInnerWidth(t *testing.T) {
	for termW := 60; termW <= 220; termW++ {
		lay := layoutFor(termW, 24)
		for _, wpm := range []float64{1, 9, 87, 100, 106, 200, 999} {
			for _, acc := range []float64{0, 87, 96, 100} {
				z := ladderRun(wpm, acc, termW, 24).heroZonesFor(lay.InnerW)
				used := z.WPMW + z.AccW + z.Gutter + z.RailW
				if z.RailW > 0 {
					used += z.Gutter
					if used != lay.InnerW {
						t.Fatalf("termW=%d wpm=%.0f acc=%.0f: zones use %d of InnerW %d (%+v)",
							termW, wpm, acc, used, lay.InnerW, z)
					}
					continue
				}
				if used > lay.InnerW {
					t.Fatalf("termW=%d wpm=%.0f acc=%.0f: zones use %d, InnerW is %d (%+v)",
						termW, wpm, acc, used, lay.InnerW, z)
				}
			}
		}
	}
}

// A value too wide to render as block art at all must still produce a band that
// fits. Block glyphs cannot be shrunk, so the only honest answer is to stop
// using them — silently overflowing would wrap the row and break the border the
// whole panel is drawn inside.
func TestHeroZones_AbsurdValueFallsBackToText(t *testing.T) {
	for _, wpm := range []float64{1e5, 1e6} {
		m := ladderRun(wpm, 96, 60, 24)
		lay := layoutFor(60, 24)
		if z := m.heroZonesFor(lay.InnerW); z.WPMBig {
			t.Fatalf("wpm=%.0f: block art claims to fit InnerW=%d (%+v)", wpm, lay.InnerW, z)
		}
		for i, line := range m.heroBand(lay) {
			if got := lipgloss.Width(line); got != lay.InnerW {
				t.Errorf("wpm=%.0f band line %d width %d, want %d", wpm, i, got, lay.InnerW)
			}
		}
		if !strings.Contains(stripANSI(strings.Join(m.heroBand(lay), "\n")), "wpm") {
			t.Errorf("wpm=%.0f: the value vanished instead of falling back to text", wpm)
		}
	}
}
