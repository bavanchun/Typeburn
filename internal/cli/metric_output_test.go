package cli

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/metrics"
)

func sampleResult() metrics.Result {
	return metrics.Result{
		NetWPM: 80, RawWPM: 90, Accuracy: 95, Consistency: 88, CPS: 7.5,
		CorrectChars: 100, IncorrectChars: 5, ExtraChars: 1, DurationMs: 30000,
		KeyMisses: []metrics.KeyMiss{
			{Key: 'e', Label: "e", Misses: 4, Attempts: 31},
			{Key: 't', Label: "t", Misses: 3, Attempts: 20},
		},
	}
}

// TestNewMetricOutput_MapsKeyMisses checks key_misses mapping and that existing
// fields are preserved (CLI contract guard).
func TestNewMetricOutput_MapsKeyMisses(t *testing.T) {
	out := newMetricOutput(sampleResult())
	if out.NetWPM != 80 || out.DurationMs != 30000 || out.CorrectChars != 100 {
		t.Fatalf("existing metric fields changed: %#v", out)
	}
	if len(out.KeyMisses) != 2 {
		t.Fatalf("want 2 key misses, got %d", len(out.KeyMisses))
	}
	if out.KeyMisses[0] != (keyMissOutput{Key: "e", Misses: 4, Attempts: 31}) {
		t.Errorf("top key miss mismatch: %#v", out.KeyMisses[0])
	}
}

// TestNewMetricOutput_EmptyKeyMissesIsArray checks an empty heatmap serializes
// as [] (stable contract), not null.
func TestNewMetricOutput_EmptyKeyMissesIsArray(t *testing.T) {
	out := newMetricOutput(metrics.Result{Accuracy: 100})
	data, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"key_misses":[]`) {
		t.Errorf("empty heatmap should marshal as []:\n%s", data)
	}
}

// TestNewMetricOutput_JSONContainsKeyMisses checks the JSON key + entry shape.
func TestNewMetricOutput_JSONContainsKeyMisses(t *testing.T) {
	data, err := json.Marshal(newMetricOutput(sampleResult()))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	for _, want := range []string{`"key_misses"`, `"key":"e"`, `"misses":4`, `"attempts":31`, `"net_wpm"`} {
		if !strings.Contains(s, want) {
			t.Errorf("JSON missing %q:\n%s", want, s)
		}
	}
}

// TestMetricTableRows_AppendsMissRows checks top-N most_missed rows are appended
// after the existing rows.
func TestMetricTableRows_AppendsMissRows(t *testing.T) {
	rows := metricTableRows(sampleResult())
	var foundE, foundT, foundDur bool
	for _, r := range rows {
		switch r[0] {
		case "most_missed_e":
			foundE = r[1] == "4/31"
		case "most_missed_t":
			foundT = r[1] == "3/20"
		case "duration_ms":
			foundDur = true
		}
	}
	if !foundDur {
		t.Error("existing duration_ms row missing")
	}
	if !foundE || !foundT {
		t.Errorf("expected most_missed rows for e and t, got rows: %#v", rows)
	}
}

// TestMetricTableRows_CapsAtFive checks no more than 5 most_missed rows appear.
func TestMetricTableRows_CapsAtFive(t *testing.T) {
	r := sampleResult()
	r.KeyMisses = nil
	for _, lbl := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		r.KeyMisses = append(r.KeyMisses, metrics.KeyMiss{Key: rune(lbl[0]), Label: lbl, Misses: 2, Attempts: 10})
	}
	rows := metricTableRows(r)
	count := 0
	for _, row := range rows {
		if strings.HasPrefix(row[0], "most_missed_") {
			count++
		}
	}
	if count != 5 {
		t.Errorf("want 5 most_missed rows (capped), got %d", count)
	}
}

// TestMetricTableRows_NoMisses checks clean runs add no most_missed rows.
func TestMetricTableRows_NoMisses(t *testing.T) {
	rows := metricTableRows(metrics.Result{Accuracy: 100})
	for _, row := range rows {
		if strings.HasPrefix(row[0], "most_missed_") {
			t.Errorf("clean run should add no most_missed rows, got %v", row)
		}
	}
}
