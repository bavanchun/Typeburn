package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// makeTarGz builds an in-memory tar.gz from the given entries (name→content);
// a non-empty linkname makes that entry a symlink instead of a regular file.
func makeTarGz(t *testing.T, reg map[string]string, symlink string) string {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range reg {
		_ = tw.WriteHeader(&tar.Header{Name: name, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg})
		_, _ = tw.Write([]byte(content))
	}
	if symlink != "" {
		_ = tw.WriteHeader(&tar.Header{Name: symlink, Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	}
	_ = tw.Close()
	_ = gz.Close()
	p := filepath.Join(t.TempDir(), "a.tar.gz")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func makeZip(t *testing.T, reg map[string]string) string {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range reg {
		w, _ := zw.Create(name)
		_, _ = w.Write([]byte(content))
	}
	_ = zw.Close()
	p := filepath.Join(t.TempDir(), "a.zip")
	if err := os.WriteFile(p, buf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestExtractBinary_TarGz(t *testing.T) {
	arc := makeTarGz(t, map[string]string{
		"typeburn":  "BINARY-CONTENT",
		"README.md": "docs decoy",
	}, "")
	dir := t.TempDir()
	got, err := extractBinary(arc, "typeburn", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	data, _ := os.ReadFile(got)
	if string(data) != "BINARY-CONTENT" {
		t.Errorf("content = %q, want BINARY-CONTENT", data)
	}
	if runtime.GOOS != "windows" {
		info, _ := os.Stat(got)
		if info.Mode().Perm() != 0o755 {
			t.Errorf("mode = %v, want 0755", info.Mode().Perm())
		}
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	arc := makeZip(t, map[string]string{
		"typeburn.exe": "WIN-BINARY",
		"LICENSE":      "decoy",
	})
	dir := t.TempDir()
	got, err := extractBinary(arc, "typeburn.exe", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if data, _ := os.ReadFile(got); string(data) != "WIN-BINARY" {
		t.Errorf("content = %q, want WIN-BINARY", data)
	}
}

func TestExtractBinary_Zip_MissingMember(t *testing.T) {
	arc := makeZip(t, map[string]string{"LICENSE": "only docs"})
	if _, err := extractBinary(arc, "typeburn.exe", t.TempDir()); err == nil {
		t.Error("expected error when the wanted member is absent from the zip")
	}
}

func TestExtractBinary_MissingMember(t *testing.T) {
	arc := makeTarGz(t, map[string]string{"README.md": "only docs"}, "")
	if _, err := extractBinary(arc, "typeburn", t.TempDir()); err == nil {
		t.Error("expected missing-member error, got nil")
	}
}

func TestExtractBinary_RejectsSymlinkMember(t *testing.T) {
	arc := makeTarGz(t, nil, "typeburn") // "typeburn" is a symlink entry
	if _, err := extractBinary(arc, "typeburn", t.TempDir()); err == nil {
		t.Error("expected symlink-member rejection, got nil")
	}
}

func TestSafeMember(t *testing.T) {
	cases := []struct {
		name, want string
		ok         bool
	}{
		{"typeburn", "typeburn", true},
		{"./typeburn", "typeburn", true},
		{"../evil", "typeburn", false},
		{"nested/typeburn", "typeburn", false},
		{`..\evil`, "typeburn", false},
		{"typeburn.exe", "typeburn.exe", true},
	}
	for _, c := range cases {
		if got := safeMember(c.name, c.want); got != c.ok {
			t.Errorf("safeMember(%q,%q) = %v, want %v", c.name, c.want, got, c.ok)
		}
	}
}
