package ui

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct{ name, in, want string }{
		{"plain", "hello", "hello"},
		{"sgr_color", "\x1b[31mred\x1b[0m", "red"},
		{"sgr_multi", "\x1b[1;31mX\x1b[0m", "X"},
		{"cursor_up", "a\x1b[Ab", "ab"},            // ESC[A non-SGR CSI
		{"erase_screen", "a\x1b[2Jb", "ab"},        // ESC[2J
		{"scroll_region", "a\x1b[1;5rb", "ab"},     // ESC[1;5r
		{"trailing_csi", "x\x1b[A", "x"},           // CSI at end of string
		{"trailing_incomplete_csi", "x\x1b[", "x"}, // ESC[ with no final byte (EOF in CSI)
		{"two_byte_esc", "a\x1bMb", "ab"},          // ESC M (RI) 2-byte sequence
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripANSI(tt.in); got != tt.want {
				t.Errorf("stripANSI(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
