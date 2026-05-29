package update

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestParseChecksums(t *testing.T) {
	in := []byte(
		"abc123  typeburn_2.2.0_linux_amd64.tar.gz\n" +
			"\n" + // blank line ignored
			"DEF456  typeburn_2.2.0_windows_amd64.zip\n" +
			"malformed-no-second-field\n" +
			"  99aa  typeburn_2.2.0_darwin_arm64.tar.gz  \n", // extra spacing
	)
	got := parseChecksums(in)

	want := map[string]string{
		"typeburn_2.2.0_linux_amd64.tar.gz":  "abc123",
		"typeburn_2.2.0_windows_amd64.zip":   "def456", // lowercased
		"typeburn_2.2.0_darwin_arm64.tar.gz": "99aa",
	}
	if len(got) != len(want) {
		t.Fatalf("entry count: got %d, want %d (%v)", len(got), len(want), got)
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s: got %q, want %q", k, got[k], v)
		}
	}
}

func writeTemp(t *testing.T, data []byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "asset.bin")
	if err := os.WriteFile(p, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func TestVerifySHA256_Match(t *testing.T) {
	data := []byte("typeburn release archive bytes")
	p := writeTemp(t, data)
	if err := verifySHA256(p, sha256Hex(data)); err != nil {
		t.Errorf("expected match, got %v", err)
	}
	// case-insensitive want
	if err := verifySHA256(p, toUpper(sha256Hex(data))); err != nil {
		t.Errorf("uppercase want should match: %v", err)
	}
}

func toUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'f' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func TestVerifySHA256_Mismatch(t *testing.T) {
	p := writeTemp(t, []byte("real bytes"))
	if err := verifySHA256(p, sha256Hex([]byte("different bytes"))); err == nil {
		t.Error("expected mismatch error, got nil")
	}
}

func TestVerifySHA256_EmptyWant(t *testing.T) {
	p := writeTemp(t, []byte("bytes"))
	if err := verifySHA256(p, ""); err == nil {
		t.Error("expected error for empty checksum, got nil")
	}
}

func TestVerifySHA256_MissingFile(t *testing.T) {
	if err := verifySHA256(filepath.Join(t.TempDir(), "nope"), "deadbeef"); err == nil {
		t.Error("expected error for missing file, got nil")
	}
}
