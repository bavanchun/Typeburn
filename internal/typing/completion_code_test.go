package typing_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// applyAll feeds every rune of s into the engine in order.
func applyAll(e *typing.Engine, s string) {
	ms := int64(0)
	for _, r := range s {
		ms += 100
		e.Apply(r, ms)
	}
}

// TestComplete_ModeCode_ExactMatchInclWhitespace verifies ModeCode finishes
// only on an exact full-text match, treating literal '\n' and '\t' as normal
// target runes (same rule as ModeQuote).
func TestComplete_ModeCode_ExactMatchInclWhitespace(t *testing.T) {
	const target = "func f() {\n\treturn 1\n}"

	t.Run("incomplete until fully typed", func(t *testing.T) {
		e := typing.New(target, config.ModeCode, 0)
		applyAll(e, "func f() {")
		if e.Complete(0) {
			t.Fatal("should not be complete mid-snippet")
		}
	})

	t.Run("complete on exact match incl newline/tab", func(t *testing.T) {
		e := typing.New(target, config.ModeCode, 0)
		applyAll(e, target) // includes '\n' and '\t' runes
		if !e.Complete(0) {
			t.Fatal("should be complete after exact full match")
		}
	})

	t.Run("wrong char is not complete", func(t *testing.T) {
		e := typing.New(target, config.ModeCode, 0)
		applyAll(e, "func g() {\n\treturn 1\n}") // f->g
		if e.Complete(0) {
			t.Fatal("must not complete on a mismatching buffer of equal length")
		}
	})

	t.Run("trailing extra runes are not complete", func(t *testing.T) {
		e := typing.New(target, config.ModeCode, 0)
		applyAll(e, target+";")
		if e.Complete(0) {
			t.Fatal("extra runes past the target must not count as complete")
		}
	})

	t.Run("no word-count dependency", func(t *testing.T) {
		// A single token with no spaces still completes on exact match.
		e := typing.New("xyz", config.ModeCode, 0)
		applyAll(e, "xyz")
		if !e.Complete(0) {
			t.Fatal("ModeCode must not require word boundaries")
		}
	})
}
