//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinary_Swaps(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "typeburn")
	newBin := filepath.Join(dir, "typeburn.new")

	if err := os.WriteFile(target, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(newBin, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(target, newBin); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "NEW" {
		t.Errorf("target content = %q, want NEW", got)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("target not executable, mode = %v", info.Mode())
	}
	if _, err := os.Stat(newBin); !os.IsNotExist(err) {
		t.Errorf("new binary should be consumed by rename, stat err = %v", err)
	}
}

func TestReplaceBinary_RefusesSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real")
	target := filepath.Join(dir, "typeburn")
	newBin := filepath.Join(dir, "typeburn.new")

	if err := os.WriteFile(real, []byte("REAL"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, target); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}
	if err := os.WriteFile(newBin, []byte("NEW"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(target, newBin); err == nil {
		t.Error("expected refusal to replace a symlinked target")
	}
}
