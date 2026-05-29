package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// tarGzBytes builds an in-memory tar.gz holding a single regular-file member.
func tarGzBytes(t *testing.T, member, content string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: member, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte(content))
	_ = tw.Close()
	_ = gz.Close()
	return buf.Bytes()
}

func TestApply_SwapsBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix rename-over-running-exe path; Windows swap covered separately")
	}
	archive := tarGzBytes(t, "typeburn", "NEW-BINARY")
	asset := assetName("2.3.0", "linux", "amd64")
	srv := fakeRelease(t, "v2.3.0", asset, archive)
	old := getDownloadBase()
	setDownloadBase(srv.URL)
	defer setDownloadBase(old)

	dir := t.TempDir()
	exec := filepath.Join(dir, "typeburn")
	if err := os.WriteFile(exec, []byte("OLD-BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	var stages []Stage
	out, err := Apply(context.Background(), "2.2.0", "v2.3.0", exec, "linux", "amd64",
		func(s Stage) { stages = append(stages, s) })
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.From != "2.2.0" || out.To != "v2.3.0" {
		t.Errorf("outcome = %+v, want 2.2.0 → v2.3.0", out)
	}
	// The reporter must see each stage, in order, on a successful update.
	want := []Stage{StageDownloading, StageVerifying, StageInstalling}
	if len(stages) != len(want) {
		t.Fatalf("stages = %v, want %v", stages, want)
	}
	for i := range want {
		if stages[i] != want[i] {
			t.Errorf("stage[%d] = %s, want %s", i, stages[i], want[i])
		}
	}

	got, _ := os.ReadFile(exec)
	if string(got) != "NEW-BINARY" {
		t.Errorf("installed binary = %q, want NEW-BINARY", got)
	}
	// Temp archive, extracted .new, and lock file must all be cleaned up.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "typeburn" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestApply_FailureLeavesOriginal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix path")
	}
	asset := assetName("2.3.0", "linux", "amd64")
	// fakeRelease publishes a checksum matching the served body, so download +
	// verify pass — but "not-a-tar-gz" is not a valid archive, so extraction
	// fails. The original binary must survive an aborted update.
	srv := fakeRelease(t, "v2.3.0", asset, []byte("not-a-tar-gz"))
	old := getDownloadBase()
	setDownloadBase(srv.URL)
	defer setDownloadBase(old)

	dir := t.TempDir()
	exec := filepath.Join(dir, "typeburn")
	if err := os.WriteFile(exec, []byte("OLD-BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	if _, err := Apply(context.Background(), "2.2.0", "v2.3.0", exec, "linux", "amd64", nil); err == nil {
		t.Error("expected failure on non-archive payload")
	}
	got, _ := os.ReadFile(exec)
	if string(got) != "OLD-BINARY" {
		t.Errorf("original binary altered on failure: %q", got)
	}
	// And no temp/lock debris remains.
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "typeburn" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}

func TestBinaryMember(t *testing.T) {
	if binaryMember("windows") != "typeburn.exe" {
		t.Error("windows member must be typeburn.exe")
	}
	if binaryMember("linux") != "typeburn" {
		t.Error("non-windows member must be typeburn")
	}
}
