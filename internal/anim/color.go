package anim

import "image/color"

// LerpColor linearly interpolates between two colors in 8-bit RGB space.
// It works on stdlib image/color so this package never imports lipgloss; the
// caller converts the returned color.RGBA to a lipgloss color at the UI seam.
//
// If either input is nil (the NO_COLOR signal from theme.Color), it returns nil
// so callers can branch to an attribute-only render path instead of color math.
// t is clamped to [0,1]. The returned color is fully opaque (A=255).
func LerpColor(from, to color.Color, t float64) color.Color {
	if from == nil || to == nil {
		return nil
	}
	t = Clamp01(t)

	fr, fg, fb, _ := from.RGBA() // each channel 0–65535
	tr, tg, tb, _ := to.RGBA()

	return color.RGBA{
		R: lerpChannel(fr, tr, t),
		G: lerpChannel(fg, tg, t),
		B: lerpChannel(fb, tb, t),
		A: 255,
	}
}

// lerpChannel interpolates one 0–65535 channel pair and scales to 0–255.
// Division by 257 maps 65535→255 exactly (255*257 == 65535).
func lerpChannel(from, to uint32, t float64) uint8 {
	f := float64(from)
	v := f + (float64(to)-f)*t
	return uint8(v/257 + 0.5)
}
