package words

import (
	"github.com/bavanchun/Typeburn/internal/config"
)

// ForMode returns the target string for a typing test given the active mode
// and user-selected length. It is the primary entry point for Phase 4.
//
//   - ModeTime:  length is the test duration in seconds (15/30/60/120);
//     ForMode returns a large word buffer sized for the longest session.
//     The length parameter is accepted for signature uniformity but does not
//     affect buffer size — TimeBuffer always overshoots the maximum duration.
//   - ModeWords: length is the word count (10/25/50/100); returns exactly that
//     many space-separated words.
//   - ModeQuote: ql selects the desired quote bucket; length is ignored.
func ForMode(g *Generator, mode config.Mode, length int, ql QuoteLen) string {
	switch mode {
	case config.ModeWords:
		return g.Words(length)
	case config.ModeQuote:
		return g.Quote(ql).Text
	default: // ModeTime and any future modes default to a time buffer
		return g.TimeBuffer()
	}
}
