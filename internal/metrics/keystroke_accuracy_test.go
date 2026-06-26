package metrics_test

import (
	"math"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/metrics"
	"github.com/bavanchun/Typeburn/internal/typing"
)

func TestKeystrokeAccuracy(t *testing.T) {
	t.Run("computes accuracy directly from log forward keystrokes", func(t *testing.T) {
		// Log has 4 forward keys, 3 correct, 1 incorrect
		e := typing.NewStrict("hello", config.ModeWords, 1, true)
		e.Apply('h', 100) // Correct
		e.Apply('x', 200) // Incorrect
		e.Apply('e', 300) // Correct
		e.Apply('l', 400) // Correct
		log := e.Log()

		res := metrics.Compute(log, config.ModeWords, 500)
		want := 100.0 * 3.0 / 4.0 // 75.0%
		if math.Abs(res.KeystrokeAccuracy-want) > 0.01 {
			t.Errorf("expected KeystrokeAccuracy %.2f, got %.2f", want, res.KeystrokeAccuracy)
		}
	})

	t.Run("empty log returns 100%", func(t *testing.T) {
		res := metrics.Compute(nil, config.ModeWords, 0)
		if res.KeystrokeAccuracy != 100.0 {
			t.Errorf("expected KeystrokeAccuracy 100.0 for empty log, got %.2f", res.KeystrokeAccuracy)
		}
	})
}
