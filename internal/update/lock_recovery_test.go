package update

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// deadPID runs a trivial process to completion and returns its PID, which is
// then guaranteed not to name a live process.
func deadPID(t *testing.T) int {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("helper relies on a POSIX no-op binary")
	}
	c := exec.Command("/bin/echo")
	if err := c.Run(); err != nil {
		t.Fatalf("spawn throwaway process: %v", err)
	}
	return c.Process.Pid
}

// writeLockRecord writes a lock file by hand in the documented on-disk format,
// so the reader is exercised against a literal payload rather than against
// whatever the writer happens to produce.
func writeLockRecord(t *testing.T, path string, pid int, taken time.Time) {
	t.Helper()
	body := fmt.Sprintf("pid %d\ntaken %d\n", pid, taken.Unix())
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

// seedAbandonedUpdate writes the four artifacts an updater killed mid-download
// leaves behind: the lock plus the three temp files, each of which independently
// blocks every later run through its own O_EXCL create.
func seedAbandonedUpdate(t *testing.T, dir, asset string, ownerPID int) {
	t.Helper()
	writeLockRecord(t, filepath.Join(dir, lockFileName), ownerPID, time.Now())
	for _, name := range []string{"checksums.txt", asset, "typeburn.new"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("stale"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
}

// An update killed abruptly leaves a lock and three temp files. Recovering must
// not require the user to delete anything by hand: the next run reclaims a lock
// whose owner is gone and clears the debris it left.
func TestApply_RecoversAbandonedUpdateInOneRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix rename-over-running-exe path")
	}
	archive := tarGzBytes(t, "typeburn", "NEW-BINARY")
	asset := assetName("9.9.9", "linux", "amd64")
	srv := fakeRelease(t, "v9.9.9", asset, archive)
	old := getDownloadBase()
	setDownloadBase(srv.URL)
	defer setDownloadBase(old)

	dir := t.TempDir()
	execPath := filepath.Join(dir, "typeburn")
	if err := os.WriteFile(execPath, []byte("OLD-BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}
	seedAbandonedUpdate(t, dir, asset, deadPID(t))

	if _, err := Apply(context.Background(), "v2.0.0", "v9.9.9", execPath, "linux", "amd64", nil); err != nil {
		t.Fatalf("abandoned update did not self-heal: %v", err)
	}

	got, _ := os.ReadFile(execPath)
	if string(got) != "NEW-BINARY" {
		t.Errorf("installed binary = %q, want NEW-BINARY", got)
	}
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "typeburn" {
			t.Errorf("leftover temp file after recovery: %s", e.Name())
		}
	}
}
