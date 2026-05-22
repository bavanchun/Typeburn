package notui

import (
	"bufio"
	"io"
	"unicode/utf8"
)

type EventKind int

const (
	EventNone EventKind = iota
	EventRune
	EventBackspace
	EventAbort
)

type Event struct {
	Kind EventKind
	Rune rune
}

func ReadEvent(r *bufio.Reader) (Event, error) {
	b, err := r.ReadByte()
	if err != nil {
		return Event{}, err
	}
	switch b {
	case 0x03:
		return Event{Kind: EventAbort}, nil
	case 0x08, 0x7f:
		return Event{Kind: EventBackspace}, nil
	case 0x1b:
		discardEscape(r)
		return Event{Kind: EventNone}, nil
	}
	if b < utf8.RuneSelf {
		return Event{Kind: EventRune, Rune: rune(b)}, nil
	}
	buf := []byte{b}
	for len(buf) < utf8.UTFMax {
		next, err := r.Peek(1)
		if err != nil {
			if err == io.EOF {
				break
			}
			return Event{}, err
		}
		if next[0]&0xc0 != 0x80 {
			break
		}
		_, _ = r.ReadByte()
		buf = append(buf, next[0])
		if rr, size := utf8.DecodeRune(buf); rr != utf8.RuneError && size == len(buf) {
			return Event{Kind: EventRune, Rune: rr}, nil
		}
	}
	rr, _ := utf8.DecodeRune(buf)
	return Event{Kind: EventRune, Rune: rr}, nil
}

func discardEscape(r *bufio.Reader) {
	b, err := r.ReadByte() // always block — handles ESC arriving at a read-buffer boundary
	if err != nil {
		return
	}
	// Preserve abort/EOF signals so the outer ReadEvent loop can handle them.
	// Without this, ESC then Ctrl-C would silently drop 0x03 and make the
	// session un-abortable.
	if b == 0x03 || b == 0x04 {
		_ = r.UnreadByte()
		return
	}
	if b != '[' && b != 'O' { // standalone ESC or 2-byte sequence — done
		return
	}
	for {
		b, err = r.ReadByte()
		if err != nil {
			return
		}
		if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
			return
		}
	}
}
