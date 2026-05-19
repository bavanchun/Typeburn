// Package codetext loads and normalizes user-supplied text/code for the
// Code typing mode. It is the I/O boundary for `--text <file>` / `--text -`
// so the pure-logic packages (words, typing) stay free of file/stdin access.
//
// Normalization (full-literal-safe): strip a leading UTF-8 BOM, convert CRLF
// to LF, and trim exactly one trailing newline (so the snippet's final line
// needs no closing Enter). Tabs, interior blank lines, and indentation are
// preserved verbatim — Code mode requires the user to type them.
package codetext

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

// Caps. A snippet over either bound is rejected (not truncated) so the
// renderer/viewport never face pathological input and the caller can show a
// clear reason instead.
const (
	maxRunes = 10000
	maxLines = 500
)

// utf8BOM is the 3-byte UTF-8 byte-order mark, matched on raw bytes (a
// literal BOM rune is illegal in Go source).
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// Sentinel causes; callers branch with errors.Is to show a precise hint.
var (
	ErrEmpty    = errors.New("codetext: input is empty or whitespace-only")
	ErrBinary   = errors.New("codetext: input is not valid UTF-8 text")
	ErrTooLarge = errors.New("codetext: input exceeds the size limit")
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

// loadReader is the FS-independent core: read all, normalize, validate.
func loadReader(r io.Reader) (string, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return "", fmt.Errorf("codetext: read: %w", err)
	}

	raw = bytes.TrimPrefix(raw, utf8BOM)

	// Binary guard: invalid UTF-8 or a NUL byte means this is not text.
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return "", ErrBinary
	}

	s := strings.ReplaceAll(string(raw), "\r\n", "\n")

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
