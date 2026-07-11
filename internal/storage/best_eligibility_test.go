package storage

import "testing"

func TestEligibleForBest(t *testing.T) {
	tests := []struct {
		name string
		rec  Record
		want bool
	}{
		{name: "normal time", rec: Record{Mode: "time"}, want: true},
		{name: "strict time", rec: Record{Mode: "time", Strict: true}, want: false},
		{name: "code", rec: Record{Mode: "code"}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := EligibleForBest(tt.rec); got != tt.want {
				t.Fatalf("EligibleForBest(%+v) = %t, want %t", tt.rec, got, tt.want)
			}
		})
	}
}

func TestBestWPMPerBucket_IgnoresIneligibleRuns(t *testing.T) {
	rows := []Record{
		{Mode: "time", Length: 30, WPM: 80, NetWPM: 80},
		{Mode: "time", Length: 30, WPM: 120, NetWPM: 120, Strict: true},
		{Mode: "code", Length: 142, WPM: 140, NetWPM: 140},
	}

	bests := BestWPMPerBucket(rows)
	if got := bests["time/30"]; got != 80 {
		t.Fatalf("time best = %v, want eligible run 80", got)
	}
	if _, ok := bests["code"]; ok {
		t.Fatalf("code must not have a best bucket: %#v", bests)
	}
}

func TestIsNewBest_IgnoresIneligibleHistory(t *testing.T) {
	hist := []Record{
		{Mode: "time", Length: 30, WPM: 120, NetWPM: 120, Strict: true},
		{Mode: "code", Length: 142, WPM: 140, NetWPM: 140},
	}
	candidate := Record{Mode: "time", Length: 30, WPM: 80, NetWPM: 80}
	if !IsNewBest(hist, candidate) {
		t.Fatal("eligible candidate must ignore strict and code history")
	}
}
