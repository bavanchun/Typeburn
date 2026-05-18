package words

import (
	_ "embed"
	"strings"
	"unicode/utf8"
)

//go:embed data/quotes.txt
var quotesRaw string

// QuoteLen classifies a quote by its rune length.
type QuoteLen int

const (
	// QuoteShort: text < 100 runes.
	QuoteShort QuoteLen = iota
	// QuoteMedium: 100 <= text < 250 runes.
	QuoteMedium
	// QuoteLong: text >= 250 runes.
	QuoteLong
)

// Quote is a single curated quote with attribution.
type Quote struct {
	Text   string
	Source string
	Bucket QuoteLen
}

// buckets holds the parsed quotes indexed by QuoteLen.
// Populated once at init from the embedded quotes.txt file.
var buckets [3][]Quote

func init() {
	for _, line := range strings.Split(quotesRaw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: text TAB source
		idx := strings.LastIndex(line, "\t")
		if idx < 0 {
			continue // malformed line — skip silently
		}
		text := strings.TrimSpace(line[:idx])
		source := strings.TrimSpace(line[idx+1:])
		if text == "" {
			continue
		}
		q := Quote{
			Text:   text,
			Source: source,
			Bucket: bucketFor(text),
		}
		buckets[q.Bucket] = append(buckets[q.Bucket], q)
	}
}

// bucketFor classifies a quote text into a QuoteLen bucket by rune count.
func bucketFor(text string) QuoteLen {
	n := utf8.RuneCountInString(text)
	switch {
	case n < 100:
		return QuoteShort
	case n < 250:
		return QuoteMedium
	default:
		return QuoteLong
	}
}

// Quote returns a random quote from the requested bucket using g's RNG.
// If the requested bucket is empty it falls back to the nearest non-empty
// bucket (medium → short → long).
func (g *Generator) Quote(l QuoteLen) Quote {
	order := fallbackOrder(l)
	for _, b := range order {
		if len(buckets[b]) > 0 {
			return buckets[b][g.rng.IntN(len(buckets[b]))]
		}
	}
	// Should never be reached when the embedded data file is non-empty.
	return Quote{Text: "The quick brown fox jumps over the lazy dog.", Source: ""}
}

// fallbackOrder returns the bucket preference order starting from l.
func fallbackOrder(l QuoteLen) []QuoteLen {
	switch l {
	case QuoteShort:
		return []QuoteLen{QuoteShort, QuoteMedium, QuoteLong}
	case QuoteMedium:
		return []QuoteLen{QuoteMedium, QuoteShort, QuoteLong}
	default:
		return []QuoteLen{QuoteLong, QuoteMedium, QuoteShort}
	}
}
