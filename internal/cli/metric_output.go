package cli

import (
	"fmt"

	"github.com/bavanchun/Typeburn/v2/internal/metrics"
)

// keyMissTableRows caps how many most_missed rows the table view shows; the
// JSON output carries the full (top-8) list from metrics.KeyHeatmap.
const keyMissTableRows = 5

type keyMissOutput struct {
	Key      string `json:"key"`
	Misses   int    `json:"misses"`
	Attempts int    `json:"attempts"`
}

type metricOutput struct {
	NetWPM         float64         `json:"net_wpm"`
	RawWPM         float64         `json:"raw_wpm"`
	Accuracy       float64         `json:"accuracy"`
	Consistency    float64         `json:"consistency"`
	CPS            float64         `json:"cps"`
	CorrectChars   int             `json:"correct_chars"`
	IncorrectChars int             `json:"incorrect_chars"`
	ExtraChars     int             `json:"extra_chars"`
	DurationMs     int64           `json:"duration_ms"`
	KeyMisses      []keyMissOutput `json:"key_misses"`
}

func newMetricOutput(r metrics.Result) metricOutput {
	misses := make([]keyMissOutput, 0, len(r.KeyMisses))
	for _, km := range r.KeyMisses {
		misses = append(misses, keyMissOutput{Key: km.Label, Misses: km.Misses, Attempts: km.Attempts})
	}
	return metricOutput{
		NetWPM:         r.NetWPM,
		RawWPM:         r.RawWPM,
		Accuracy:       r.Accuracy,
		Consistency:    r.Consistency,
		CPS:            r.CPS,
		CorrectChars:   r.CorrectChars,
		IncorrectChars: r.IncorrectChars,
		ExtraChars:     r.ExtraChars,
		DurationMs:     r.DurationMs,
		KeyMisses:      misses,
	}
}

func metricTableRows(r metrics.Result) [][]string {
	rows := [][]string{
		{"net_wpm", fmt.Sprintf("%.2f", r.NetWPM)},
		{"raw_wpm", fmt.Sprintf("%.2f", r.RawWPM)},
		{"accuracy", fmt.Sprintf("%.2f%%", r.Accuracy)},
		{"consistency", fmt.Sprintf("%.2f%%", r.Consistency)},
		{"cps", fmt.Sprintf("%.2f", r.CPS)},
		{"duration_ms", fmt.Sprint(r.DurationMs)},
	}
	for i, km := range r.KeyMisses {
		if i >= keyMissTableRows {
			break
		}
		rows = append(rows, []string{"most_missed_" + km.Label, fmt.Sprintf("%d/%d", km.Misses, km.Attempts)})
	}
	return rows
}
