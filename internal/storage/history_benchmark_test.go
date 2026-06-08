package storage

import (
	"testing"
	"time"
)

func benchRecord(i int) Record {
	mode := "time"
	length := 30
	if i%2 == 0 {
		mode = "words"
		length = 50
	}
	return Record{
		Time:        time.Unix(int64(i), 0).UTC(),
		Mode:        mode,
		Length:      length,
		WPM:         60 + i%80,
		NetWPM:      float64(60+i%80) + 0.25,
		RawWPM:      float64(70 + i%90),
		Accuracy:    95,
		Consistency: 85,
	}
}

func BenchmarkIsNewBestHistory200(b *testing.B) {
	hist := make([]Record, 200)
	for i := range hist {
		hist[i] = benchRecord(i)
	}
	candidate := Record{Mode: "time", Length: 30, WPM: 150, NetWPM: 150.5}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = IsNewBest(hist, candidate)
	}
}
