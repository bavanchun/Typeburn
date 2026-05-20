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
