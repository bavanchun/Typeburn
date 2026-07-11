package typing_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

func TestStrictEngine(t *testing.T) {
	t.Run("strict mode blocks incorrect rune and logs error", func(t *testing.T) {
		// Target is "hello", strict=true
		e := typing.NewStrict("hello", config.ModeWords, 1, true)

		// Type correct 'h'
		e.Apply('h', 100)
		if len(e.Typed()) != 1 {
			t.Errorf("expected typed length 1, got %d", len(e.Typed()))
		}
		if e.States()[1] != typing.Current {
			t.Errorf("expected cursor at index 1, got %v", e.States()[1])
		}

		// Type incorrect 'x' (target is 'e')
		e.Apply('x', 200)
		if len(e.Typed()) != 1 {
			t.Errorf("expected typed length to remain 1, got %d", len(e.Typed()))
		}
		if e.States()[1] != typing.Current {
			t.Errorf("expected cursor to remain at index 1, got %v", e.States()[1])
		}

		log := e.Log()
		if len(log) != 2 {
			t.Errorf("expected log length 2, got %d", len(log))
		}
		if log[1].Typed != 'x' || log[1].Correct != false || log[1].Target != 'e' {
			t.Errorf("unexpected error log entry: %+v", log[1])
		}

		// Type correct 'e'
		e.Apply('e', 300)
		if len(e.Typed()) != 2 {
			t.Errorf("expected typed length 2, got %d", len(e.Typed()))
		}
		if e.States()[2] != typing.Current {
			t.Errorf("expected cursor to advance to index 2, got %v", e.States()[2])
		}
	})
}
