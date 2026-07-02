package words

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
)

// TestForMode_TimeReturnsNonEmpty verifies ForMode(ModeTime) returns a
// non-empty buffer regardless of the length parameter.
func TestForMode_TimeReturnsNonEmpty(t *testing.T) {
	g := NewGenerator(1)
	for _, length := range []int{15, 30, 60, 120} {
		got := ForMode(g, config.ModeTime, length, QuoteShort, false, false)
		if got == "" {
			t.Errorf("ForMode(ModeTime, %d): got empty string", length)
		}
		// Time buffer should contain multiple words.
		if count := len(strings.Fields(got)); count < 10 {
			t.Errorf("ForMode(ModeTime, %d): too few words (%d)", length, count)
		}
	}
}

// TestForMode_WordsExactCount verifies ForMode(ModeWords, n) returns exactly n words.
func TestForMode_WordsExactCount(t *testing.T) {
	g := NewGenerator(42)
	for _, n := range []int{10, 25, 50, 100} {
		got := ForMode(g, config.ModeWords, n, QuoteShort, false, false)
		tokens := strings.Fields(got)
		if len(tokens) != n {
			t.Errorf("ForMode(ModeWords, %d): want %d words, got %d", n, n, len(tokens))
		}
	}
}

// TestForMode_QuoteNonEmpty verifies ForMode(ModeQuote) returns non-empty text
// for every supported bucket (Short, Medium, Long).
func TestForMode_QuoteNonEmpty(t *testing.T) {
	g := NewGenerator(7)
	for _, ql := range []QuoteLen{QuoteShort, QuoteMedium, QuoteLong} {
		got := ForMode(g, config.ModeQuote, 0, ql, false, false)
		if got == "" {
			t.Errorf("ForMode(ModeQuote, ql=%d): got empty string", ql)
		}
	}
}

// TestForMode_WordsDeterministic verifies same seed + same call → same result.
func TestForMode_WordsDeterministic(t *testing.T) {
	a := NewGenerator(99)
	b := NewGenerator(99)
	if ForMode(a, config.ModeWords, 25, QuoteShort, false, false) != ForMode(b, config.ModeWords, 25, QuoteShort, false, false) {
		t.Error("ForMode(ModeWords, 25): same seed gave different results")
	}
}

// TestForMode_TimeDeterministic verifies same seed → same time buffer.
func TestForMode_TimeDeterministic(t *testing.T) {
	a := NewGenerator(5)
	b := NewGenerator(5)
	if ForMode(a, config.ModeTime, 30, QuoteShort, false, false) != ForMode(b, config.ModeTime, 30, QuoteShort, false, false) {
		t.Error("ForMode(ModeTime, 30): same seed gave different results")
	}
}

// TestForMode_UnknownModeFallsBackToTime verifies that an unknown mode
// returns the time buffer (default branch).
func TestForMode_UnknownModeFallsBackToTime(t *testing.T) {
	g := NewGenerator(3)
	got := ForMode(g, config.Mode("unknown"), 30, QuoteShort, false, false)
	if got == "" {
		t.Error("ForMode(unknown mode): expected time buffer fallback, got empty string")
	}
}

// TestForMode_WordsAppliesPunctuationAndNumbers verifies ForMode threads the
// punctuation/numbers flags into ModeWords generation.
func TestForMode_WordsAppliesPunctuationAndNumbers(t *testing.T) {
	plainGen := NewGenerator(11)
	plain := ForMode(plainGen, config.ModeWords, 60, QuoteShort, false, false)

	optGen := NewGenerator(11)
	got := ForMode(optGen, config.ModeWords, 60, QuoteShort, true, true)

	if got == plain {
		t.Error("ForMode(ModeWords, punctuation=true, numbers=true) produced unchanged output")
	}
	if len(strings.Fields(got)) != len(strings.Fields(plain)) {
		t.Error("ForMode(ModeWords, ...): token count changed with options applied")
	}
}

// TestForMode_TimeAppliesPunctuationAndNumbers verifies ForMode threads the
// punctuation/numbers flags into ModeTime generation.
func TestForMode_TimeAppliesPunctuationAndNumbers(t *testing.T) {
	plainGen := NewGenerator(21)
	plain := ForMode(plainGen, config.ModeTime, 30, QuoteShort, false, false)

	optGen := NewGenerator(21)
	got := ForMode(optGen, config.ModeTime, 30, QuoteShort, true, true)

	if got == plain {
		t.Error("ForMode(ModeTime, punctuation=true, numbers=true) produced unchanged output")
	}
}

// TestForMode_QuoteIgnoresPunctuationAndNumbers verifies ForMode(ModeQuote)
// is byte-identical regardless of the flags — Quote mode already has real
// punctuation and must not be transformed.
func TestForMode_QuoteIgnoresPunctuationAndNumbers(t *testing.T) {
	a := NewGenerator(7)
	b := NewGenerator(7)
	plain := ForMode(a, config.ModeQuote, 0, QuoteShort, false, false)
	got := ForMode(b, config.ModeQuote, 0, QuoteShort, true, true)
	if got != plain {
		t.Errorf("ForMode(ModeQuote, ...): flags must be ignored:\nwant %q\ngot  %q", plain, got)
	}
}
