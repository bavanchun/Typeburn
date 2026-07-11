package cli

import (
	"bytes"
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/bavanchun/Typeburn/v2/internal/config"
)

func testRunCmd(flags runFlags, changed ...string) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Int("duration", 0, "")
	cmd.Flags().Int("words", 0, "")
	cmd.Flags().String("quote-len", "", "")
	cmd.Flags().String("text", "", "")
	for _, name := range changed {
		_ = cmd.Flags().Set(name, "1")
	}
	_ = flags
	return cmd
}

func TestBuildRunRequestValidCombos(t *testing.T) {
	settings := config.Defaults()
	tests := []struct {
		name    string
		flags   runFlags
		changed []string
		want    config.Mode
		length  int
	}{
		{"default", runFlags{}, nil, config.ModeTime, 30},
		{"time", runFlags{mode: "time", duration: 15}, []string{"duration"}, config.ModeTime, 15},
		{"words", runFlags{mode: "words", words: 25}, []string{"words"}, config.ModeWords, 25},
		{"quote", runFlags{mode: "quote", quoteLen: "short"}, []string{"quote-len"}, config.ModeQuote, 0},
		{"code", runFlags{mode: "code", text: "x.go"}, []string{"text"}, config.ModeCode, 0},
		{"theme", runFlags{mode: "time", duration: 30, theme: "nord"}, []string{"duration"}, config.ModeTime, 30},
		{"no-tui json", runFlags{mode: "words", words: 10, noTUI: true, json: true}, []string{"words"}, config.ModeWords, 10},
		{"words default length", runFlags{mode: "words"}, nil, config.ModeWords, 25},
		{"quote medium default", runFlags{mode: "quote"}, nil, config.ModeQuote, 0},
		{"code stdin", runFlags{mode: "code", text: "-"}, []string{"text"}, config.ModeCode, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildRunRequest(testRunCmd(tt.flags, tt.changed...), tt.flags, settings)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.mode != tt.want || got.length != tt.length {
				t.Fatalf("want %s/%d, got %s/%d", tt.want, tt.length, got.mode, got.length)
			}
		})
	}
}

func TestBuildRunRequestRejectsBadCombos(t *testing.T) {
	tests := []struct {
		name    string
		flags   runFlags
		changed []string
	}{
		{"time words", runFlags{mode: "time", words: 25}, []string{"words"}},
		{"words duration", runFlags{mode: "words", duration: 30}, []string{"duration"}},
		{"code missing text", runFlags{mode: "code"}, nil},
		{"unknown theme", runFlags{mode: "time", theme: "zzz"}, nil},
		{"json without no-tui", runFlags{mode: "words", words: 10, json: true}, []string{"words"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := buildRunRequest(testRunCmd(tt.flags, tt.changed...), tt.flags, config.Defaults())
			if err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
}

func TestRunCommandInvokesModelRunner(t *testing.T) {
	called := false
	root := NewRoot(
		WithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
		WithModelRunner(func(_ context.Context, model tea.Model) error {
			called = model != nil
			return nil
		}),
		WithSettingsStore(func() config.Settings { return config.Defaults() }, nil),
	)
	root.SetArgs([]string{"run", "--mode", "time", "--duration", "30", "--theme", "nord"})
	if err := root.Execute(); err != nil {
		t.Fatalf("run command: %v", err)
	}
	if !called {
		t.Fatal("model runner was not invoked")
	}
}
