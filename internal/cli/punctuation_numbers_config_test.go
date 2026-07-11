package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

func TestPunctuationNumbersSetGet(t *testing.T) {
	settings := config.Defaults()
	var out bytes.Buffer

	execRoot := func(args ...string) error {
		out.Reset()
		root := NewRoot(
			WithWriters(&out, &bytes.Buffer{}),
			WithHomeRunner(func(context.Context, string, string) error { return nil }),
			WithSettingsStore(
				func() config.Settings { return settings },
				func(s config.Settings) error { settings = s; return nil },
			),
		)
		root.SetArgs(args)
		return root.Execute()
	}

	for _, key := range []string{"punctuation", "numbers"} {
		// Default is false.
		if err := execRoot("config", "get", key); err != nil {
			t.Fatalf("get %s: %v", key, err)
		}
		if strings.TrimSpace(out.String()) != "false" {
			t.Fatalf("default %s: want false, got %q", key, out.String())
		}

		// Set to true and read back.
		if err := execRoot("config", "set", key, "true"); err != nil {
			t.Fatalf("set %s true: %v", key, err)
		}
		if err := execRoot("config", "get", key); err != nil {
			t.Fatalf("get %s after set: %v", key, err)
		}
		if strings.TrimSpace(out.String()) != "true" {
			t.Fatalf("after set: want true, got %q", out.String())
		}

		// Reject invalid values.
		if err := execRoot("config", "set", key, "bogus"); err == nil {
			t.Fatalf("expected error setting invalid %s", key)
		}
	}
}

func TestConfigListIncludesPunctuationAndNumbers(t *testing.T) {
	var out bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithSettingsStore(func() config.Settings { return config.Defaults() }, nil),
	)
	root.SetArgs([]string{"config", "list"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config list: %v", err)
	}
	if !strings.Contains(out.String(), "punctuation") {
		t.Errorf("config list output missing punctuation:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "numbers") {
		t.Errorf("config list output missing numbers:\n%s", out.String())
	}
}
