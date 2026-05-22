package notui

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestReadEvent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		kind EventKind
		rune rune
	}{
		{"ascii", "a", EventRune, 'a'},
		{"utf8", "é", EventRune, 'é'},
		{"backspace del", string([]byte{0x7f}), EventBackspace, 0},
		{"backspace bs", string([]byte{0x08}), EventBackspace, 0},
		{"escape", "\x1b[A", EventNone, 0},
		{"ctrl-c", "\x03", EventAbort, 0},
		{"ss3 f1", "\x1bOP", EventNone, 0},
		{"csi tilde", "\x1b[3~", EventNone, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ev, err := ReadEvent(bufio.NewReader(strings.NewReader(tt.in)))
			if err != nil {
				t.Fatal(err)
			}
			if ev.Kind != tt.kind || ev.Rune != tt.rune {
				t.Fatalf("want %v/%q, got %v/%q", tt.kind, tt.rune, ev.Kind, ev.Rune)
			}
		})
	}
}

// TestReadEvent_SplitEscape verifies that an ESC arriving at the end of one
// read buffer is correctly joined with the introducer/final bytes that arrive
// in the next read. Under the old buffer-gated discardEscape, the '[' would
// have been emitted as a stray EventRune.
func TestReadEvent_SplitEscape(t *testing.T) {
	r := bufio.NewReader(io.MultiReader(
		strings.NewReader("\x1b"),
		strings.NewReader("[Ax"),
	))
	ev, err := ReadEvent(r) // consumes ESC + [A across the buffer split
	if err != nil || ev.Kind != EventNone {
		t.Fatalf("escape: want EventNone, got %v (err %v)", ev.Kind, err)
	}
	ev, err = ReadEvent(r) // 'x' must be next — not a stray '['
	if err != nil {
		t.Fatal(err)
	}
	if ev.Kind != EventRune || ev.Rune != 'x' {
		t.Fatalf("after split escape: want rune 'x', got %v/%q", ev.Kind, ev.Rune)
	}
}

// TestReadEvent_StandaloneESC documents that the byte following a bare ESC is
// consumed as the "introducer" and dropped. This is an accepted trade-off:
// bare ESC is not a typing input in raw mode.
func TestReadEvent_StandaloneESC(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\x1bx"))
	ev, err := ReadEvent(r)
	if err != nil || ev.Kind != EventNone {
		t.Fatalf("standalone ESC: want EventNone, got %v (err %v)", ev.Kind, err)
	}
	// 'x' was consumed as the introducer — stream is now empty.
	_, err = ReadEvent(r)
	if err == nil {
		t.Fatal("expected EOF after standalone ESC consumed the next byte")
	}
}

// TestReadEvent_EscThenCtrlC verifies that Ctrl-C pressed after ESC is
// preserved: discardEscape unreads 0x03 so the outer loop returns EventAbort.
func TestReadEvent_EscThenCtrlC(t *testing.T) {
	r := bufio.NewReader(strings.NewReader("\x1b\x03"))
	ev, err := ReadEvent(r) // ESC triggers discardEscape, which unreads 0x03
	if err != nil || ev.Kind != EventNone {
		t.Fatalf("ESC: want EventNone, got %v (err %v)", ev.Kind, err)
	}
	ev, err = ReadEvent(r) // 0x03 was unread — must come back as EventAbort
	if err != nil || ev.Kind != EventAbort {
		t.Fatalf("Ctrl-C after ESC: want EventAbort, got %v (err %v)", ev.Kind, err)
	}
}
