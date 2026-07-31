package updateui

import "strings"

// stripANSI removes SGR escape sequences so border and padding arithmetic
// counts terminal cells rather than bytes.
func stripANSI(s string) string {
	var b strings.Builder
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc && r == 'm':
			esc = false
		case !esc:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// visLen counts visible runes, ignoring SGR escapes.
func visLen(s string) int {
	n := 0
	esc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			esc = true
		case esc && r == 'm':
			esc = false
		case !esc:
			n++
		}
	}
	return n
}
