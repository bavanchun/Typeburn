package update

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// decompressCap bounds the decompressed binary size (defends a decompression
// bomb). Same order of magnitude as the download cap.
const decompressCap = 60 << 20 // 60 MiB

// extractBinary extracts exactly wantMember from the archive at archivePath and
// writes it (mode 0o755) into destDir, returning the written path. destDir is
// the install directory so the atomic-replace swap is a same-filesystem rename. Type
// is chosen by archivePath extension: ".zip" → zip, else tar.gz. Bundled docs
// (README/LICENSE/CHANGELOG) are ignored; a missing member is an error.
func extractBinary(archivePath, wantMember, destDir string) (string, error) {
	dest := filepath.Join(destDir, wantMember+".new")
	if strings.HasSuffix(strings.ToLower(archivePath), ".zip") {
		return dest, extractZip(archivePath, wantMember, dest)
	}
	return dest, extractTarGz(archivePath, wantMember, dest)
}

// safeMember reports whether name is the wanted member and not a traversal or
// nested path. Only the exact top-level member is accepted.
func safeMember(name, want string) bool {
	clean := path.Clean("/" + strings.ReplaceAll(name, `\`, "/"))
	return clean == "/"+want
}

// writeMember copies src into dest (created O_EXCL, 0o755), bounded by
// decompressCap. Removes the partial file on failure.
func writeMember(src io.Reader, dest string) error {
	f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o755)
	if err != nil {
		return fmt.Errorf("update: create extracted binary: %w", err)
	}
	n, copyErr := io.Copy(f, io.LimitReader(src, decompressCap+1))
	closeErr := f.Close()
	if copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		_ = os.Remove(dest)
		return fmt.Errorf("update: write extracted binary: %w", copyErr)
	}
	if n > decompressCap {
		_ = os.Remove(dest)
		return fmt.Errorf("update: extracted binary exceeds %d-byte cap", decompressCap)
	}
	return nil
}

func extractTarGz(archivePath, wantMember, dest string) error {
	f, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("update: open archive: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("update: gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("update: tar: %w", err)
		}
		if !safeMember(hdr.Name, wantMember) {
			continue
		}
		if hdr.Typeflag != tar.TypeReg {
			return fmt.Errorf("update: archive member %q is not a regular file", wantMember)
		}
		return writeMember(tr, dest)
	}
	return fmt.Errorf("update: member %q not found in archive", wantMember)
}

func extractZip(archivePath, wantMember, dest string) error {
	zr, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("update: open zip: %w", err)
	}
	defer zr.Close()

	for _, zf := range zr.File {
		if !safeMember(zf.Name, wantMember) {
			continue
		}
		if !zf.Mode().IsRegular() {
			return fmt.Errorf("update: archive member %q is not a regular file", wantMember)
		}
		rc, err := zf.Open()
		if err != nil {
			return fmt.Errorf("update: open zip member: %w", err)
		}
		defer rc.Close()
		return writeMember(rc, dest)
	}
	return fmt.Errorf("update: member %q not found in archive", wantMember)
}
