package update

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"
)

// parseChecksums parses GoReleaser's checksums.txt — one "<sha256>  <filename>"
// line per asset — into a filename→lowercase-hex map. Blank lines and malformed
// lines (fewer than two fields) are skipped. It takes the first field as the
// hash and the last as the name, tolerating the two-space GoReleaser separator;
// GoReleaser asset names contain no spaces, so last-field is unambiguous here.
func parseChecksums(data []byte) map[string]string {
	out := make(map[string]string)
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		out[fields[len(fields)-1]] = strings.ToLower(fields[0])
	}
	return out
}

// verifySHA256 streams the file at path and compares its SHA-256 against wantHex
// (case-insensitive). Returns an error on empty want, read failure, or mismatch.
// On any error the caller must treat the file as untrusted.
func verifySHA256(path, wantHex string) error {
	want := strings.ToLower(strings.TrimSpace(wantHex))
	if want == "" {
		return fmt.Errorf("update: no checksum listed for asset")
	}
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("update: open for verify: %w", err)
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return fmt.Errorf("update: hash: %w", err)
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != want {
		return fmt.Errorf("update: checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}
