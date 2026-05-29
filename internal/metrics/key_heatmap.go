package metrics

import (
	"sort"
	"strconv"
	"unicode"

	"github.com/bavanchun/Typeburn/internal/typing"
)

// KeyMiss is a per-key fumble tally for one case-folded target rune.
// Attempts counts every forward keystroke aimed at that target (correct or
// wrong); Misses counts only the wrong ones (including fumbles later corrected
// via backspace). Key holds the folded rune for stable sorting; Label is the
// display form (e.g. "␣" for space).
type KeyMiss struct {
	Key      rune   `json:"-"`   // case-folded target rune (sort key)
	Label    string `json:"key"` // display label: "e", "t", "␣" for space
	Misses   int    `json:"misses"`
	Attempts int    `json:"attempts"`
}

// heatmapTopN caps how many missed keys KeyHeatmap returns. Every surface
// (Result screen, CLI JSON, CLI table) inherits this bound.
const heatmapTopN = 8

// KeyHeatmap tallies per-key misses across the keystroke log via a single pass,
// then returns the top heatmapTopN keys with ≥1 miss, sorted deterministically:
// Misses desc → Attempts desc → Key asc.
//
// A counted keystroke is a forward keystroke (Typed != 0) aimed at a real target
// (Target != 0). Backspace markers (Typed == 0) and extra chars past the target
// end (Target == 0) are skipped. Targets are case-folded so a/A merge into one
// key. The pass is timing-independent: it does not depend on AFK trim order.
func KeyHeatmap(log []typing.Keystroke) []KeyMiss {
	tally := make(map[rune]*KeyMiss)

	for _, k := range log {
		if k.Typed == 0 || k.Target == 0 {
			continue // backspace marker or extra char — no target to attribute
		}
		key := unicode.ToLower(k.Target)
		km := tally[key]
		if km == nil {
			km = &KeyMiss{Key: key, Label: keyLabel(key)}
			tally[key] = km
		}
		km.Attempts++
		if !k.Correct {
			km.Misses++
		}
	}

	out := make([]KeyMiss, 0, len(tally))
	for _, km := range tally {
		if km.Misses > 0 {
			out = append(out, *km)
		}
	}
	if len(out) == 0 {
		return nil
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Misses != out[j].Misses {
			return out[i].Misses > out[j].Misses
		}
		if out[i].Attempts != out[j].Attempts {
			return out[i].Attempts > out[j].Attempts
		}
		return out[i].Key < out[j].Key
	})
	if len(out) > heatmapTopN {
		out = out[:heatmapTopN]
	}
	return out
}

// keyLabel returns the display label for a folded target rune.
//   - space → "␣" (U+2423 visible-space glyph)
//   - printable rune → the rune itself
//   - other control/whitespace (tab, newline; only reachable in Code mode) →
//     its Go-quoted escape with the surrounding quotes trimmed (e.g. `\t`).
func keyLabel(r rune) string {
	if r == ' ' {
		return "␣"
	}
	if unicode.IsPrint(r) {
		return string(r)
	}
	q := strconv.QuoteRune(r) // e.g. "'\\t'"
	return q[1 : len(q)-1]    // trim surrounding single quotes → `\t`
}
