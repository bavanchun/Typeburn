package typing_test

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/mode"
	"github.com/bavanchun/Typeburn/internal/typing"
)

func benchWordsTarget(n int) string {
	words := make([]string, n)
	for i := range words {
		words[i] = "alpha"
	}
	return strings.Join(words, " ")
}

func benchCodeTarget(n int) string {
	var b strings.Builder
	for b.Len() < n {
		b.WriteString("func main() {\n\tprintln(\"typeburn\")\n}\n")
	}
	return b.String()[:n]
}

func benchFilledEngine(target string, m mode.Mode, wordTarget int) *typing.Engine {
	e := typing.New(target, m, wordTarget)
	for i, r := range target {
		e.Apply(r, int64(i+1))
	}
	return e
}

func BenchmarkEngineApplyWords100(b *testing.B) {
	target := benchWordsTarget(100)
	runes := []rune(target)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := typing.New(target, mode.ModeWords, 100)
		for j, r := range runes {
			e.Apply(r, int64(j+1))
		}
	}
}

func BenchmarkEngineApplyCode10k(b *testing.B) {
	target := benchCodeTarget(10000)
	runes := []rune(target)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		e := typing.New(target, mode.ModeCode, 0)
		for j, r := range runes {
			e.Apply(r, int64(j+1))
		}
	}
}

func BenchmarkEngineStatesCode10k(b *testing.B) {
	e := benchFilledEngine(benchCodeTarget(10000), mode.ModeCode, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = e.States()
	}
}

func BenchmarkEngineLogCode10k(b *testing.B) {
	e := benchFilledEngine(benchCodeTarget(10000), mode.ModeCode, 0)
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = e.Log()
	}
}
