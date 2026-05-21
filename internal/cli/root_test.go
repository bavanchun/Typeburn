package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func executeRoot(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &errOut),
		WithHomeRunner(func(_ context.Context, _, _ string) error { return nil }),
	)
	root.SetArgs(args)
	err := root.Execute()
	return out.String(), errOut.String(), err
}

func TestRootHelp(t *testing.T) {
	out, _, err := executeRoot(t, "-h")
	if err != nil {
		t.Fatalf("help returned error: %v", err)
	}
	if !strings.Contains(out, "typeburn") || !strings.Contains(out, "version") {
		t.Fatalf("help missing expected content:\n%s", out)
	}
}

func TestRootUnknownArgsFallThrough(t *testing.T) {
	tests := [][]string{
		{"anything-unknown"},
		{"--bogus-root-flag"},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			_, _, err := executeRoot(t, args...)
			if err != nil {
				t.Fatalf("root fall-through returned error: %v", err)
			}
		})
	}
}

func TestRootTextAliasLoadsCodeBeforeTUI(t *testing.T) {
	var gotText, gotHint string
	root := NewRoot(
		WithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		WithCodeLoader(func(path string) (string, error) {
			if path != "snippet.go" {
				t.Fatalf("want snippet.go, got %q", path)
			}
			return "package main", nil
		}),
		WithHomeRunner(func(_ context.Context, text, hint string) error {
			gotText, gotHint = text, hint
			return nil
		}),
	)
	root.SetArgs([]string{"--text", "snippet.go"})
	if err := root.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if gotText != "package main" || gotHint != "" {
		t.Fatalf("unexpected TUI text/hint: %q / %q", gotText, gotHint)
	}
}

func TestVersionCommand(t *testing.T) {
	out, _, err := executeRoot(t, "version")
	if err != nil {
		t.Fatalf("version returned error: %v", err)
	}
	if !strings.Contains(out, "typeburn") {
		t.Fatalf("version output missing banner: %q", out)
	}
}

func TestVersionJSON(t *testing.T) {
	out, _, err := executeRoot(t, "version", "--json")
	if err != nil {
		t.Fatalf("version --json returned error: %v", err)
	}
	if !strings.Contains(out, `"version"`) || !strings.Contains(out, `"go_version"`) {
		t.Fatalf("version JSON missing fields: %q", out)
	}
}

func TestSubcommandStrictness(t *testing.T) {
	_, _, err := executeRoot(t, "version", "--bogus")
	if err == nil {
		t.Fatal("version --bogus must return an error")
	}
}
