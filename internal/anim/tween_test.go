package anim

import "testing"

func TestTweenProgressBounds(t *testing.T) {
	tw := Tween{StartMs: 1000, DurMs: 200} // linear (Ease nil)
	cases := []struct {
		now  int64
		want float64
	}{
		{900, 0},    // before start
		{1000, 0},   // at start
		{1100, 0.5}, // midpoint
		{1200, 1},   // at end
		{1300, 1},   // after end
	}
	for _, c := range cases {
		if got := tw.Progress(c.now); !approx(got, c.want) {
			t.Errorf("Progress(%d)=%v want %v", c.now, got, c.want)
		}
	}
}

func TestTweenProgressEased(t *testing.T) {
	tw := Tween{StartMs: 0, DurMs: 100, Ease: EaseOutQuad}
	// At raw t=0.5, EaseOutQuad(0.5) = 0.75.
	if got := tw.Progress(50); !approx(got, 0.75) {
		t.Errorf("eased Progress(50)=%v want 0.75", got)
	}
}

func TestTweenZeroDuration(t *testing.T) {
	tw := Tween{StartMs: 100, DurMs: 0}
	if got := tw.Progress(99); got != 0 {
		t.Errorf("zero-dur before start = %v want 0", got)
	}
	if got := tw.Progress(100); got != 1 {
		t.Errorf("zero-dur at start = %v want 1", got)
	}
	if !tw.Done(100) {
		t.Errorf("zero-dur should be Done at start")
	}
}

func TestTweenDone(t *testing.T) {
	tw := Tween{StartMs: 1000, DurMs: 200}
	if tw.Done(1199) {
		t.Errorf("Done(1199) true, want false")
	}
	if !tw.Done(1200) {
		t.Errorf("Done(1200) false, want true")
	}
}

func TestLerpFloat(t *testing.T) {
	if got := LerpFloat(0, 10, 0.5); !approx(got, 5) {
		t.Errorf("LerpFloat = %v want 5", got)
	}
	if got := LerpFloat(4, 8, 0); !approx(got, 4) {
		t.Errorf("LerpFloat t=0 = %v want 4", got)
	}
}

func TestLerpInt(t *testing.T) {
	cases := []struct {
		from, to int
		t        float64
		want     int
	}{
		{0, 100, 0, 0},
		{0, 100, 1, 100},
		{0, 100, 0.5, 50},
		{0, 10, 0.25, 3}, // 2.5 rounds to 3 (round-half-up)
		{0, 9, 0.999, 9}, // near-end lands on target
		{10, 0, 0.5, 5},  // descending
	}
	for _, c := range cases {
		if got := LerpInt(c.from, c.to, c.t); got != c.want {
			t.Errorf("LerpInt(%d,%d,%v)=%d want %d", c.from, c.to, c.t, got, c.want)
		}
	}
}
