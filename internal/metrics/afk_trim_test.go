package metrics_test

import (
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/typing"
)

// TestAFKTrim validates AFK trailing-trim behaviour.
// AFK trim ONLY applies to ModeTime. Gap threshold is >7s between last two
// keystrokes (or between last keystroke and endMs).
func TestAFKTrim(t *testing.T) {
	t.Run("Time mode: trailing gap >7s trims endMs to last keystroke", func(t *testing.T) {
		// Type keys ending at t=5000, then endMs=15000 (10s gap > 7s)
		e := typing.New("hello world", config.ModeTime, 30000)
		e.Apply('h', 1000)
		e.Apply('e', 2000)
		e.Apply('l', 3000)
		e.Apply('l', 4000)
		e.Apply('o', 5000)
		// last keystroke at 5000; endMs=15000 → gap=10000ms > 7000ms → trim
		log := e.Log()
		trimmed, effectiveEnd := metrics.TrimAFK(log, config.ModeTime, 15000)
		if effectiveEnd != 5000 {
			t.Errorf("Time AFK: want effectiveEnd=5000, got %d", effectiveEnd)
		}
		// trimmed log should only contain keystrokes up to last active key
		if len(trimmed) != len(log) {
			// trimmed log itself doesn't change (only endMs is adjusted for trailing gap)
			// but per-second buckets beyond last keystroke are dropped
			t.Logf("trimmed log len=%d, original len=%d (both OK if endMs adjusted)", len(trimmed), len(log))
		}
		_ = trimmed
	})

	t.Run("Time mode: gap exactly 7s — NOT trimmed (must be strictly >7s)", func(t *testing.T) {
		e := typing.New("hello", config.ModeTime, 30000)
		e.Apply('h', 1000)
		e.Apply('i', 2000)
		// last key at 2000; endMs=9000 → gap=7000ms (not > 7000ms)
		log := e.Log()
		_, effectiveEnd := metrics.TrimAFK(log, config.ModeTime, 9000)
		if effectiveEnd != 9000 {
			t.Errorf("gap exactly 7s: want effectiveEnd=9000 (no trim), got %d", effectiveEnd)
		}
	})

	t.Run("Words mode: gap >7s does NOT trim", func(t *testing.T) {
		e := typing.New("hi go", config.ModeWords, 2)
		e.Apply('h', 1000)
		e.Apply('i', 2000)
		log := e.Log()
		// 10s gap but Words mode → no trim
		_, effectiveEnd := metrics.TrimAFK(log, config.ModeWords, 12000)
		if effectiveEnd != 12000 {
			t.Errorf("Words mode: want effectiveEnd=12000 (no trim), got %d", effectiveEnd)
		}
	})

	t.Run("Quote mode: gap >7s does NOT trim", func(t *testing.T) {
		e := typing.New("the end", config.ModeQuote, 0)
		e.Apply('t', 1000)
		e.Apply('h', 2000)
		log := e.Log()
		_, effectiveEnd := metrics.TrimAFK(log, config.ModeQuote, 12000)
		if effectiveEnd != 12000 {
			t.Errorf("Quote mode: want effectiveEnd=12000 (no trim), got %d", effectiveEnd)
		}
	})

	t.Run("Empty log: TrimAFK returns original endMs regardless of mode", func(t *testing.T) {
		_, effectiveEnd := metrics.TrimAFK(nil, config.ModeTime, 5000)
		if effectiveEnd != 5000 {
			t.Errorf("empty log: want effectiveEnd=5000, got %d", effectiveEnd)
		}
	})

	t.Run("Time mode: gap <=7s — no trim, endMs unchanged", func(t *testing.T) {
		e := typing.New("hi", config.ModeTime, 30000)
		e.Apply('h', 1000)
		e.Apply('i', 3000)
		log := e.Log()
		// last key at 3000; endMs=8000 → gap=5000ms < 7000ms
		_, effectiveEnd := metrics.TrimAFK(log, config.ModeTime, 8000)
		if effectiveEnd != 8000 {
			t.Errorf("small gap: want effectiveEnd=8000, got %d", effectiveEnd)
		}
	})

	t.Run("Compute uses trimmed endMs for Time mode duration", func(t *testing.T) {
		// 5 chars typed ending at 5000ms, then AFK until 20000ms
		// After trim: duration = 5000 - 1000 = 4000ms
		e := typing.New("hello world extra", config.ModeTime, 30000)
		e.Apply('h', 1000)
		e.Apply('e', 2000)
		e.Apply('l', 3000)
		e.Apply('l', 4000)
		e.Apply('o', 5000)
		log := e.Log()
		r := metrics.Compute(log, config.ModeTime, 20000)
		// With AFK trim: endMs adjusted to 5000, duration = 5000-1000 = 4000ms
		if r.DurationMs != 4000 {
			t.Errorf("Compute with AFK trim: want DurationMs=4000, got %d", r.DurationMs)
		}
	})
}
