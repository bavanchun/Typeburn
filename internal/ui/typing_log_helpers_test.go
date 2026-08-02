package ui

import (
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// TestTypedFromLog_AgreesWithTheEngineUnderStrict: this file used to carry its
// own copy of the replay loop, so when strict mode began logging keystrokes it
// had refused, the same defect appeared twice — the buffer reconstructed here
// held characters the engine never accepted, and every renderer that asked for
// it drew a run nobody typed.
func TestTypedFromLog_AgreesWithTheEngineUnderStrict(t *testing.T) {
	e := typing.NewStrict("abcdef", config.ModeWords, 1, true)

	ts := int64(1000)
	apply := func(s string) {
		for _, r := range s {
			e.Apply(r, ts)
			ts += 100
		}
	}
	apply("abc")
	apply("zzzzz") // refused at position 3
	for i := 0; i < 3; i++ {
		e.Backspace(ts)
		ts += 100
	}
	apply("abcdef")

	want := string(e.Typed())
	if got := string(typedFromLog(e.Log())); got != want {
		t.Errorf("typedFromLog reconstructed %q, engine holds %q", got, want)
	}
}
