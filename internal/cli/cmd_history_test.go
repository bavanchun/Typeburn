package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/internal/storage"
)

func TestHistoryEmptyJSON(t *testing.T) {
	var out bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithHistoryLoader(func() []storage.Record { return nil }),
	)
	root.SetArgs([]string{"history", "--json"})
	if err := root.Execute(); err != nil {
		t.Fatalf("history --json: %v", err)
	}
	if out.String() != "[]\n" {
		t.Fatalf("want empty JSON array, got %q", out.String())
	}
}

func TestHistoryEmptyTable(t *testing.T) {
	var out bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithHistoryLoader(func() []storage.Record { return nil }),
	)
	root.SetArgs([]string{"history"})
	if err := root.Execute(); err != nil {
		t.Fatalf("history: %v", err)
	}
	if !strings.Contains(out.String(), "no history yet") {
		t.Fatalf("missing empty message: %q", out.String())
	}
}

func TestHistoryNewestFirstAndLimit(t *testing.T) {
	var out bytes.Buffer
	old := storage.Record{Time: time.Date(2026, 5, 19, 1, 0, 0, 0, time.UTC), Mode: "words", Length: 10, WPM: 70}
	newer := storage.Record{Time: time.Date(2026, 5, 20, 1, 0, 0, 0, time.UTC), Mode: "time", Length: 30, WPM: 90}
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithHistoryLoader(func() []storage.Record { return []storage.Record{old, newer} }),
	)
	root.SetArgs([]string{"history", "-n", "1"})
	if err := root.Execute(); err != nil {
		t.Fatalf("history -n 1: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "90") || strings.Contains(got, "70") {
		t.Fatalf("limit/newest ordering failed:\n%s", got)
	}
}
