package cli

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/internal/config"
)

func TestConfigGetSetList(t *testing.T) {
	settings := config.Defaults()
	var saved config.Settings
	var out bytes.Buffer
	root := NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithSettingsStore(
			func() config.Settings { return settings },
			func(s config.Settings) error { saved = s; settings = s; return nil },
		),
	)
	root.SetArgs([]string{"config", "set", "theme", "nord"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config set: %v", err)
	}
	if saved.Theme != "nord" {
		t.Fatalf("theme not saved: %#v", saved)
	}

	out.Reset()
	root = NewRoot(
		WithWriters(&out, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithSettingsStore(func() config.Settings { return settings }, nil),
	)
	root.SetArgs([]string{"config", "get", "theme"})
	if err := root.Execute(); err != nil {
		t.Fatalf("config get: %v", err)
	}
	if strings.TrimSpace(out.String()) != "nord" {
		t.Fatalf("unexpected get output: %q", out.String())
	}
}

func TestConfigSetInvalidDoesNotSave(t *testing.T) {
	saves := 0
	root := NewRoot(
		WithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithSettingsStore(
			func() config.Settings { return config.Defaults() },
			func(config.Settings) error { saves++; return nil },
		),
	)
	root.SetArgs([]string{"config", "set", "theme", "zzz"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected invalid theme error")
	}
	if saves != 0 {
		t.Fatalf("invalid config wrote settings %d times", saves)
	}
	if !strings.Contains(err.Error(), "valid:") {
		t.Fatalf("error should list valid values: %v", err)
	}
}

func TestConfigRejectsInvalidValues(t *testing.T) {
	tests := [][]string{
		{"default_mode", "bad"},
		{"default_length", "999"},
		{"blink_cursor", "maybe"},
		{"update_check", "maybe"},
		{"unknown", "value"},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt, "="), func(t *testing.T) {
			err := configSet(&config.Settings{Theme: "default", DefaultMode: config.ModeTime, DefaultLength: 30}, tt[0], tt[1])
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestUpdateCheckSetGet(t *testing.T) {
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
	if err := execRoot("config", "get", "update_check"); err != nil {
		t.Fatalf("get update_check: %v", err)
	}
	if strings.TrimSpace(out.String()) != "false" {
		t.Fatalf("default update_check: want false, got %q", out.String())
	}

	// Set to true and read back.
	if err := execRoot("config", "set", "update_check", "true"); err != nil {
		t.Fatalf("set update_check true: %v", err)
	}
	if err := execRoot("config", "get", "update_check"); err != nil {
		t.Fatalf("get update_check after set: %v", err)
	}
	if strings.TrimSpace(out.String()) != "true" {
		t.Fatalf("after set: want true, got %q", out.String())
	}

	// Test parseBool directly for all accepted forms.
	for _, trueVal := range []string{"true", "1", "on", "yes", "TRUE", "ON", " yes "} {
		v, ok := parseBool(trueVal)
		if !ok || !v {
			t.Errorf("parseBool(%q) = (%v, %v), want (true, true)", trueVal, v, ok)
		}
	}
	for _, falseVal := range []string{"false", "0", "off", "no", "FALSE", "OFF", " no "} {
		v, ok := parseBool(falseVal)
		if !ok || v {
			t.Errorf("parseBool(%q) = (%v, %v), want (false, true)", falseVal, v, ok)
		}
	}
	for _, badVal := range []string{"maybe", "yes1", "2", ""} {
		_, ok := parseBool(badVal)
		if ok {
			t.Errorf("parseBool(%q) should reject", badVal)
		}
	}
}

func TestConfigListIncludesUpdateCheck(t *testing.T) {
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
	if !strings.Contains(out.String(), "update_check") {
		t.Errorf("config list output missing update_check:\n%s", out.String())
	}
}
