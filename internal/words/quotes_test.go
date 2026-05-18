package words

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestQuotePack_AllBucketsNonEmpty(t *testing.T) {
	names := map[QuoteLen]string{
		QuoteShort:  "short",
		QuoteMedium: "medium",
		QuoteLong:   "long",
	}
	for bucket, name := range names {
		if len(buckets[bucket]) == 0 {
			t.Errorf("quote bucket %q is empty — add more quotes to data/quotes.txt", name)
		}
	}
}

func TestQuotePack_LengthClassification(t *testing.T) {
	for _, q := range buckets[QuoteShort] {
		n := utf8.RuneCountInString(q.Text)
		if n >= 100 {
			t.Errorf("short bucket contains quote with %d runes (≥100): %q", n, q.Text[:min(40, len(q.Text))])
		}
	}
	for _, q := range buckets[QuoteMedium] {
		n := utf8.RuneCountInString(q.Text)
		if n < 100 || n >= 250 {
			t.Errorf("medium bucket contains quote with %d runes (want 100–249): %q", n, q.Text[:min(40, len(q.Text))])
		}
	}
	for _, q := range buckets[QuoteLong] {
		n := utf8.RuneCountInString(q.Text)
		if n < 250 {
			t.Errorf("long bucket contains quote with %d runes (<250): %q", n, q.Text[:min(40, len(q.Text))])
		}
	}
}

func TestQuotePack_AllTextsNonEmpty(t *testing.T) {
	for _, bucket := range buckets {
		for _, q := range bucket {
			if strings.TrimSpace(q.Text) == "" {
				t.Errorf("quote has empty Text, source=%q", q.Source)
			}
		}
	}
}

func TestQuotePack_MinimumCount(t *testing.T) {
	total := 0
	for _, bucket := range buckets {
		total += len(bucket)
	}
	const minQuotes = 30
	if total < minQuotes {
		t.Errorf("total quotes: %d, want at least %d", total, minQuotes)
	}
}

func TestQuote_Deterministic(t *testing.T) {
	bucketCases := []QuoteLen{QuoteShort, QuoteMedium, QuoteLong}
	for _, l := range bucketCases {
		a := NewGenerator(42)
		b := NewGenerator(42)
		qa := a.Quote(l)
		qb := b.Quote(l)
		if qa.Text != qb.Text {
			t.Errorf("Quote(%d): same seed produced different text", l)
		}
	}
}

func TestQuote_FallbackNeverPanics(t *testing.T) {
	// Even with unusual bucket requests the generator must return a non-empty quote.
	g := NewGenerator(1)
	for _, l := range []QuoteLen{QuoteShort, QuoteMedium, QuoteLong} {
		q := g.Quote(l)
		if q.Text == "" {
			t.Errorf("Quote(%d): returned empty Text", l)
		}
	}
}

func TestForMode_NonEmpty(t *testing.T) {
	g := NewGenerator(7)
	cases := []struct {
		mode   string
		length int
		ql     QuoteLen
	}{
		{"time", 30, QuoteShort},
		{"words", 25, QuoteShort},
		{"quote", 0, QuoteShort},
		{"quote", 0, QuoteMedium},
		{"quote", 0, QuoteLong},
	}
	for _, tc := range cases {
		var result string
		switch tc.mode {
		case "time":
			result = ForMode(g, "time", tc.length, tc.ql)
		case "words":
			result = ForMode(g, "words", tc.length, tc.ql)
		case "quote":
			result = ForMode(g, "quote", tc.length, tc.ql)
		}
		if strings.TrimSpace(result) == "" {
			t.Errorf("ForMode(%s, %d, %d): returned empty string", tc.mode, tc.length, tc.ql)
		}
	}
}

// min is a local helper for older Go compat in test slicing.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
