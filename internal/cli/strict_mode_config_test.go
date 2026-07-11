package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

func TestStrictModeSetGet(t *testing.T) {
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

	// Default is false.
	if err := execRoot("config", "get", "strict_mode"); err != nil {
		t.Fatalf("get strict_mode: %v", err)
	}
	if strings.TrimSpace(out.String()) != "false" {
		t.Fatalf("default strict_mode: want false, got %q", out.String())
	}

	// Set to true and read back.
	if err := execRoot("config", "set", "strict_mode", "true"); err != nil {
		t.Fatalf("set strict_mode true: %v", err)
	}
	if err := execRoot("config", "get", "strict_mode"); err != nil {
		t.Fatalf("get strict_mode after set: %v", err)
	}
	if strings.TrimSpace(out.String()) != "true" {
		t.Fatalf("after set: want true, got %q", out.String())
	}

	// Reject invalid values.
	if err := execRoot("config", "set", "strict_mode", "bogus"); err == nil {
		t.Fatal("expected error setting invalid strict_mode")
	}
}

func TestConfigListIncludesStrictMode(t *testing.T) {
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
	if !strings.Contains(out.String(), "strict_mode") {
		t.Errorf("config list output missing strict_mode:\n%s", out.String())
	}
}
