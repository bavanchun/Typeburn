package ui

import "math"

// Tier classifies the terminal width into layout bands used by footer, logo,
// and word-stream to adapt their rendering per design §4.1.
type Tier int

const (
	// TierDegraded: termW<60 or termH<20 — root blocks all screen rendering.
	TierDegraded Tier = iota
	// TierNarrow: 60 ≤ termW < 72 — footer glyphs only, word-stream uses termW-4.
	TierNarrow
	// TierMid: 72 ≤ termW < 88 — content width termW-8, footers may show short actions.
	TierMid
	// TierWide: termW ≥ 88 — full layout, 80-col content, full footer labels.
	TierWide
)

// WidthTier classifies a terminal width into the appropriate layout Tier.
// termH is used only to detect TierDegraded (below 20 rows).
func WidthTier(termW, termH int) Tier {
	if termW < 60 || termH < 20 {
		return TierDegraded
	}
	switch {
	case termW < 72:
		return TierNarrow
	case termW < 88:
		return TierMid
	default:
		return TierWide
	}
}

// ContentWidth returns the word-stream content width for a given terminal width
// and tier.
//
//   - Wide (≥88): ~82% of termW, floored at 80 (never narrower than a mid
//     terminal) and capped at termW-8 so the centered block keeps breathing
//     room. The stream grows with the screen instead of being stuck at 80.
//   - Mid (72–87): termW-8
//   - Narrow (60–71): termW-4
//   - Degraded or tiny: 20 (defensive minimum; caller should not reach here)
func ContentWidth(termW int, tier Tier) int {
	switch tier {
	case TierWide:
		w := int(math.Round(float64(termW) * 0.82))
		if w < 80 {
			w = 80
		}
		if maxW := termW - 8; w > maxW {
			w = maxW
		}
		return w
	case TierMid:
		w := termW - 8
		if w < 20 {
			return 20
		}
		return w
	case TierNarrow:
		w := termW - 4
		if w < 20 {
			return 20
		}
		return w
	default:
		return 20
	}
}
