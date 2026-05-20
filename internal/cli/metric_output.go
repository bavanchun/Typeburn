package cli

import (
	"fmt"

	"github.com/bavanchun/Typeburn/internal/metrics"
)

type metricOutput struct {
	NetWPM         float64 `json:"net_wpm"`
	RawWPM         float64 `json:"raw_wpm"`
	Accuracy       float64 `json:"accuracy"`
	Consistency    float64 `json:"consistency"`
	CPS            float64 `json:"cps"`
	CorrectChars   int     `json:"correct_chars"`
	IncorrectChars int     `json:"incorrect_chars"`
	ExtraChars     int     `json:"extra_chars"`
	DurationMs     int64   `json:"duration_ms"`
}

func newMetricOutput(r metrics.Result) metricOutput {
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
	}
}

func metricTableRows(r metrics.Result) [][]string {
	return [][]string{
		{"net_wpm", fmt.Sprintf("%.2f", r.NetWPM)},
		{"raw_wpm", fmt.Sprintf("%.2f", r.RawWPM)},
		{"accuracy", fmt.Sprintf("%.2f%%", r.Accuracy)},
		{"consistency", fmt.Sprintf("%.2f%%", r.Consistency)},
		{"cps", fmt.Sprintf("%.2f", r.CPS)},
		{"duration_ms", fmt.Sprint(r.DurationMs)},
	}
}
