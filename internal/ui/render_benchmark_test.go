package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/theme"
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

func benchFilledEngine(target string, mode config.Mode, wordTarget int) *typing.Engine {
	e := typing.New(target, mode, wordTarget)
	for i, r := range target {
		e.Apply(r, int64(i+1))
	}
	return e
}

func BenchmarkRenderWordStreamWords100(b *testing.B) {
	target := benchWordsTarget(100)
	e := benchFilledEngine(target, config.ModeWords, 100)
	states := e.States()
	typed := typedFromLog(e.Log())
	targetRunes := []rune(target)
	th := theme.Default()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = RenderWordStream(states, targetRunes, typed, 100, th)
	}
}

func BenchmarkRenderCodeStreamCode10k(b *testing.B) {
	target := benchCodeTarget(10000)
	e := benchFilledEngine(target, config.ModeCode, 0)
	states := e.States()
	typed := typedFromLog(e.Log())
	targetRunes := []rune(target)
	th := theme.Default()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = RenderCodeStream(states, targetRunes, typed, 100, 20, th)
	}
}

func BenchmarkTypingViewCode10k(b *testing.B) {
	target := benchCodeTarget(10000)
	th := theme.Default()
	km := config.DefaultKeymap()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		m := NewTypingCode(target, th, km, false).SetSize(120, 40)
		m.eng = benchFilledEngine(target, config.ModeCode, 0)
		m.startMs = 1
		m.nowMs = 100000
		_ = m.View()
	}
}
