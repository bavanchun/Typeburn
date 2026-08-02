// Package codetext loads and normalizes user-supplied text/code for the
// Code typing mode. It is the I/O boundary for `--text <file>` / `--text -`
// so the pure-logic packages (words, typing) stay free of file/stdin access.
//
// Normalization (full-literal-safe): strip a leading UTF-8 BOM, convert CRLF
// to LF, and trim exactly one trailing newline (so the snippet's final line
// needs no closing Enter). Tabs, interior blank lines, and indentation are
// preserved verbatim — Code mode requires the user to type them. Every other
// control rune is rejected, because no key press can produce one and a snippet
// containing one could never be completed.
package codetext

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"
)

// Caps. A snippet over either bound is rejected (not truncated) so the
// renderer/viewport never face pathological input and the caller can show a
// clear reason instead.
const (
	maxRunes = 10000
	maxLines = 500

	// bomBytes is the length of utf8BOM, spelled as a constant so maxBytes can
	// be one too (len of a var slice is not constant).
	bomBytes = 3

	// maxBytes is the largest input that could still be within maxRunes: every
	// rune costs at most utf8.UTFMax bytes, plus a possible leading BOM. The
	// bound is enforced on the byte count *before* the input is decoded, so an
	// oversize file is rejected instead of being materialised first.
	maxBytes = maxRunes*utf8.UTFMax + bomBytes
)

// utf8BOM is the 3-byte UTF-8 byte-order mark, matched on raw bytes (a
// literal BOM rune is illegal in Go source).
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Sentinel causes; callers branch with errors.Is to show a precise hint.
var (
	ErrEmpty    = errors.New("codetext: input is empty or whitespace-only")
	ErrBinary   = errors.New("codetext: input is not valid UTF-8 text")
	ErrTooLarge = errors.New("codetext: input exceeds the size limit")
	ErrControl  = errors.New("codetext: input contains control characters")
)

// Load reads from a file path, or from stdin when path == "-", then
// normalizes and validates. The returned string is ready to use as a Code
// target verbatim.
func Load(path string) (string, error) {
	if path == "-" {
		return loadReader(os.Stdin)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("codetext: open %s: %w", path, err)
	}
	defer f.Close()
	return loadReader(f)
}

// Normalize applies the exact same BOM/binary/CRLF/trim/empty/cap pipeline as
// Load to an already in-memory string (no file I/O). It exists so an in-app
// paste is validated by identical rules/caps as `--text` — Load and Normalize
// share the single normalize core below, so the rules cannot diverge.
func Normalize(s string) (string, error) {
	return normalize([]byte(s))
}

// loadReader is the FS-independent reader core: read a bounded prefix, then
// normalize.
//
// The bound is the point of this function. Reading to EOF means a 400 MiB file
// is allocated before the rune cap can reject it, and a stream that never ends
// — /dev/zero, or `--text -` on a pipe whose writer never closes — is read
// until the machine gives out. One byte past maxBytes is read so that an
// oversize input is an error rather than a silent truncation.
func loadReader(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxBytes+1))
	if err != nil {
		return "", fmt.Errorf("codetext: read: %w", err)
	}
	return normalize(raw)
}

// normalize is the single post-read pipeline shared by Load (via loadReader)
// and Normalize: strip a leading UTF-8 BOM, reject binary, CRLF→LF, trim one
// trailing newline, then enforce the empty/rune/line rules. Operating on the
// raw bytes keeps the BOM check byte-level for both entry points.
func normalize(raw []byte) (string, error) {
	// Byte bound first: it is the only check that can be made without walking
	// or copying the input, and it makes the rune cap unreachable by anything
	// that would cost real memory to decode.
	if len(raw) > maxBytes {
		return "", fmt.Errorf("%w (max %d runes)", ErrTooLarge, maxRunes)
	}

	raw = bytes.TrimPrefix(raw, utf8BOM)

	// Binary guard: invalid UTF-8 or a NUL byte means this is not text.
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return "", ErrBinary
	}

	s := strings.ReplaceAll(string(raw), "\r\n", "\n")

	// CRLF has already become LF above, so what is left here is a bare CR, a
	// backspace, an ESC starting an ANSI sequence, or similar. Code mode wants
	// an exact rune match and the typing screen only forwards runes that a key
	// press produced as text: ESC is bound to Back and 0x08 to Backspace, so
	// those runes can never be typed. A snippet containing one would be a test
	// the user can neither finish nor leave. LF and TAB — the two this package
	// documents as intentional — are the only controls that survive.
	if r, found := firstControlRune(s); found {
		return "", fmt.Errorf("%w (found %q)", ErrControl, r)
	}

	// Trim exactly one trailing newline (not all of them).
	s = strings.TrimSuffix(s, "\n")

	if strings.TrimSpace(s) == "" {
		return "", ErrEmpty
	}
	if utf8.RuneCountInString(s) > maxRunes {
		return "", fmt.Errorf("%w (max %d runes)", ErrTooLarge, maxRunes)
	}
	if strings.Count(s, "\n")+1 > maxLines {
		return "", fmt.Errorf("%w (max %d lines)", ErrTooLarge, maxLines)
	}
	return s, nil
}

// firstControlRune returns the first control rune in s that is not LF or TAB,
// and whether one was found. NUL is reported here too, but the binary guard
// runs earlier and claims it as ErrBinary.
func firstControlRune(s string) (rune, bool) {
	for _, r := range s {
		if r == '\n' || r == '\t' {
			continue
		}
		if unicode.IsControl(r) {
			return r, true
		}
	}
	return 0, false
}
