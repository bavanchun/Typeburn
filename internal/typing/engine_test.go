package typing_test

import (
	"testing"

	"monkeytype-tui/internal/config"
	"monkeytype-tui/internal/typing"
)

// TestBackspace validates deletion and corrected-error behaviour.
func TestBackspace(t *testing.T) {
	t.Run("Backspace removes last typed rune", func(t *testing.T) {
		e := typing.New("hello", config.ModeWords, 1)
		e.Apply('h', 100)
		e.Apply('e', 200)
		e.Backspace(300)
		states := e.States()
		// index 0 Correct, index 1 back to Current
		if states[0] != typing.Correct {
			t.Errorf("index 0: want Correct, got %v", states[0])
		}
		if states[1] != typing.Current {
			t.Errorf("index 1: want Current after backspace, got %v", states[1])
		}
	})

	t.Run("Corrected error results in Correct final state", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('x', 100) // wrong
		e.Backspace(200)
		e.Apply('h', 300) // correct
		states := e.States()
		if states[0] != typing.Correct {
			t.Errorf("corrected error: want Correct, got %v", states[0])
		}
	})

	t.Run("Backspace on empty typed buffer is a no-op", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Backspace(100) // should not panic
		states := e.States()
		if states[0] != typing.Current {
			t.Errorf("after no-op backspace: want Current at 0, got %v", states[0])
		}
	})

	t.Run("Log records keystrokes including backspace-causing key", func(t *testing.T) {
		e := typing.New("hi", config.ModeWords, 1)
		e.Apply('x', 100)
		e.Backspace(200)
		log := e.Log()
		// Apply + Backspace should both be recorded
		if len(log) < 2 {
			t.Errorf("want at least 2 log entries, got %d", len(log))
		}
	})
}

// TestClockStart validates that startMs is set on first Apply.
func TestClockStart(t *testing.T) {
	t.Run("Before first Apply, Progress reports 0 done", func(t *testing.T) {
		e := typing.New("hello world", config.ModeWords, 2)
		done, total := e.Progress()
		if done != 0 {
			t.Errorf("want done=0 before typing, got %d", done)
		}
		if total != 2 {
			t.Errorf("want total=2, got %d", total)
		}
	})

	t.Run("StartMs zero before first keystroke", func(t *testing.T) {
		e := typing.New("hello", config.ModeWords, 1)
		if e.StartMs() != 0 {
			t.Errorf("want StartMs=0 before first key, got %d", e.StartMs())
		}
	})

	t.Run("StartMs set on first Apply", func(t *testing.T) {
		e := typing.New("hello", config.ModeWords, 1)
		e.Apply('h', 5000)
		if e.StartMs() != 5000 {
			t.Errorf("want StartMs=5000, got %d", e.StartMs())
		}
	})

	t.Run("StartMs not updated on subsequent Apply", func(t *testing.T) {
		e := typing.New("hello", config.ModeWords, 1)
		e.Apply('h', 5000)
		e.Apply('e', 6000)
		if e.StartMs() != 5000 {
			t.Errorf("want StartMs=5000, got %d", e.StartMs())
		}
	})
}

// TestCompletion validates mode-aware completion logic.
func TestCompletion(t *testing.T) {
	t.Run("Words mode: complete when N words typed", func(t *testing.T) {
		// target: "hi go", wordTarget=2
		e := typing.New("hi go", config.ModeWords, 2)
		typeString(e, "hi ", 100)
		if e.Complete(400) {
			t.Error("should not be complete after 1 word")
		}
		typeString(e, "go", 400)
		if !e.Complete(600) {
			t.Error("should be complete after 2 words")
		}
	})

	t.Run("Quote mode: complete when full text matched", func(t *testing.T) {
		target := "the quick"
		e := typing.New(target, config.ModeQuote, 0)
		typeString(e, "the quic", 100)
		if e.Complete(900) {
			t.Error("should not be complete with partial text")
		}
		typeString(e, "k", 900)
		if !e.Complete(1000) {
			t.Error("should be complete when full text matched")
		}
	})

	t.Run("Time mode: Complete false before endMs >= limit", func(t *testing.T) {
		// wordTarget used as limitMs in Time mode
		e := typing.New("hello world", config.ModeTime, 30000)
		typeString(e, "hello", 100)
		if e.Complete(29000) {
			t.Error("Time mode: should not complete before limit")
		}
	})

	t.Run("Time mode: Complete true when endMs >= limit", func(t *testing.T) {
		e := typing.New("hello world", config.ModeTime, 30000)
		typeString(e, "hello", 100)
		if !e.Complete(30000) {
			t.Error("Time mode: should complete at limit")
		}
	})

	t.Run("Words mode: exactly N words triggers completion", func(t *testing.T) {
		e := typing.New("a b c", config.ModeWords, 3)
		typeString(e, "a b ", 100)
		if e.Complete(500) {
			t.Error("2 words done, need 3")
		}
		typeString(e, "c", 500)
		if !e.Complete(600) {
			t.Error("should complete at exactly 3rd word")
		}
	})
}

// TestProgress validates Progress() counts.
func TestProgress(t *testing.T) {
	t.Run("Words mode progress counts words", func(t *testing.T) {
		e := typing.New("one two three", config.ModeWords, 3)
		typeString(e, "one ", 100)
		done, total := e.Progress()
		if done != 1 {
			t.Errorf("want done=1, got %d", done)
		}
		if total != 3 {
			t.Errorf("want total=3, got %d", total)
		}
	})

	t.Run("Quote mode progress counts runes", func(t *testing.T) {
		e := typing.New("hello", config.ModeQuote, 0)
		typeString(e, "hel", 100)
		done, total := e.Progress()
		if done != 3 {
			t.Errorf("want done=3, got %d", done)
		}
		if total != 5 {
			t.Errorf("want total=5, got %d", total)
		}
	})
}

// typeString is a helper that applies each rune of s with incrementing timestamps.
func typeString(e *typing.Engine, s string, startMs int64) {
	for i, r := range s {
		e.Apply(r, startMs+int64(i)*10)
	}
}
