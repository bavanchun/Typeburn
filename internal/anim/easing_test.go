package anim

import "testing"

func TestClamp01(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{-1, 0}, {-0.001, 0}, {0, 0}, {0.5, 0.5}, {1, 1}, {1.0001, 1}, {99, 1},
	}
	for _, c := range cases {
		if got := Clamp01(c.in); got != c.want {
			t.Errorf("Clamp01(%v)=%v want %v", c.in, got, c.want)
		}
	}
}

// easingFns are the shaping curves; all must satisfy f(0)=0, f(1)=1 and stay
// within [0,1] across the domain (monotonic non-decreasing).
func TestEasingEndpointsAndMonotonic(t *testing.T) {
	fns := map[string]func(float64) float64{
		"EaseOutCubic":  EaseOutCubic,
		"EaseOutQuad":   EaseOutQuad,
		"EaseInOutQuad": EaseInOutQuad,
	}
	for name, f := range fns {
		if got := f(0); !approx(got, 0) {
			t.Errorf("%s(0)=%v want 0", name, got)
		}
		if got := f(1); !approx(got, 1) {
			t.Errorf("%s(1)=%v want 1", name, got)
		}
		prev := f(0)
		for i := 1; i <= 100; i++ {
			x := float64(i) / 100
			v := f(x)
			if v < -1e-9 || v > 1+1e-9 {
				t.Errorf("%s(%v)=%v out of [0,1]", name, x, v)
			}
			if v < prev-1e-9 {
				t.Errorf("%s not monotonic at %v: %v < %v", name, x, v, prev)
			}
			prev = v
		}
	}
}

func TestEaseInOutQuadMidpoint(t *testing.T) {
	if got := EaseInOutQuad(0.5); !approx(got, 0.5) {
		t.Errorf("EaseInOutQuad(0.5)=%v want 0.5", got)
	}
}

func TestEaseOutQuadFasterStart(t *testing.T) {
	// EaseOut curves should be above the linear line in the first half.
	if EaseOutQuad(0.25) <= 0.25 {
		t.Errorf("EaseOutQuad(0.25)=%v should exceed linear 0.25", EaseOutQuad(0.25))
	}
	if EaseOutCubic(0.25) <= 0.25 {
		t.Errorf("EaseOutCubic(0.25)=%v should exceed linear 0.25", EaseOutCubic(0.25))
	}
}

func approx(a, b float64) bool {
	d := a - b
	return d < 1e-9 && d > -1e-9
}
