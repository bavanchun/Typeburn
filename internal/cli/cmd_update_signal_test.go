//go:build !windows

package cli

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/v2/internal/update"
)

// Environment keys used to drive the helper subprocess below. The interrupt
// path can only be observed in a real OS process: an in-process test never
// reaches the kernel's default signal disposition, which is precisely the
// behaviour under test.
const (
	envUpdateHelper = "TYPEBURN_TEST_UPDATE_HELPER"
	envInstallDir   = "TYPEBURN_TEST_INSTALL_DIR"
	envHangProxy    = "TYPEBURN_TEST_HANG_PROXY"
)

// TestUpdateHelperProcess is not a test. Re-executed as a child process, it runs
// the real `typeburn update --yes` command against the real update.Apply, with
// only the release check stubbed and the asset download pointed at a proxy that
// never answers. Everything else — the O_EXCL lock, the temp artifacts, the
// signal disposition — is production code.
func TestUpdateHelperProcess(t *testing.T) {
	if os.Getenv(envUpdateHelper) != "1" {
		t.Skip("helper for TestUpdate_PlainPathSurvivesSignals")
	}

	setCheckFn(stubCheck(&update.Result{
		Current:          "v2.0.0",
		Latest:           "v9.9.9",
		UpgradeAvailable: true,
	}, nil))

	execPath := filepath.Join(os.Getenv(envInstallDir), "typeburn")
	execPathFn = func() (string, error) { return execPath, nil }

	root := NewRoot(WithWriters(os.Stdout, os.Stderr))
	root.SetArgs([]string{"update", "--yes"})
	err := root.Execute()
	if err != nil {
		fmt.Fprintf(os.Stdout, "ERR %v\n", err)
	}
	fmt.Fprintf(os.Stdout, "EXIT %d\n", ExitCode(err))
	os.Exit(ExitCode(err))
}

// hangingProxy serves CONNECT requests that never receive a response, so an
// asset download started through it stays in flight until its context is
// cancelled. This is what a stalled network looks like to the updater.
func hangingProxy(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

// startUpdateHelper launches the helper subprocess against installDir and
// returns the command plus a func that reads everything it printed.
func startUpdateHelper(t *testing.T, installDir string) (*exec.Cmd, func() string) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestUpdateHelperProcess$", "-test.timeout=60s")
	proxy := hangingProxy(t)
	cmd.Env = append(os.Environ(),
		envUpdateHelper+"=1",
		envInstallDir+"="+installDir,
		envHangProxy+"="+proxy,
		"HTTPS_PROXY="+proxy,
		"https_proxy="+proxy,
		"HTTP_PROXY="+proxy,
		"http_proxy="+proxy,
		"NO_PROXY=",
		"no_proxy=",
	)
	var out strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start helper: %v", err)
	}
	t.Cleanup(func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	})
	return cmd, out.String
}

// waitForFile polls until path exists or the deadline passes.
func waitForFile(t *testing.T, path string, within time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(within)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// An update killed by a signal must leave the install directory exactly as it
// found it. The lock is an O_EXCL file with no owner tracking at creation time,
// so a leaked one makes every later `typeburn update` refuse to start.
func TestUpdate_PlainPathSurvivesSignals(t *testing.T) {
	signals := map[string]syscall.Signal{
		"interrupt":  syscall.SIGINT,
		"terminated": syscall.SIGTERM,
	}
	for name, sig := range signals {
		t.Run(name, func(t *testing.T) {
			installDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(installDir, "typeburn"), []byte("OLD"), 0o755); err != nil {
				t.Fatal(err)
			}
			lockPath := filepath.Join(installDir, ".typeburn-update.lock")

			cmd, output := startUpdateHelper(t, installDir)
			if !waitForFile(t, lockPath, 20*time.Second) {
				t.Fatalf("update never reached the download stage; output:\n%s", output())
			}

			if err := cmd.Process.Signal(sig); err != nil {
				t.Fatalf("signal: %v", err)
			}

			done := make(chan error, 1)
			go func() { done <- cmd.Wait() }()
			var waitErr error
			select {
			case waitErr = <-done:
			case <-time.After(30 * time.Second):
				t.Fatalf("helper did not exit after %s; output:\n%s", sig, output())
			}

			// An interrupted update must exit on the documented abort code and
			// say what it did, not die silently on the signal's own status.
			if code := helperExitCode(t, waitErr); code != ExitAbort {
				t.Errorf("%s: exit code = %d, want %d (ExitAbort)\noutput:\n%s", sig, code, ExitAbort, output())
			}
			if !strings.Contains(output(), "update cancelled; nothing was installed") {
				t.Errorf("%s: interrupt was not reported to the user\noutput:\n%s", sig, output())
			}

			if _, err := os.Stat(lockPath); err == nil {
				t.Errorf("%s: lock file leaked, every future update refuses to start\noutput:\n%s", sig, output())
			}
			leftovers := leftoverArtifacts(t, installDir)
			if len(leftovers) > 0 {
				t.Errorf("%s: partial artifacts left behind: %v\noutput:\n%s", sig, leftovers, output())
			}
		})
	}
}

// helperExitCode extracts the child's process exit status from cmd.Wait's error.
func helperExitCode(t *testing.T, err error) int {
	t.Helper()
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("helper failed without an exit status: %v", err)
	}
	return exitErr.ExitCode()
}

// leftoverArtifacts lists everything in dir that is not the installed binary.
func leftoverArtifacts(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	var extra []string
	for _, e := range entries {
		if e.Name() != "typeburn" {
			extra = append(extra, e.Name())
		}
	}
	return extra
}
