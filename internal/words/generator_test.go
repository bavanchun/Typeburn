package words

import (
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
