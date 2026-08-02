package app

import "strings"

// stripANSI removes ANSI CSI escape sequences (ESC [ ... final byte 0x40-0x7E)
// so assertions are colour-agnostic and frame widths can be measured in cells.
//
// It is a state machine rather than a scan to the next 'm' because it is also
// the measurement primitive for the frame-fits assertions. A scan-to-'m' works
// on SGR alone, but on any other CSI — ESC[2K, ESC[H — it swallows everything
// up to the next 'm' anywhere downstream, quietly shortening the line. A frame
// that overflowed would then measure as fitting.
//
// Mirrors the implementation in internal/ui; Go test helpers cannot cross the
// package boundary, and neither copy belongs in the shipped binary.
func stripANSI(s string) string {
	var out strings.Builder
	const (
		stNorm = iota
		stEsc  // saw ESC, expecting introducer
		stCSI  // inside CSI params/intermediates, waiting for final byte
	)
	state := stNorm
	for _, r := range s {
		switch state {
		case stNorm:
			if r == '\x1b' {
				state = stEsc
			} else {
				out.WriteRune(r)
			}
		case stEsc:
			if r == '[' {
				state = stCSI
			} else {
				state = stNorm // non-[ introducer: drop it, done with this escape
			}
		case stCSI:
			if r >= '@' && r <= '~' { // CSI final byte 0x40..0x7E
				state = stNorm
			}
			// else: param/intermediate byte — keep dropping
		}
	}
	return out.String()
}
