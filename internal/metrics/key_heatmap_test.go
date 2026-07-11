package metrics_test

import (
	"reflect"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/metrics"
	"github.com/bavanchun/Typeburn/v2/internal/typing"
)

// ks is a terse Keystroke constructor for table-driven heatmap tests.
func ks(typed, target rune, correct bool) typing.Keystroke {
	return typing.Keystroke{Typed: typed, Target: target, Correct: correct}
}

// TestKeyHeatmap_EmptyLog returns nil for an empty log.
func TestKeyHeatmap_EmptyLog(t *testing.T) {
	if got := metrics.KeyHeatmap(nil); got != nil {
		t.Errorf("empty log: want nil, got %#v", got)
	}
}

// TestKeyHeatmap_OnlyCorrect returns nil when there are no misses.
func TestKeyHeatmap_OnlyCorrect(t *testing.T) {
	log := []typing.Keystroke{
		ks('h', 'h', true),
		ks('i', 'i', true),
	}
	if got := metrics.KeyHeatmap(log); got != nil {
		t.Errorf("only-correct log: want nil, got %#v", got)
	}
}

// TestKeyHeatmap_CorrectedFumbleCounts confirms a wrong keystroke that is later
// fixed via backspace still counts as one miss; the backspace itself is ignored.
func TestKeyHeatmap_CorrectedFumbleCounts(t *testing.T) {
	log := []typing.Keystroke{
		ks('x', 'e', false), // fumble
		ks(0, 'e', false),   // backspace (excluded)
		ks('e', 'e', true),  // correction
	}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{{Key: 'e', Label: "e", Misses: 1, Attempts: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("corrected fumble:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_CaseFolding merges misses on 'a' and 'A' into one key 'a'.
func TestKeyHeatmap_CaseFolding(t *testing.T) {
	log := []typing.Keystroke{
		ks('x', 'a', false),
		ks('y', 'A', false),
	}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{{Key: 'a', Label: "a", Misses: 2, Attempts: 2}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("case folding:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_ExcludesExtrasAndBackspaces ignores extra chars (Target==0)
// and backspace markers (Typed==0).
func TestKeyHeatmap_ExcludesExtrasAndBackspaces(t *testing.T) {
	log := []typing.Keystroke{
		ks('z', 0, false),   // extra char past target end (excluded)
		ks(0, 't', false),   // backspace (excluded)
		ks('q', 't', false), // real miss on 't'
	}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{{Key: 't', Label: "t", Misses: 1, Attempts: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("excludes extras/backspaces:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_SpaceLabel maps a missed space to the ␣ glyph.
func TestKeyHeatmap_SpaceLabel(t *testing.T) {
	log := []typing.Keystroke{ks('x', ' ', false)}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{{Key: ' ', Label: "␣", Misses: 1, Attempts: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("space label:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_SortDeterminism checks misses desc → attempts desc → key asc.
func TestKeyHeatmap_SortDeterminism(t *testing.T) {
	// key 'a': 2 misses, 2 attempts
	// key 'b': 2 misses, 3 attempts  → before 'a' (attempts desc)
	// key 'c': 2 misses, 2 attempts  → after 'a'  (key asc tiebreak)
	// key 'd': 3 misses, 1 attempt   → first      (misses desc)
	log := []typing.Keystroke{
		ks('x', 'a', false), ks('y', 'a', false),
		ks('x', 'b', false), ks('y', 'b', false), ks('b', 'b', true),
		ks('x', 'c', false), ks('y', 'c', false),
		ks('x', 'd', false), ks('y', 'd', false), ks('z', 'd', false),
	}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{
		{Key: 'd', Label: "d", Misses: 3, Attempts: 3},
		{Key: 'b', Label: "b", Misses: 2, Attempts: 3},
		{Key: 'a', Label: "a", Misses: 2, Attempts: 2},
		{Key: 'c', Label: "c", Misses: 2, Attempts: 2},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sort determinism:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_AttemptsCountAllForward verifies attempts tally correct +
// wrong forward keystrokes against a target, while misses count only wrong.
func TestKeyHeatmap_AttemptsCountAllForward(t *testing.T) {
	log := []typing.Keystroke{
		ks('e', 'e', true),  // correct attempt
		ks('x', 'e', false), // wrong attempt (miss)
		ks('e', 'e', true),  // correct attempt
	}
	got := metrics.KeyHeatmap(log)
	want := []metrics.KeyMiss{{Key: 'e', Label: "e", Misses: 1, Attempts: 3}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("attempts count:\n got  %#v\n want %#v", got, want)
	}
}

// TestKeyHeatmap_CapsAtTopN confirms the result is bounded to the top 8 keys.
func TestKeyHeatmap_CapsAtTopN(t *testing.T) {
	// 12 distinct keys each with one miss → only the top 8 should survive.
	var log []typing.Keystroke
	for _, target := range "abcdefghijkl" {
		log = append(log, ks('x', target, false))
	}
	got := metrics.KeyHeatmap(log)
	if len(got) != 8 {
		t.Errorf("want top-8 cap, got %d entries: %#v", len(got), got)
	}
	// Tie on misses/attempts → key asc, so the survivors are a..h.
	for i, want := range "abcdefgh" {
		if got[i].Key != want {
			t.Errorf("entry %d: want key %q, got %q", i, want, got[i].Key)
		}
	}
}

// TestCompute_PopulatesKeyMisses confirms Compute fills Result.KeyMisses, and
// that early-return paths leave it nil.
func TestCompute_PopulatesKeyMisses(t *testing.T) {
	t.Run("populated from log", func(t *testing.T) {
		log := buildLogTimed("hello world", config.ModeWords, 2, "hellx world", 0, 60000)
		r := metrics.Compute(log, config.ModeWords, 60000)
		if len(r.KeyMisses) == 0 {
			t.Fatalf("expected KeyMisses populated, got empty")
		}
		// 'o' typed as 'x' → one miss on 'o'.
		found := false
		for _, km := range r.KeyMisses {
			if km.Label == "o" && km.Misses == 1 {
				found = true
			}
		}
		if !found {
			t.Errorf("expected a miss on 'o', got %#v", r.KeyMisses)
		}
	})

	t.Run("empty log leaves KeyMisses nil", func(t *testing.T) {
		r := metrics.Compute(nil, config.ModeWords, 0)
		if r.KeyMisses != nil {
			t.Errorf("empty log: want nil KeyMisses, got %#v", r.KeyMisses)
		}
	})
}
