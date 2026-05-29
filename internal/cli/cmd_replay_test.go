package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
	"github.com/bavanchun/Typeburn/internal/typing"
)

func TestReplayJSON(t *testing.T) {
	var out bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
	)
	root.SetArgs([]string{"replay", "../../testdata/sample-keystroke-log.json", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("replay --json: %v", err)
	}
	if !strings.Contains(out.String(), `"net_wpm"`) || !strings.Contains(out.String(), `"mode": "words"`) {
		t.Fatalf("unexpected replay JSON:\n%s", out.String())
	}
	// key_misses is additive to the CLI contract; the all-correct fixture has none.
	if !strings.Contains(out.String(), `"key_misses"`) {
		t.Fatalf("replay JSON missing key_misses field:\n%s", out.String())
	}
}

func TestReplayErrors(t *testing.T) {
	dir := t.TempDir()
	badJSON := filepath.Join(dir, "bad.json")
	if err := os.WriteFile(badJSON, []byte("{"), 0600); err != nil {
		t.Fatal(err)
	}
	wrongSchema := filepath.Join(dir, "schema.json")
	writeReplayFixture(t, wrongSchema, replayInput{
		SchemaVersion: 2,
		Mode:          config.ModeWords,
		EndMs:         1000,
		Log:           []typing.Keystroke{{TimeMs: 0, Typed: 'a', Target: 'a', Correct: true}},
	})
	tests := []string{"missing.json", badJSON, wrongSchema}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			_, err := loadReplayInput(path)
			if err == nil {
				t.Fatal("expected replay load error")
			}
			if ExitCode(err) != ExitIO {
				t.Fatalf("want IO exit, got %d for %v", ExitCode(err), err)
			}
		})
	}
}

func TestKeystrokeJSONRoundTrip(t *testing.T) {
	want := typing.Keystroke{TimeMs: 1, Typed: 'h', Target: 'h', Correct: true}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got typing.Keystroke
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("round-trip drift: %#v != %#v", got, want)
	}
}

func writeReplayFixture(t *testing.T, path string, input replayInput) {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}
