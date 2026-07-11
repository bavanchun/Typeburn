package metrics

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

func TestLiveWPM(t *testing.T) {
	fwd := func(r rune) typing.Keystroke { return typing.Keystroke{Typed: r} }
	bsp := func() typing.Keystroke { return typing.Keystroke{Typed: 0} }

	tests := []struct {
		name      string
		log       []typing.Keystroke
		elapsedMs int64
		want      float64
	}{
		{"empty_log", nil, 60000, 0},
		{"below_500ms_guard", []typing.Keystroke{fwd('a'), fwd('b')}, 400, 0},
		{"zero_elapsed", []typing.Keystroke{fwd('a')}, 0, 0},
		{"negative_elapsed", []typing.Keystroke{fwd('a')}, -100, 0},
		{
			"forward_only",
			[]typing.Keystroke{fwd('a'), fwd('b'), fwd('c'), fwd('d'), fwd('e'),
				fwd('f'), fwd('g'), fwd('h'), fwd('i'), fwd('j')},
			60000,
			2.0, // 10 chars / 5 / (60000/60000) = 2.0 WPM
		},
		{
			"with_backspaces",
			[]typing.Keystroke{fwd('a'), fwd('b'), bsp(), fwd('c')},
			60000,
			// 3 forward keystrokes (backspace Typed==0 excluded), elapsed=60s → 3/5/1 = 0.6
			0.6,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LiveWPM(tt.log, tt.elapsedMs)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("LiveWPM(%d keystrokes, %dms) = %v, want %v",
					len(tt.log), tt.elapsedMs, got, tt.want)
			}
		})
	}
}

func TestLiveWPMFromCount(t *testing.T) {
	if got := LiveWPMFromCount(25, 60000); got != 5 {
		t.Fatalf("LiveWPMFromCount() = %v, want 5", got)
	}
	if got := LiveWPMFromCount(25, 499); got != 0 {
		t.Fatalf("LiveWPMFromCount below guard = %v, want 0", got)
	}
	if got := LiveWPMFromCount(0, 60000); got != 0 {
		t.Fatalf("LiveWPMFromCount zero count = %v, want 0", got)
	}
}
