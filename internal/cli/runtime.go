package cli

import (
	"context"
	"errors"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/bavanchun/Typeburn/v2/internal/app"
	"github.com/bavanchun/Typeburn/v2/internal/codetext"
	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/storage"
	"github.com/bavanchun/Typeburn/v2/internal/update"
	"github.com/bavanchun/Typeburn/v2/internal/version"
)

type env struct {
	stdout io.Writer
	stderr io.Writer
	stdin  io.Reader

	loadCode     func(string) (string, error)
	loadHistory  func() []storage.Record
	loadSettings func() config.Settings
	saveSettings func(config.Settings) error
	runHome      func(context.Context, string, string) error
	runModel     func(context.Context, tea.Model) error
}

func defaultEnv() env {
	return env{
		stdout:       os.Stdout,
		stderr:       os.Stderr,
		stdin:        os.Stdin,
		loadCode:     codetext.Load,
		loadHistory:  storage.LoadHistory,
		loadSettings: storage.LoadSettings,
		saveSettings: storage.SaveSettings,
		runHome:      runHomeTUI,
		runModel:     runModelTUI,
	}
}

type Option func(*env)

func WithWriters(stdout, stderr io.Writer) Option {
	return func(e *env) {
		e.stdout = stdout
		e.stderr = stderr
	}
}

func WithStdin(stdin io.Reader) Option {
	return func(e *env) { e.stdin = stdin }
}

func WithCodeLoader(load func(string) (string, error)) Option {
	return func(e *env) { e.loadCode = load }
}

func WithHistoryLoader(load func() []storage.Record) Option {
	return func(e *env) { e.loadHistory = load }
}

func WithSettingsStore(load func() config.Settings, save func(config.Settings) error) Option {
	return func(e *env) {
		e.loadSettings = load
		e.saveSettings = save
	}
}

func WithHomeRunner(run func(context.Context, string, string) error) Option {
	return func(e *env) { e.runHome = run }
}

func WithModelRunner(run func(context.Context, tea.Model) error) Option {
	return func(e *env) { e.runModel = run }
}

// resolveUpdateHint performs the opportunistic update check when the user has
// opted in (settings.UpdateCheck == true) and the build has a real release version.
// Uses an 800ms timeout — tighter than the explicit --check-update path (1.5s) so
// TUI launch latency on a cache miss stays imperceptible.
// Returns nil on any error or dev/pseudo-version — caller always gets a clean hint or nothing.
func resolveUpdateHint(ctx context.Context, settings config.Settings) *update.Result {
	if !settings.UpdateCheck {
		return nil
	}
	ver := version.Resolve().Version
	ctx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()
	r, err := update.Check(ctx, ver, false)
	if err != nil || r == nil || !r.UpgradeAvailable {
		return nil
	}
	return r
}

func runHomeTUI(ctx context.Context, codeText, codeHint string) error {
	settings := storage.LoadSettings()
	hint := resolveUpdateHint(ctx, settings)
	return runModelTUI(ctx, app.NewFromDisk(codeText, codeHint, hint))
}

func runModelTUI(ctx context.Context, model tea.Model) error {
	p := tea.NewProgram(model, tea.WithContext(ctx))
	_, err := p.Run()
	return err
}

func codeHintFor(err error) string {
	switch {
	case errors.Is(err, codetext.ErrEmpty):
		return "text file is empty"
	case errors.Is(err, codetext.ErrTooLarge):
		return "text file too large"
	case errors.Is(err, codetext.ErrBinary):
		return "file is not text"
	default:
		return "could not read text file"
	}
}
