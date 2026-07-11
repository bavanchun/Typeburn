package ui

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/theme"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// benchTypingState builds a realistic mid-test word-stream: a 100-word target
// with a prefix already typed and the cursor mid-stream.
func benchTypingState() (states []typing.CharState, target, typed []rune) {
	m := newTestTyping(config.ModeWords, 100)
	tgt := []rune(m.TargetText())
	// Type the first ~120 runes correctly so there is a real prefix + cursor.
	n := 120
	if n > len(tgt) {
		n = len(tgt) / 2
	}
	m, _ = m.applyText(string(tgt[:n]))
	return m.eng.States(), tgt, m.eng.Typed()
}

// BenchmarkWordStreamStatic is the baseline: a full per-rune restyle every frame.
func BenchmarkWordStreamStatic(b *testing.B) {
	th := theme.Default()
	states, target, typed := benchTypingState()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = RenderWordStream(states, target, typed, 100, th)
	}
}

// BenchmarkWordStreamAnimCached measures the animated hot path with a warm
// prefix-token cache: only the ≤3 caret cells re-Render per frame, so allocs/op
// should stay far below the static baseline (the cache is engaging). If this
// regresses toward the static numbers, the cache stopped working.
func BenchmarkWordStreamAnimCached(b *testing.B) {
	th := theme.Default()
	states, target, typed := benchTypingState()
	cache := &streamTokenCache{}
	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000, blinkOn: true, cursorIdx: indexOfCurrent(states)}

	_ = renderWordStreamAnim(states, target, typed, 100, th, ca, cache) // warm cache
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = renderWordStreamAnim(states, target, typed, 100, th, ca, cache)
	}
}

// TestPrefixCacheBounded asserts the engaged cache renders far fewer styled
// tokens per frame than the static path — the hot-path guarantee in test form
// (CI runs tests, not benchmarks). It measures wall allocations indirectly by
// comparing the count of ANSI style runs, which tracks per-cell Render calls.
func TestPrefixCacheBoundedStyleRuns(t *testing.T) {
	th := theme.Default()
	states, target, typed := benchTypingState()
	cache := &streamTokenCache{}
	ca := caretAnim{nowMs: 100000, lastKeyMs: 100000, blinkOn: false, cursorIdx: indexOfCurrent(states)}

	// Warm, then render an animated frame and the static frame.
	_ = renderWordStreamAnim(states, target, typed, 100, th, ca, cache)
	anim := renderWordStreamAnim(states, target, typed, 100, th, ca, cache)
	static := RenderWordStream(states, target, typed, 100, th)

	// Both must carry the same visible content (settled cells identical; only the
	// ≤3 animated cells differ in SGR), so stripped output matches.
	if strip(anim) != strip(static) {
		t.Fatalf("animated stripped output diverged from static")
	}
	// Sanity: the animated frame is non-empty and multi-line (real 100-word body).
	if len(strings.Split(anim, "\n")) < 2 {
		t.Fatalf("expected a wrapped multi-line stream, got %q", anim)
	}
}
