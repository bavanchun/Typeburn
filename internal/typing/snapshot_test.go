package typing_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/internal/mode"
	"github.com/bavanchun/Typeburn/internal/typing"
)

func TestTypedReturnsCurrentBufferCopy(t *testing.T) {
	e := typing.New("hello", mode.ModeWords, 1)
	e.Apply('h', 100)
	e.Apply('x', 200)
	e.Backspace(300)
	e.Apply('e', 400)

	got := e.Typed()
	if string(got) != "he" {
		t.Fatalf("Typed() = %q, want %q", string(got), "he")
	}

	got[0] = 'x'
	again := e.Typed()
	if string(again) != "he" {
		t.Fatalf("Typed() leaked mutable state: got %q", string(again))
	}
}

func TestForwardKeystrokesIgnoresBackspace(t *testing.T) {
	e := typing.New("hello", mode.ModeWords, 1)
	e.Apply('h', 100)
	e.Apply('x', 200)
	e.Backspace(300)
	e.Apply('e', 400)

	if got := e.ForwardKeystrokes(); got != 3 {
		t.Fatalf("ForwardKeystrokes() = %d, want 3", got)
	}
}
