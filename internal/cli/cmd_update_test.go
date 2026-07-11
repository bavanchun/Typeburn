package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// upgradeResult is a stub Check result with an upgrade available.
func upgradeResult() *update.Result {
	return &update.Result{
		Current:          "2.2.0",
		Latest:           "v2.3.0",
		UpgradeAvailable: true,
		ReleaseURL:       "https://github.com/bavanchun/Typeburn/releases/tag/v2.3.0",
		CheckedAt:        time.Now().UTC(),
	}
}

// recordingApply returns an applyFn stub and a pointer to its called-flag. It
// drives the progress reporter through every stage so callers can assert the
// CLI prints the step lines.
func recordingApply(called *bool) func(context.Context, string, string, string, string, string, func(update.Stage)) (update.Outcome, error) {
	return func(_ context.Context, from, to, _, _, _ string, reportFn func(update.Stage)) (update.Outcome, error) {
		*called = true
		if reportFn != nil {
			reportFn(update.StageDownloading)
			reportFn(update.StageVerifying)
			reportFn(update.StageInstalling)
		}
		return update.Outcome{From: from, To: to}, nil
	}
}

func updateRoot(t *testing.T, out, errOut *bytes.Buffer, stdin io.Reader, args ...string) error {
	t.Helper()
	root := NewRoot(
		WithWriters(out, errOut),
		WithStdin(stdin),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
	)
	root.SetArgs(args)
	return root.Execute()
}

// withExecPath points execPathFn at a writable temp "install" and restores it.
func withExecPath(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	exec := filepath.Join(dir, "typeburn")
	if err := os.WriteFile(exec, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	orig := execPathFn
	execPathFn = func() (string, error) { return exec, nil }
	t.Cleanup(func() { execPathFn = orig })
	return exec
}

func TestUpdate_CheckReportsAvailability(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--check"); err != nil {
		t.Fatalf("update --check: %v", err)
	}
	if !strings.Contains(out.String(), "v2.3.0") {
		t.Errorf("expected latest version in output, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "Release notes: https://github.com/bavanchun/Typeburn/releases/tag/v2.3.0") {
		t.Errorf("expected release-notes URL, got:\n%s", out.String())
	}
	if called {
		t.Error("--check must not install")
	}
}

func TestUpdate_CheckOmitsReleaseNotesWhenEmpty(t *testing.T) {
	orig := getCheckFn()
	r := upgradeResult()
	r.ReleaseURL = "" // guarded away upstream (non-repo URL) → no notes line
	setCheckFn(stubCheck(r, nil))
	defer setCheckFn(orig)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--check"); err != nil {
		t.Fatalf("update --check: %v", err)
	}
	if strings.Contains(out.String(), "Release notes:") {
		t.Errorf("expected no release-notes line for empty URL, got:\n%s", out.String())
	}
}

func TestUpdate_CheckNetworkError(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(nil, errors.New("network unreachable")))
	defer setCheckFn(orig)

	var out, errOut bytes.Buffer
	// --check mirrors `version --check-update`: a check error is non-fatal,
	// reported on stderr, exit 0.
	if err := updateRoot(t, &out, &errOut, &bytes.Buffer{}, "update", "--check"); err != nil {
		t.Fatalf("update --check with error should exit 0, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "could not check for updates") {
		t.Errorf("expected error on stderr, got:\n%s", errOut.String())
	}
}

func TestUpdate_CheckDevSkip(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(nil, nil)) // dev/pseudo
	defer setCheckFn(orig)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--check"); err != nil {
		t.Fatalf("update --check (dev): %v", err)
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Errorf("expected 'skipped', got:\n%s", out.String())
	}
}

func TestUpdate_DevInstallRefused(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(nil, nil))
	defer setCheckFn(orig)

	err := updateRoot(t, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--yes")
	if ExitCode(err) != ExitUsage {
		t.Errorf("dev install should exit %d, got %d (err=%v)", ExitUsage, ExitCode(err), err)
	}
}

func TestUpdate_AlreadyLatest(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(&update.Result{Current: "2.3.0", Latest: "v2.3.0", UpgradeAvailable: false}, nil))
	defer setCheckFn(orig)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--yes"); err != nil {
		t.Fatalf("update already-latest should exit 0: %v", err)
	}
	if !strings.Contains(out.String(), "latest") {
		t.Errorf("expected 'latest', got:\n%s", out.String())
	}
}

