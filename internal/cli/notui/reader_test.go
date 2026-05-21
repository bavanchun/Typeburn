package notui

import (
	"bufio"
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
