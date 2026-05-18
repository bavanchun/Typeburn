package typing_test

import (
	"testing"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/typing"
)

// TestCharStates validates all 6 CharState classifications via engine.States().
func TestCharStates(t *testing.T) {
	t.Run("Untyped chars before cursor are Untyped", func(t *testing.T) {
		e := typing.New("hello", config.ModeWords, 1)
		states := e.States()
		// nothing typed yet — all Untyped except index 0 which is Current
		for i, s := range states {
			if i == 0 {
				if s != typing.Current {
					t.Errorf("index 0: want Current, got %v", s)
				}
			} else {
				if s != typing.Untyped {
					t.Errorf("index %d: want Untyped, got %v", i, s)
				}
			}
		}
	})

	t.Run("Correct char classification", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('h', 100)
		states := e.States()
		if states[0] != typing.Correct {
			t.Errorf("want Correct, got %v", states[0])
		}
		if states[1] != typing.Current {
			t.Errorf("want Current at index 1, got %v", states[1])
		}
	})

	t.Run("Incorrect char classification", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('x', 100)
		states := e.States()
		if states[0] != typing.Incorrect {
			t.Errorf("want Incorrect, got %v", states[0])
		}
	})

	t.Run("IncorrectSpace: wrong char where target is space", func(t *testing.T) {
		// target: "a b", type 'a' then 'x' where space is expected
		e := typing.New("a b", config.ModeWords, 2)
		e.Apply('a', 100)
		e.Apply('x', 200) // space slot → IncorrectSpace
		states := e.States()
		if states[1] != typing.IncorrectSpace {
			t.Errorf("want IncorrectSpace at index 1, got %v", states[1])
		}
	})

	t.Run("Extra chars past word end", func(t *testing.T) {
		// target "hi", type "hix" — third char is Extra
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('h', 100)
		e.Apply('i', 200)
		e.Apply('x', 300) // extra
		states := e.States()
		if len(states) < 3 {
			t.Fatalf("want at least 3 states, got %d", len(states))
		}
		if states[2] != typing.Extra {
			t.Errorf("want Extra at index 2, got %v", states[2])
		}
	})

	t.Run("Current follows cursor position", func(t *testing.T) {
		e := typing.New("abc", config.ModeWords, 1)
		e.Apply('a', 100)
		states := e.States()
		// cursor now at index 1
		if states[1] != typing.Current {
			t.Errorf("cursor at 1: want Current, got %v", states[1])
		}
	})

	t.Run("Multi-byte rune handling — café", func(t *testing.T) {
		e := typing.New("café", config.ModeWords, 1)
		for _, r := range []rune("café") {
			e.Apply(r, 100)
		}
		states := e.States()
		for i, s := range states {
			if s != typing.Correct {
				t.Errorf("café index %d: want Correct, got %v", i, s)
			}
		}
	})

	t.Run("CJK multi-byte rune handling", func(t *testing.T) {
		e := typing.New("你好", config.ModeWords, 1)
		e.Apply('你', 100)
		states := e.States()
		if states[0] != typing.Correct {
			t.Errorf("CJK index 0: want Correct, got %v", states[0])
		}
		if states[1] != typing.Current {
			t.Errorf("CJK index 1: want Current, got %v", states[1])
		}
	})
}