func TestUpdate_ManagedInstallRefused(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)

	origExec := execPathFn
	execPathFn = func() (string, error) {
		return "/opt/homebrew/Cellar/typeburn/2.2.0/bin/typeburn", nil
	}
	defer func() { execPathFn = origExec }()

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--yes")
	if ExitCode(err) != ExitManagedInstall {
		t.Errorf("managed install should exit %d, got %d", ExitManagedInstall, ExitCode(err))
	}
	if !strings.Contains(out.String(), "brew upgrade typeburn") {
		t.Errorf("expected brew instruction, got:\n%s", out.String())
	}
	if called {
		t.Error("managed install must not run the pipeline")
	}
}

func TestUpdate_YesSkipsPromptAndInstalls(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)
	withExecPath(t)

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, &bytes.Buffer{}, "update", "--yes"); err != nil {
		t.Fatalf("update --yes: %v", err)
	}
	if !called {
		t.Error("--yes should run the pipeline")
	}
	if !strings.Contains(out.String(), "updated") {
		t.Errorf("expected 'updated' confirmation, got:\n%s", out.String())
	}
	// Each progress stage must surface to the user during the swap.
	for _, stage := range []string{"downloading", "verifying", "installing"} {
		if !strings.Contains(out.String(), stage) {
			t.Errorf("expected %q progress line, got:\n%s", stage, out.String())
		}
	}
	// Release notes URL must be shown before the install.
	if !strings.Contains(out.String(), "Release notes: https://github.com/bavanchun/Typeburn/releases/tag/v2.3.0") {
		t.Errorf("expected release-notes URL, got:\n%s", out.String())
	}
}

func TestUpdate_NonTTYWithoutYesRefuses(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)
	withExecPath(t)

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	// stdin is a buffer (non-tty) and no --yes → must refuse, not block.
	err := updateRoot(t, &bytes.Buffer{}, &bytes.Buffer{}, &bytes.Buffer{}, "update")
	if ExitCode(err) != ExitUsage {
		t.Errorf("non-tty without --yes should exit %d, got %d", ExitUsage, ExitCode(err))
	}
	if called {
		t.Error("must not install when refusing to prompt")
	}
}

func TestUpdate_PromptDeclineAborts(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)
	withExecPath(t)

	origTTY := isInteractive
	isInteractive = func(io.Reader) bool { return true }
	defer func() { isInteractive = origTTY }()

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, strings.NewReader("n\n"), "update"); err != nil {
		t.Fatalf("declined update should exit 0: %v", err)
	}
	if called {
		t.Error("declined prompt must not install")
	}
	if !strings.Contains(out.String(), "cancelled") {
		t.Errorf("expected 'cancelled', got:\n%s", out.String())
	}
}

func TestUpdate_PromptAcceptInstalls(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(upgradeResult(), nil))
	defer setCheckFn(orig)
	withExecPath(t)

	origTTY := isInteractive
	isInteractive = func(io.Reader) bool { return true }
	defer func() { isInteractive = origTTY }()

	var called bool
	setApplyFn(recordingApply(&called))
	defer setApplyFn(update.Apply)

	var out bytes.Buffer
	if err := updateRoot(t, &out, &bytes.Buffer{}, strings.NewReader("y\n"), "update"); err != nil {
		t.Fatalf("accepted update: %v", err)
	}
	if !called {
		t.Error("accepted prompt should install")
	}
}

// TestUpdate_RecognizedSubcommand guards the regression where `typeburn update`
// fell through to the TUI because no such subcommand existed.
func TestUpdate_RecognizedSubcommand(t *testing.T) {
	orig := getCheckFn()
	setCheckFn(stubCheck(nil, nil)) // dev → refuse path, but proves it parsed as a subcommand
	defer setCheckFn(orig)

	homeLaunched := false
	root := NewRoot(
		WithWriters(&bytes.Buffer{}, &bytes.Buffer{}),
		WithStdin(&bytes.Buffer{}),
		WithHomeRunner(func(context.Context, string, string) error {
			homeLaunched = true
			return nil
		}),
	)
	root.SetArgs([]string{"update", "--yes"})
	_ = root.Execute()
	if homeLaunched {
		t.Error("`update` must be a subcommand, not a TUI fall-through")
	}
}
