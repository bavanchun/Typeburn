package words

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/mode"
)

// TestWords_ClampsAtTheCap: the slice is sized directly from the caller's
// number, and that number reaches here from a flag and from a settings file.
// Two billion words is not a long test, it is an out-of-memory kill.
//
// The assertion is made one word past the cap rather than at two billion: a
// regression must fail this test, not take the machine down with it.
func TestWords_ClampsAtTheCap(t *testing.T) {
	g := NewGenerator(42)

	got := g.Words(MaxWords + 1)

	if n := len(strings.Fields(got)); n != MaxWords {
		t.Errorf("got %d words, want the cap of %d", n, MaxWords)
	}
}

// TestWords_LeavesRealCountsAlone pins both sides of the boundary so the clamp
// cannot creep down into the range the UI actually offers.
func TestWords_LeavesRealCountsAlone(t *testing.T) {
	g := NewGenerator(42)
	for _, n := range []int{1, 10, 25, 50, 100, MaxWords} {
		if got := len(strings.Fields(g.Words(n))); got != n {
			t.Errorf("Words(%d) produced %d words", n, got)
		}
	}
}

// TestForMode_CodeProducesNoWords: Code mode types the user's file. A silent
// fall-through to the time buffer would start them on random English prose and
// look, from the outside, like the file failed to load.
func TestForMode_CodeProducesNoWords(t *testing.T) {
	g := NewGenerator(42)

	if got := ForMode(g, mode.ModeCode, 0, QuoteShort, false, false); got != "" {
		t.Errorf("ForMode(ModeCode) returned %d bytes of generated text: %q", len(got), got)
	}
}
