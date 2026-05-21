package runner

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/words"
)

func TestNewSession_DeterministicTargets(t *testing.T) {
	tests := []struct {
		name   string
		mode   config.Mode
		length int
		ql     words.QuoteLen
		seed   int64
		want   string
	}{
		{
			name:   "words",
			mode:   config.ModeWords,
			length: 5,
			seed:   42,
			want:   words.ForMode(words.NewGenerator(42), config.ModeWords, 5, words.QuoteMedium),
		},
		{
			name: "quote",
			mode: config.ModeQuote,
			ql:   words.QuoteShort,
			seed: 7,
			want: words.ForMode(words.NewGenerator(7), config.ModeQuote, 0, words.QuoteShort),
		},
		{
			name:   "time",
			mode:   config.ModeTime,
			length: 15,
			seed:   99,
			want:   words.ForMode(words.NewGenerator(99), config.ModeTime, 15, words.QuoteMedium),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSession(tt.mode, tt.length, tt.ql, tt.seed)
			if got.Target != tt.want {
				t.Fatalf("target drift:\nwant %q\ngot  %q", tt.want, got.Target)
			}
			if got.Mode != tt.mode || got.Length != tt.length || got.QuoteLen != tt.ql {
				t.Fatalf("metadata drift: %#v", got)
			}
		})
	}
}

func TestNewCodeSession(t *testing.T) {
	s := NewCodeSession("fmt.Println(\"hi\")")
	if s.Mode != config.ModeCode {
		t.Fatalf("want code mode, got %q", s.Mode)
	}
	if s.Target != s.CodeText {
		t.Fatalf("target and code text diverged: %q != %q", s.Target, s.CodeText)
	}
	for _, r := range s.Target {
		s.Engine.Apply(r, 1)
	}
	if !s.Engine.Complete(1) {
		t.Fatal("code session should complete after exact snippet")
	}
}

func TestRebuildEngine_TimeUsesMilliseconds(t *testing.T) {
	eng := RebuildEngine(strings.Repeat("word ", 20), config.ModeTime, 30)
	if eng.Complete(29999) {
		t.Fatal("time engine completed before millisecond limit")
	}
	if !eng.Complete(30000) {
		t.Fatal("time engine did not complete at millisecond limit")
	}
}

func TestRebuildEngine_WordsUsesWordCount(t *testing.T) {
	eng := RebuildEngine("one two", config.ModeWords, 2)
	for i, r := range "one two" {
		eng.Apply(r, int64(i+1))
	}
	if !eng.Complete(8) {
		t.Fatal("words engine did not complete after configured word count")
	}
}
