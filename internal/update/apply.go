package update

import (
	"context"
	"os"
	"path/filepath"
)

// Outcome reports the version transition an Apply performed.
type Outcome struct {
	From string
	To   string
}

// Apply downloads the release identified by tag, verifies its checksum, extracts
// the platform binary, and atomically swaps it over execPath. tag MUST come from
// a live Check(force=true) Result (never the on-disk cache) so a poisoned cache
// cannot redirect the download. execPath must be the resolved path of the running
// binary; its directory is where the temp download, extraction, and atomic rename
// all happen, so the rename stays same-filesystem.
//
// Integrity rests on TLS + the published sha256 checksums — not signatures; this
// detects corruption and truncation, not a compromised release host.
func Apply(ctx context.Context, currentVer, tag, execPath, goos, goarch string) (Outcome, error) {
	dir := filepath.Dir(execPath)

	release, err := acquireUpdateLock(dir)
	if err != nil {
		return Outcome{}, err
	}
	defer release()

	archivePath, err := downloadVerified(ctx, tag, goos, goarch, dir)
	if err != nil {
		return Outcome{}, err
	}
	defer cleanup(archivePath)

	member := binaryMember(goos)
	newBin, err := extractBinary(archivePath, member, dir)
	if err != nil {
		return Outcome{}, err
	}
	defer cleanup(newBin)

	if err := replaceBinary(execPath, newBin); err != nil {
		return Outcome{}, err
	}
	return Outcome{From: currentVer, To: tag}, nil
}

// binaryMember is the binary filename inside a release archive: GoReleaser pins
// it to the lowercase "typeburn", with the ".exe" suffix on Windows.
func binaryMember(goos string) string {
	if goos == "windows" {
		return "typeburn.exe"
	}
	return "typeburn"
}

func cleanup(path string) {
	if path != "" {
		_ = os.Remove(path)
	}
}
