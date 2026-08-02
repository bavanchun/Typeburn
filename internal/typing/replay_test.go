package typing_test

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/mode"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// typeFrom types s into e one rune per 100ms, starting at startMs.
func typeFrom(e *typing.Engine, s string, startMs int64) int64 {
	ts := startMs
	for _, r := range s {
		e.Apply(r, ts)
		ts += 100
	}
	return ts
}

// TestReplayFromLog_MatchesEngineBuffer is the property every consumer of a
// keystroke log depends on: replaying the log reconstructs exactly the buffer
// the engine that wrote it holds. Strict mode broke it by logging keys it had
// refused, after which a backspace popped a slot the engine never had and every
// later position described a run nobody typed.
func TestReplayFromLog_MatchesEngineBuffer(t *testing.T) {
	for _, tc := range []struct {
		name   string
		strict bool
		run    func(e *typing.Engine)
	}{
		{"plain typing", false, func(e *typing.Engine) { typeFrom(e, "abc", 1000) }},
		{"typing with corrections", false, func(e *typing.Engine) {
			ts := typeFrom(e, "abx", 1000)
			e.Backspace(ts)
			e.Apply('c', ts+100)
		}},
		{"strict: wrong keys then retype", true, func(e *typing.Engine) {
			ts := typeFrom(e, "abc", 1000)
			ts = typeFrom(e, "zzzzz", ts) // all refused at position 3
			for i := 0; i < 3; i++ {
				e.Backspace(ts)
				ts += 100
			}
			typeFrom(e, "abcdef", ts)
		}},
		{"strict: wrong key on the first position", true, func(e *typing.Engine) {
			typeFrom(e, "qqa", 1000)
		}},
		{"strict: keys past the end of the target", true, func(e *typing.Engine) {
			typeFrom(e, "abcdefzzz", 1000)
		}},
		{"backspace on an empty buffer", false, func(e *typing.Engine) {
			e.Backspace(1000)
			typeFrom(e, "ab", 1100)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			e := typing.NewStrict("abcdef", mode.ModeWords, 1, tc.strict)
			tc.run(e)

			want := string(e.Typed())
			if got := string(typing.TypedFromLog(e.Log())); got != want {
				t.Errorf("replayed buffer %q, engine holds %q", got, want)
			}
		})
	}
}

// TestReplayBuffer_CarriesTargetAndCorrectness checks the slot detail metrics
// counts from, not just the runes.
func TestReplayBuffer_CarriesTargetAndCorrectness(t *testing.T) {
	e := typing.New("ab", mode.ModeWords, 1)
	typeFrom(e, "aX", 1000)

	buf := typing.ReplayBuffer(e.Log())
	if len(buf) != 2 {
		t.Fatalf("want 2 slots, got %d", len(buf))
	}
	if !buf[0].Correct || buf[0].Target != 'a' {
		t.Errorf("slot 0: want correct 'a', got %+v", buf[0])
	}
	if buf[1].Correct || buf[1].Target != 'b' || buf[1].Typed != 'X' {
		t.Errorf("slot 1: want wrong 'X' against 'b', got %+v", buf[1])
	}
}

// TestKeystrokeJSON_LogWrittenBeforeBlockedExisted decodes a log recorded
// before the Blocked field existed. Absent means false, which is the non-strict
// meaning every such log was written with, so replay must behave as it did.
func TestKeystrokeJSON_LogWrittenBeforeBlockedExisted(t *testing.T) {
	data, err := os.ReadFile("testdata/log_without_blocked_field.json")
	if err != nil {
		t.Fatal(err)
	}

	var log []typing.Keystroke
	if err := json.Unmarshal(data, &log); err != nil {
		t.Fatalf("a log written before the field existed must still parse: %v", err)
	}
	for i, k := range log {
		if k.Blocked {
			t.Errorf("keystroke %d decoded as blocked; an absent field must mean not blocked", i)
		}
	}

	// "abx" then backspace then "c" — the buffer this log has always described.
	if got := string(typing.TypedFromLog(log)); got != "abc" {
		t.Errorf("replayed %q, want %q", got, "abc")
	}
}

// TestKeystrokeJSON_BlockedOmittedWhenFalse keeps old readers able to read new
// logs: an ordinary keystroke must serialise exactly as it did before.
func TestKeystrokeJSON_BlockedOmittedWhenFalse(t *testing.T) {
	out, err := json.Marshal(typing.Keystroke{TimeMs: 1000, Typed: 'a', Target: 'a', Correct: true})
	if err != nil {
		t.Fatal(err)
	}
	want := `{"time_ms":1000,"typed":97,"target":97,"correct":true}`
	if string(out) != want {
		t.Errorf("marshalled %s, want %s", out, want)
	}
}
