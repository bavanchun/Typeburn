package typing_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/mode"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// TestProgress_IsACountInEveryMode pins the contract a caller rendering
// "done / total" or a percentage relies on: both halves are the same unit, and
// total is a real denominator.
//
// A timed run's limit is a deadline in milliseconds. Reporting it as the total
// made a 30-second run read "0 / 30000", and a Code run reported a total of
// zero, which is a percentage nobody can compute.
func TestProgress_IsACountInEveryMode(t *testing.T) {
	const target = "one two three four"

	t.Run("time reports words, not the millisecond deadline", func(t *testing.T) {
		e := typing.New(target, mode.ModeTime, 30*1000)
		done, total := e.Progress()
		if total != 4 {
			t.Errorf("want total=4 words, got %d", total)
		}
		if done != 0 {
			t.Errorf("want done=0 before typing, got %d", done)
		}

		ts := int64(1000)
		for _, r := range "one two " {
			e.Apply(r, ts)
			ts += 100
		}
		if done, total = e.Progress(); done != 2 || total != 4 {
			t.Errorf("want 2/4 after two words, got %d/%d", done, total)
		}
	})

	t.Run("code reports runes against a non-zero total", func(t *testing.T) {
		e := typing.New("x = 1", mode.ModeCode, 0)
		done, total := e.Progress()
		if total != 5 {
			t.Errorf("want total=5 runes, got %d", total)
		}
		if done != 0 {
			t.Errorf("want done=0, got %d", done)
		}

		ts := int64(1000)
		for _, r := range "x =" {
			e.Apply(r, ts)
			ts += 100
		}
		if done, total = e.Progress(); done != 3 || total != 5 {
			t.Errorf("want 3/5, got %d/%d", done, total)
		}
	})

	t.Run("every mode reports done within total", func(t *testing.T) {
		for _, m := range []mode.Mode{mode.ModeTime, mode.ModeWords, mode.ModeQuote, mode.ModeCode} {
			e := typing.New(target, m, 4)
			ts := int64(1000)
			for _, r := range target + "zzz" { // overtype past the end
				e.Apply(r, ts)
				ts += 100
			}
			done, total := e.Progress()
			if total <= 0 {
				t.Errorf("%v: total=%d is not a denominator", m, total)
			}
			if done > total {
				t.Errorf("%v: done=%d exceeds total=%d", m, done, total)
			}
		}
	})
}

// TestStrict_RefusesKeysPastTheEndOfTheTarget: strict mode stops the cursor on
// any wrong key, and a key typed past the end of the target has nothing to
// match, so it is wrong. Letting those through filled the buffer with extra
// characters the mode exists to prevent.
func TestStrict_RefusesKeysPastTheEndOfTheTarget(t *testing.T) {
	e := typing.NewStrict("ab", mode.ModeWords, 1, true)
	ts := int64(1000)
	for _, r := range "ab" {
		e.Apply(r, ts)
		ts += 100
	}
	for i := 0; i < 10; i++ {
		e.Apply('z', ts)
		ts += 100
	}

	if got := string(e.Typed()); got != "ab" {
		t.Errorf("buffer is %q; strict mode must not accept a key past the target", got)
	}
	blocked := 0
	for _, k := range e.Log() {
		if k.Blocked {
			blocked++
		}
	}
	if blocked != 10 {
		t.Errorf("want 10 refused keystrokes recorded, got %d", blocked)
	}
}
