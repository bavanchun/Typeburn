package metrics_test

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/mode"
	"github.com/bavanchun/Typeburn/internal/typing"
)

func benchCodeTarget(n int) string {
	var b strings.Builder
	for b.Len() < n {
		b.WriteString("func main() {\n\tprintln(\"typeburn\")\n}\n")
	}
	return b.String()[:n]
}

func benchLog(target string, m mode.Mode, wordTarget int) []typing.Keystroke {
	e := typing.New(target, m, wordTarget)
	for i, r := range target {
		if i%97 == 0 {
			e.Apply('x', int64(i+1))
			e.Backspace(int64(i + 2))
		}
		e.Apply(r, int64(i+3))
	}
	return e.Log()
}

func BenchmarkMetricsComputeCode10k(b *testing.B) {
	log := benchLog(benchCodeTarget(10000), mode.ModeCode, 0)
	endMs := log[len(log)-1].TimeMs + 1000
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = metrics.Compute(log, mode.ModeCode, endMs)
	}
}

func BenchmarkLiveWPMCode10k(b *testing.B) {
	log := benchLog(benchCodeTarget(10000), mode.ModeCode, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = metrics.LiveWPM(log, 120000)
	}
}

func BenchmarkKeyHeatmapCode10k(b *testing.B) {
	log := benchLog(benchCodeTarget(10000), mode.ModeCode, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = metrics.KeyHeatmap(log)
	}
}
