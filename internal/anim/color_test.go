package anim

import (
	"image/color"
	"testing"
)

func TestLerpColorEndpoints(t *testing.T) {
	from := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	to := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	if got := LerpColor(from, to, 0).(color.RGBA); got != from {
		t.Errorf("t=0 got %+v want %+v", got, from)
	}
	if got := LerpColor(from, to, 1).(color.RGBA); got != to {
		t.Errorf("t=1 got %+v want %+v", got, to)
	}
}

func TestLerpColorMidpoint(t *testing.T) {
	from := color.RGBA{R: 0, G: 0, B: 0, A: 255}
	to := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	got := LerpColor(from, to, 0.5).(color.RGBA)
	want := color.RGBA{R: 100, G: 50, B: 25, A: 255}
	if got != want {
		t.Errorf("midpoint got %+v want %+v", got, want)
	}
}

func TestLerpColorClampsT(t *testing.T) {
	from := color.RGBA{R: 10, G: 20, B: 30, A: 255}
	to := color.RGBA{R: 200, G: 100, B: 50, A: 255}
	if got := LerpColor(from, to, -1).(color.RGBA); got != from {
		t.Errorf("t<0 should clamp to from, got %+v", got)
	}
	if got := LerpColor(from, to, 5).(color.RGBA); got != to {
		t.Errorf("t>1 should clamp to to, got %+v", got)
	}
}

func TestLerpColorNilPassthrough(t *testing.T) {
	x := color.RGBA{R: 1, G: 2, B: 3, A: 255}
	if got := LerpColor(nil, x, 0.5); got != nil {
		t.Errorf("LerpColor(nil,x) = %v want nil", got)
	}
	if got := LerpColor(x, nil, 0.5); got != nil {
		t.Errorf("LerpColor(x,nil) = %v want nil", got)
	}
	if got := LerpColor(nil, nil, 0.5); got != nil {
		t.Errorf("LerpColor(nil,nil) = %v want nil", got)
	}
}

// 65535/257 == 255 exactly: a full-white color.RGBA round-trips through RGBA().
func TestLerpColorFullChannelExact(t *testing.T) {
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	if got := LerpColor(white, white, 0.5).(color.RGBA); got != white {
		t.Errorf("white lerp got %+v want %+v", got, white)
	}
}
