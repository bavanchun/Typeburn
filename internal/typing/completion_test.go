package typing_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

func TestComplete_ModeTime(t *testing.T) {
	e := typing.New("hello world test", config.ModeTime, 30000) // 30s in ms
	if e.Complete(29999) {
		t.Error("should not complete before time limit")
	}
	if !e.Complete(30000) {
		t.Error("should complete at exact time limit")
	}
	if !e.Complete(30001) {
		t.Error("should complete after time limit")
	}
}

func TestComplete_ModeWords(t *testing.T) {
	// wordTarget = 3 → need 3 completed words
	e := typing.New("hello world test", config.ModeWords, 3)
	applyAll(e, "hello ")
	if e.Complete(0) {
		t.Error("1 word should not complete 3-word target")
	}
	applyAll(e, "world ")
	if e.Complete(0) {
		t.Error("2 words should not complete 3-word target")
	}
	applyAll(e, "test")
	if !e.Complete(0) {
		t.Error("3 words typed should complete 3-word target")
	}
}

func TestComplete_ModeQuote(t *testing.T) {
	e := typing.New("hi", config.ModeQuote, 0)
	applyAll(e, "h")
	if e.Complete(0) {
		t.Error("partial should not complete")
	}
	applyAll(e, "i")
	if !e.Complete(0) {
		t.Error("exact match should complete")
	}
}

func TestComplete_DefaultMode(t *testing.T) {
	// Unknown mode should return false
	e := typing.New("test", "invalid", 0) // unrecognized mode
	applyAll(e, "test")
	if e.Complete(0) {
		t.Error("unknown mode should never complete")
	}
}
