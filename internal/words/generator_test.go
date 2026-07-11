package words

import (
	"regexp"
	"strings"
	"testing"
)

func TestWords_ExactCount(t *testing.T) {
	g := NewGenerator(42)
	cases := []int{10, 25, 50, 100}
	for _, n := range cases {
		got := g.Words(n)
		tokens := strings.Fields(got)
		if len(tokens) != n {
			t.Errorf("Words(%d): got %d tokens, want %d", n, len(tokens), n)
		}
	}
}

func TestWords_ZeroReturnsEmpty(t *testing.T) {
	g := NewGenerator(1)
	if got := g.Words(0); got != "" {
		t.Errorf("Words(0): want empty string, got %q", got)
	}
}

func TestWords_AllFromWordList(t *testing.T) {
	g := NewGenerator(99)
	got := g.Words(50)
	// Build a set for fast lookup.
	set := make(map[string]bool, len(wordList))
	for _, w := range wordList {
		set[w] = true
	}
	for _, tok := range strings.Fields(got) {
		if !set[tok] {
			t.Errorf("Words produced %q which is not in the embedded word list", tok)
		}
	}
}

func TestWords_Deterministic(t *testing.T) {
	a := NewGenerator(7)
	b := NewGenerator(7)
	if a.Words(25) != b.Words(25) {
		t.Error("Words(25): same seed produced different output")
	}
}

func TestWords_DifferentSeeds(t *testing.T) {
	a := NewGenerator(1)
	b := NewGenerator(2)
	// Overwhelmingly unlikely to collide across 25 words.
	if a.Words(25) == b.Words(25) {
		t.Error("Words(25): different seeds produced identical output (collision?)")
	}
}

func TestTimeBuffer_NonEmpty(t *testing.T) {
	g := NewGenerator(3)
	buf := g.TimeBuffer()
	if buf == "" {
		t.Fatal("TimeBuffer: returned empty string")
	}
}

func TestTimeBuffer_SufficientLength(t *testing.T) {
	g := NewGenerator(5)
	buf := g.TimeBuffer()
	tokens := strings.Fields(buf)
	// Must have at least timeBufferWords tokens.
	if len(tokens) < timeBufferWords {
		t.Errorf("TimeBuffer: got %d words, want at least %d", len(tokens), timeBufferWords)
	}
}

func TestTimeBuffer_Deterministic(t *testing.T) {
	a := NewGenerator(11)
	b := NewGenerator(11)
	if a.TimeBuffer() != b.TimeBuffer() {
		t.Error("TimeBuffer: same seed produced different output")
	}
}

func TestWordList_NonEmpty(t *testing.T) {
	if len(wordList) == 0 {
		t.Fatal("embedded word list is empty — check data/english-1000.txt embed path")
	}
	const minWords = 200
	if len(wordList) < minWords {
		t.Errorf("word list has %d entries, want at least %d", len(wordList), minWords)
	}
}

func TestApplyOptions_NoOp(t *testing.T) {
	g := NewGenerator(42)
	text := g.Words(50)
	got := g.ApplyOptions(text, false, false)
	if got != text {
		t.Errorf("ApplyOptions with both flags false must be a no-op:\ngot:  %q\nwant: %q", got, text)
	}
}

func TestApplyOptions_PunctuationDeterministic(t *testing.T) {
	a := NewGenerator(7)
	textA := a.Words(50)
	gotA := a.ApplyOptions(textA, true, false)

	b := NewGenerator(7)
	textB := b.Words(50)
	gotB := b.ApplyOptions(textB, true, false)

	if gotA != gotB {
		t.Error("ApplyOptions(punctuation=true): same seed produced different output")
	}
}

func TestApplyOptions_PunctuationAddsMarks(t *testing.T) {
	g := NewGenerator(1)
	text := g.Words(80)
	got := g.ApplyOptions(text, true, false)

	found := false
	for _, r := range got {
		if r == ',' || r == '.' || r == ';' {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ApplyOptions(punctuation=true) on 80 words produced no punctuation marks: %q", got)
	}
}

func TestApplyOptions_PunctuationCapitalizesAfterPeriod(t *testing.T) {
	g := NewGenerator(3)
	text := g.Words(80)
	got := g.ApplyOptions(text, true, false)

	tokens := strings.Fields(got)
	found := false
	for i := 0; i < len(tokens)-1; i++ {
		if strings.HasSuffix(tokens[i], ".") {
			next := tokens[i+1]
			r := []rune(next)[0]
			if r >= 'A' && r <= 'Z' {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected at least one word following a '.' to be capitalized: %q", got)
	}
}

func TestApplyOptions_PunctuationCapitalizesAfterSemicolon(t *testing.T) {
	g := NewGenerator(6)
	text := g.Words(200)
	got := g.ApplyOptions(text, true, false)

	tokens := strings.Fields(got)
	found := false
	for i := 0; i < len(tokens)-1; i++ {
		if strings.HasSuffix(tokens[i], ";") {
			next := tokens[i+1]
			r := []rune(next)[0]
			if r >= 'A' && r <= 'Z' {
				found = true
				break
			}
		}
	}
	if !found {
		t.Errorf("expected at least one word following a ';' to be capitalized: %q", got)
	}
}

func TestApplyOptions_PunctuationWrapsSomeTokensInQuotes(t *testing.T) {
	g := NewGenerator(4)
	text := g.Words(300)
	got := g.ApplyOptions(text, true, false)

	found := strings.Contains(got, `"`)
	if !found {
		t.Errorf("ApplyOptions(punctuation=true) on 300 words produced no quote-wrapped token: %q", got)
	}
}

func TestApplyOptions_NumbersProduceDigits(t *testing.T) {
	g := NewGenerator(2)
	text := g.Words(80)
	got := g.ApplyOptions(text, false, true)

	digitToken := regexp.MustCompile(`^\d+[,.;]?$`)
	found := false
	for _, tok := range strings.Fields(got) {
		if digitToken.MatchString(tok) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("ApplyOptions(numbers=true) on 80 words produced no numeric token: %q", got)
	}
}

func TestApplyOptions_TokenCountPreserved(t *testing.T) {
	cases := []struct {
		punctuation, numbers bool
	}{
		{false, false},
		{true, false},
		{false, true},
		{true, true},
	}
	for _, c := range cases {
		g := NewGenerator(9)
		text := g.Words(60)
		wantCount := len(strings.Fields(text))
		got := g.ApplyOptions(text, c.punctuation, c.numbers)
		gotCount := len(strings.Fields(got))
		if gotCount != wantCount {
			t.Errorf("punctuation=%v numbers=%v: token count changed: got %d, want %d",
				c.punctuation, c.numbers, gotCount, wantCount)
		}
	}
}

func TestApplyOptions_DeterministicAcrossOptions(t *testing.T) {
	a := NewGenerator(15)
	textA := a.Words(60)
	gotA := a.ApplyOptions(textA, true, true)

	b := NewGenerator(15)
	textB := b.Words(60)
	gotB := b.ApplyOptions(textB, true, true)

	if gotA != gotB {
		t.Error("ApplyOptions(punctuation=true, numbers=true): same seed produced different output")
	}
}
