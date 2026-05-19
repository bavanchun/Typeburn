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

// TestNormalize_Normalization mirrors every TestLoadReader_Normalization case
// against the in-memory string path: Normalize must apply the identical
// BOM/CRLF/trim/preserve pipeline as the reader core.
func TestNormalize_Normalization(t *testing.T) {
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
			got, err := Normalize(c.in)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != c.want {
				t.Errorf("got %q want %q", got, c.want)
			}
		})
	}
}

// TestNormalize_Errors mirrors TestLoadReader_Errors: the same sentinels must
// surface for the string path.
func TestNormalize_Errors(t *testing.T) {
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
			_, err := Normalize(c.in)
			if !errors.Is(err, c.wantErr) {
				t.Errorf("got err %v, want %v", err, c.wantErr)
			}
		})
	}
}

// TestNormalize_AtCapBoundaries mirrors the at-cap reader boundaries.
func TestNormalize_AtCapBoundaries(t *testing.T) {
	exactRunes := strings.Repeat("x", 10000)
	if _, err := Normalize(exactRunes); err != nil {
		t.Errorf("exactly 10000 runes should pass, got %v", err)
	}
	lines500 := strings.Repeat("a\n", 499) + "b" // 500 lines
	if _, err := Normalize(lines500); err != nil {
		t.Errorf("exactly 500 lines should pass, got %v", err)
	}
}

// TestNormalize_LoadReaderParity is the regression lock against rule
// divergence: for a representative set of inputs (incl. a byte-level BOM
// prefix), Normalize(s) must produce the SAME result and error class as
// loadReader(strings.NewReader(s)). One shared core, no drift.
func TestNormalize_LoadReaderParity(t *testing.T) {
	inputs := []string{
		"func f(){}\n",
		"a\r\nb\r\nc",
		"\xef\xbb\xbfpackage x",   // byte-level BOM prefix
		"\xef\xbb\xbf\r\nx\r\n\n", // BOM + CRLF + trailing
		"a\n\n\tb",
		"café — ünïcode",
		"",
		"  \n\t\n  ",
		"pkg\x00main",
		"abc\xff\xfe",
		strings.Repeat("x", 10001),
		strings.Repeat("a\n", 500) + "b",
		strings.Repeat("x", 10000),
		strings.Repeat("a\n", 499) + "b",
	}
	for i, in := range inputs {
		t.Run(string(rune('A'+i)), func(t *testing.T) {
			gotN, errN := Normalize(in)
			gotL, errL := loadReader(strings.NewReader(in))
			if gotN != gotL {
				t.Errorf("result divergence: Normalize=%q loadReader=%q", gotN, gotL)
			}
			switch {
			case errN == nil && errL == nil:
				// both ok
			case errN == nil || errL == nil:
				t.Errorf("error presence divergence: Normalize err=%v loadReader err=%v", errN, errL)
			default:
				for _, sentinel := range []error{ErrEmpty, ErrBinary, ErrTooLarge} {
					if errors.Is(errL, sentinel) != errors.Is(errN, sentinel) {
						t.Errorf("error class divergence on %v: Normalize=%v loadReader=%v",
							sentinel, errN, errL)
					}
				}
			}
		})
	}
}
