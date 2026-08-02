package update

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// paddedTarGz builds a tar.gz whose first member is padBytes of highly
// compressible filler and whose second member is the wanted binary. The filler
// is walked past, not extracted, which is precisely the case a per-member bound
// never sees.
func paddedTarGz(t *testing.T, member, content string, padBytes int64) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{Name: "PADDING", Mode: 0o644, Size: padBytes, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.CopyN(tw, zeroReader{}, padBytes); err != nil {
		t.Fatal(err)
	}
	if err := tw.WriteHeader(&tar.Header{Name: member, Mode: 0o755, Size: int64(len(content)), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0
	}
	return len(p), nil
}

// A megabyte of archive that expands to a gigabyte is still a resource attack
// even though the checksum matched — the release host is already trusted at
// this point, so this bounds damage rather than preventing intrusion. The bound
// has to cover members that are skipped, because walking past them decompresses
// them just the same.
func TestExtractBinary_BoundsPaddingWalkedPastTheWantedMember(t *testing.T) {
	archive := paddedTarGz(t, "typeburn", "NEW-BINARY", decompressCap+(1<<20))
	if len(archive) > 1<<20 {
		t.Fatalf("padded archive is %d bytes on the wire; the test needs it small to be meaningful", len(archive))
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "typeburn_9.9.9_linux_amd64.tar.gz")
	if err := os.WriteFile(path, archive, 0o600); err != nil {
		t.Fatal(err)
	}

	dest, err := extractBinary(path, "typeburn", dir)
	if !errors.Is(err, errDecompressCap) {
		t.Fatalf("err = %v, want it to report the decompression cap", err)
	}
	if _, statErr := os.Stat(dest); !os.IsNotExist(statErr) {
		t.Errorf("a partial extraction was left at %s", dest)
	}
}

// The bound must not clip a real release, which is one binary plus a few small
// docs well inside the cap.
func TestExtractBinary_AllowsAnArchiveWithinTheCap(t *testing.T) {
	archive := paddedTarGz(t, "typeburn", "NEW-BINARY", 4<<20)
	dir := t.TempDir()
	path := filepath.Join(dir, "typeburn_9.9.9_linux_amd64.tar.gz")
	if err := os.WriteFile(path, archive, 0o600); err != nil {
		t.Fatal(err)
	}

	dest, err := extractBinary(path, "typeburn", dir)
	if err != nil {
		t.Fatalf("extractBinary: %v", err)
	}
	got, _ := os.ReadFile(dest)
	if string(got) != "NEW-BINARY" {
		t.Errorf("extracted %q, want NEW-BINARY", got)
	}
}
