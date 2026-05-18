package ui

import "github.com/bavanchun/Typeburn/internal/metrics"

// makeTestMetricsResult returns a sample metrics.Result suitable for use
// across multiple screen test files in this package.
func makeTestMetricsResult() metrics.Result {
	return metrics.Result{
		NetWPM:         94,
		RawWPM:         108,
		Accuracy:       97,
		Consistency:    95,
		CorrectChars:   142,
		IncorrectChars: 4,
		ExtraChars:     1,
		Errors:         4,
		DurationMs:     30000,
		PerSecond: []metrics.PerSecond{
			{Sec: 0, RawWPM: 60},
			{Sec: 1, RawWPM: 84},
			{Sec: 2, RawWPM: 96},
			{Sec: 3, RawWPM: 108},
			{Sec: 4, RawWPM: 120},
		},
	}
}
