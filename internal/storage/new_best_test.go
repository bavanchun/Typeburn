package storage

import "testing"

func TestBestBucketKey(t *testing.T) {
	tests := []struct {
		name   string
		mode   string
		length int
		want   string
	}{
		{name: "time includes length", mode: "time", length: 30, want: "time/30"},
		{name: "words includes length", mode: "words", length: 50, want: "words/50"},
		{name: "quote ignores length", mode: "quote", length: 0, want: "quote"},
		{name: "code ignores length", mode: "code", length: 0, want: "code"},
		// Edge cases
		{name: "time zero length", mode: "time", length: 0, want: "time/0"},
		{name: "time negative length", mode: "time", length: -1, want: "time/-1"},
		{name: "words zero", mode: "words", length: 0, want: "words/0"},
		{name: "code with length", mode: "code", length: 42, want: "code"},
		{name: "unknown mode", mode: "unknown", length: 10, want: "unknown"},
		{name: "empty mode", mode: "", length: 10, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BestBucketKey(tt.mode, tt.length); got != tt.want {
				t.Fatalf("BestBucketKey(%q, %d) = %q, want %q", tt.mode, tt.length, got, tt.want)
			}
		})
	}
}

func TestEffectiveWPM(t *testing.T) {
	if got := EffectiveWPM(Record{WPM: 80}); got != 80 {
		t.Fatalf("legacy EffectiveWPM = %v, want 80", got)
	}
	if got := EffectiveWPM(Record{WPM: 80, NetWPM: 80.42}); got != 80.42 {
		t.Fatalf("net EffectiveWPM = %v, want 80.42", got)
	}
}

func TestBestWPMPerBucket(t *testing.T) {
	rows := []Record{
		{Mode: "time", Length: 30, WPM: 70, NetWPM: 70.1},
		{Mode: "time", Length: 30, WPM: 75, NetWPM: 75.2},
		{Mode: "time", Length: 60, WPM: 90, NetWPM: 90.3},
		{Mode: "words", Length: 50, WPM: 100},
		{Mode: "quote", WPM: 65, NetWPM: 65.4},
	}

	bests := BestWPMPerBucket(rows)
	want := map[string]float64{
		"time/30":  75.2,
		"time/60":  90.3,
		"words/50": 100,
		"quote":    65.4,
	}
	for key, expected := range want {
		if got := bests[key]; got != expected {
			t.Fatalf("bests[%q] = %v, want %v", key, got, expected)
		}
	}
}
