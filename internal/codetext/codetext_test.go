package codetext

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReader_Normalization(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"crlf to lf", "a\r\nb\r\nc", "a\nb\nc"},
		{"strip utf8 bom", "\xef\xbb\xbfpackage x", "package x"},
		{"trim exactly one trailing nl", "a\n\n", "a\n"},
		{"single trailing nl trimmed", "a\n", "a"},
		{"no trailing nl untouched", "a", "a"},
		{"interior blanks and tabs preserved", "a\n\n\tb", "a\n\n\tb"},
		{"unicode kept", "café — ünïcode", "café — ünïcode"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := loadReader(strings.NewReader(c.in))
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}

func TestLoadReader_Errors(t *testing.T) {
	long := strings.Repeat("x", 10001)
	manyLines := strings.Repeat("a\n", 500) + "b" // 501 lines
	cases := []struct {
		name    string
		in      string
		wantErr error
	}{
		{"empty", "", ErrEmpty},
		{"whitespace only", "  \n\t\n  ", ErrEmpty},
		{"nul byte is binary", "package\x00main", ErrBinary},
		{"invalid utf8 is binary", "abc\xff\xfe", ErrBinary},
		{"too many runes", long, ErrTooLarge},
		{"too many lines", manyLines, ErrTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := loadReader(strings.NewReader(c.in))
			if !errors.Is(err, c.wantErr) {
				t.Errorf("got err %v, want %v", err, c.wantErr)
			}
		})
	}
}

func TestLoadReader_AtCapBoundaries(t *testing.T) {
	exactRunes := strings.Repeat("x", 10000)
	if _, err := loadReader(strings.NewReader(exactRunes)); err != nil {
		t.Errorf("exactly 10000 runes should pass, got %v", err)
	}
	lines500 := strings.Repeat("a\n", 499) + "b" // 500 lines
	if _, err := loadReader(strings.NewReader(lines500)); err != nil {
		t.Errorf("exactly 500 lines should pass, got %v", err)
	}
}

func TestLoad_FileAndStdin(t *testing.T) {
	dir := t.TempDir()
	fp := filepath.Join(dir, "snippet.go")
	if err := os.WriteFile(fp, []byte("func main() {\n\tprintln(1)\n}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := Load(fp)
	if err != nil {
		t.Fatalf("Load(file): %v", err)
	}
	if got != "func main() {\n\tprintln(1)\n}" {
		t.Errorf("file load mismatch: %q", got)
	}

	if _, err := Load(filepath.Join(dir, "does-not-exist")); err == nil {
		t.Error("missing file should error")
	}
}
