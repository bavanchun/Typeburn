package mode

import (
	"reflect"
	"testing"
)

var knownModes = []Mode{ModeTime, ModeWords, ModeQuote, ModeCode}

func TestModeStringValues(t *testing.T) {
	tests := map[Mode]string{
		ModeTime:  "time",
		ModeWords: "words",
		ModeQuote: "quote",
		ModeCode:  "code",
	}
	for m, want := range tests {
		if string(m) != want {
			t.Fatalf("%v string = %q, want %q", m, string(m), want)
		}
	}
}

func TestLengthsFor(t *testing.T) {
	tests := []struct {
		mode Mode
		want []int
	}{
		{ModeTime, []int{15, 30, 60, 120}},
		{ModeWords, []int{10, 25, 50, 100}},
		{ModeQuote, nil},
		{ModeCode, nil},
	}
	for _, tt := range tests {
		if got := LengthsFor(tt.mode); !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("LengthsFor(%q) = %v, want %v", tt.mode, got, tt.want)
		}
	}
}

func TestLengthsForHandlesEveryKnownMode(t *testing.T) {
	for _, m := range knownModes {
		lens := LengthsFor(m)
		switch m {
		case ModeQuote, ModeCode:
			if lens != nil {
				t.Fatalf("LengthsFor(%q): want nil, got %v", m, lens)
			}
		default:
			if len(lens) == 0 {
				t.Fatalf("LengthsFor(%q): want options, got %v", m, lens)
			}
		}
	}
}
