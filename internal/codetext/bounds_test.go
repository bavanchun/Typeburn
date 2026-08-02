package codetext

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// floodReader produces an endless run of 'x' and records how much of it the
// caller took.
//
// It stops with an error past ceiling instead of running forever, so a reader
// core that went back to reading until EOF fails this test on the byte count
// rather than by exhausting the machine it runs on.
type floodReader struct {
	consumed int
	ceiling  int
}

func (f *floodReader) Read(p []byte) (int, error) {
	if f.consumed >= f.ceiling {
		return 0, errors.New("floodReader: caller read past the ceiling")
	}
	for i := range p {
		p[i] = 'x'
	}
	f.consumed += len(p)
	return len(p), nil
}

// TestLoadReader_StopsReadingAtTheBound is the allocation assertion: a caller
// handed an arbitrarily large stream must take a bounded prefix of it, so a
// 400 MiB file is refused without ever being in memory.
func TestLoadReader_StopsReadingAtTheBound(t *testing.T) {
	r := &floodReader{ceiling: 8 << 20}

	_, err := loadReader(r)

	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("oversize stream: got err %v, want ErrTooLarge", err)
	}
	if r.consumed > maxBytes+1 {
		t.Errorf("read %d bytes to reject an input capped at %d — the whole input was pulled into memory first",
			r.consumed, maxBytes)
	}
}

// TestLoad_OversizeFileIsRejectedNotTruncated: silently handing back the first
// 10 000 runes of a large file would start a test against a target the user
// never chose.
func TestLoad_OversizeFileIsRejectedNotTruncated(t *testing.T) {
	path := filepath.Join(t.TempDir(), "big.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", 1<<20)), 0600); err != nil {
		t.Fatal(err)
	}

	got, err := Load(path)

	if !errors.Is(err, ErrTooLarge) {
		t.Fatalf("got err %v, want ErrTooLarge", err)
	}
	if got != "" {
		t.Errorf("rejected input still returned %d bytes of text", len(got))
	}
}

// TestLoad_EndlessDeviceReturns covers the stream that has no size at all.
// /dev/zero never reaches EOF, so a read-to-EOF loader does not fail here —
// it never comes back. The deadline is the assertion.
func TestLoad_EndlessDeviceReturns(t *testing.T) {
	const dev = "/dev/zero"
	if _, err := os.Stat(dev); err != nil {
		t.Skipf("%s not available: %v", dev, err)
	}

	done := make(chan error, 1)
	go func() {
		_, err := Load(dev)
		done <- err
	}()

	select {
	case err := <-done:
		if err == nil {
			t.Fatalf("an endless stream of NUL bytes loaded as a snippet")
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Load did not return within 10s: it is reading to an EOF that never arrives")
	}
}

// TestNormalize_RejectsControlRunes pins the runes that make a Code target
// uncompletable. ESC and 0x08 are bound to Back and Backspace, so no key press
// can produce them; a target containing one is a test with no way out.
func TestNormalize_RejectsControlRunes(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"bare carriage return", "a\rb"},
		{"backspace", "func\x08 main"},
		{"vertical tab", "a\vb"},
		{"form feed", "a\fb"},
		{"delete", "a\x7fb"},
		{"ansi csi colour", "\x1b[31mred\x1b[0m"},
		{"ansi osc title", "\x1b]0;title\atext"},
		{"c1 next-line", "a\u0085b"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := Normalize(tc.in); !errors.Is(err, ErrControl) {
				t.Errorf("Normalize(%q): got err %v, want ErrControl", tc.in, err)
			}
			if _, err := loadReader(strings.NewReader(tc.in)); !errors.Is(err, ErrControl) {
				t.Errorf("loadReader(%q): got err %v, want ErrControl", tc.in, err)
			}
		})
	}
}

// TestNormalize_KeepsTheTextControlsCodeModeNeeds is the other half: rejecting
// controls must not reject a real source file. CRLF is converted, not refused,
// and tabs are the indentation the user is asked to type.
func TestNormalize_KeepsTheTextControlsCodeModeNeeds(t *testing.T) {
	for _, tc := range []struct {
		name, in, want string
	}{
		{"crlf file", "func main() {\r\n\tprintln(1)\r\n}\r\n", "func main() {\n\tprintln(1)\n}"},
		{"crlf with a blank line", "a\r\n\r\nb", "a\n\nb"},
		{"bom then crlf", "\xef\xbb\xbfa\r\nb\r\n", "a\nb"},
		{"tabs and newlines", "a\n\tb\n", "a\n\tb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Normalize(tc.in)
			if err != nil {
				t.Fatalf("Normalize(%q): unexpected err %v", tc.in, err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// TestNormalize_LoadReaderParity_Controls extends the shared-core lock to the
// new rule: one core means the paste path and the --text path cannot disagree
// about what a control rune is.
func TestNormalize_LoadReaderParity_Controls(t *testing.T) {
	for _, in := range []string{"a\rb", "\x1b[31mx", "a\vb", "a\r\nb", "a\tb\n"} {
		gotN, errN := Normalize(in)
		gotL, errL := loadReader(strings.NewReader(in))
		if gotN != gotL {
			t.Errorf("%q: result divergence Normalize=%q loadReader=%q", in, gotN, gotL)
		}
		if errors.Is(errN, ErrControl) != errors.Is(errL, ErrControl) {
			t.Errorf("%q: error class divergence Normalize=%v loadReader=%v", in, errN, errL)
		}
	}
}
