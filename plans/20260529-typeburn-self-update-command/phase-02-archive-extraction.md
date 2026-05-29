---
phase: 2
title: Archive Extraction
status: completed
priority: P1
effort: 0.5d
dependencies:
  - 1
---

# Phase 2: Archive Extraction

## Overview

Extract exactly the expected binary member (`typeburn` on unix, `typeburn.exe`
on windows) from the verified archive — `tar.gz` for unix, `zip` for windows —
with a path-traversal guard. Output: the extracted binary written to a temp file
(executable mode), ready for Phase 3's atomic swap.

## Requirements

- Functional:
  - `extractBinary(archivePath, wantMember, destDir string) (tmpBinPath string, err error)`
    dispatching on archive type (tar.gz vs zip) by extension or explicit arg.
    Writes the extracted binary into `destDir` (= the install dir, per RT fix #2),
    not the OS temp dir.
  - Extract ONLY `wantMember`; ignore `README.md`/`LICENSE`/`CHANGELOG.md` that
    GoReleaser bundles. Error if the member is absent.
  - Path-traversal guard: reject any archive entry whose cleaned name escapes the
    extraction root or contains `..` (defense even though we target one member).
  - Write the member to a temp file with `0o755`; return its path.
- Non-functional: stdlib only (`archive/tar`, `compress/gzip`, `archive/zip`);
  files <200 LOC.

## Architecture

New file `internal/update/archive.go` (+ test). Two unexported helpers
`extractTarGz` / `extractZip` behind `extractBinary`. Bounded decompression: cap
the total decompressed bytes (defend against decompression bombs) using the same
order of magnitude as the download cap.

## Related Code Files

- Create: `internal/update/archive.go`, `internal/update/archive_test.go`
- Read for context: `internal/update/download.go` (Phase 1; temp-file conventions)

## Implementation Steps (TDD — tests first)

1. **Write `archive_test.go`:** build in-memory fixtures — a `tar.gz` containing
   `typeburn` (+ a `README.md` decoy) and a `zip` containing `typeburn.exe`.
   Assert: correct member extracted, mode `0o755`, decoy ignored; missing member
   → error; an entry named `../evil` → traversal error; oversized decompressed
   stream → error.
2. Implement `extractBinary` + `extractTarGz` + `extractZip` to green.
3. `gofmt`, `go vet`, `go test ./internal/update/ -race -count=1`.

## Success Criteria

- [ ] Extracts the binary member from both tar.gz and zip; ignores bundled docs.
- [ ] Path-traversal and oversize-decompression attempts are rejected.
- [ ] Extracted file is `0o755`; missing member errors clearly.
- [ ] Tests pass under `-race`; files <200 LOC; no UI deps.

## Risk Assessment

- **Path traversal / zip-slip:** clean + verify each entry name; never join raw.
- **Decompression bomb:** cap decompressed size via `io.LimitReader`.
- **Member name mismatch:** `typeburn` vs `typeburn.exe` — derive from goos, pass
  `wantMember` explicitly from the caller (Phase 4 knows the running os).

## Red Team Adjustments (applied 2026-05-29)

1. **[High] Symlink/non-regular member (S4):** the traversal guard is not enough.
   Reject any member whose type is not a regular file — `hdr.Typeflag !=
   tar.TypeReg` (tar) or a zip entry with `os.ModeSymlink`/non-zero mode-type
   bits. An unsigned archive (in scope) can ship `typeburn` as a symlink that
   `chmod 0o755` + swap would follow. Mirrors `install.sh:143-144`.
2. **[Critical] Extract into the target's directory (F2/EXDEV):** `extractBinary`
   takes a `destDir` arg and writes the temp binary into `filepath.Dir(target)`
   (the install dir), NOT the OS temp dir. This makes the Phase-3 swap a pure
   same-filesystem `os.Rename` and avoids the non-atomic cross-device copy on
   every run when `/tmp` is a separate FS. Signature becomes
   `extractBinary(archivePath, wantMember, destDir string) (tmpBinPath, error)`.
3. **[Medium] Size-cap checkpoint (A6):** after this phase compiles, run
   `make size-check` as a checkpoint — `archive/zip`+`compress/flate`+`gzip` are
   newly linked and headroom over the 10 MiB cap (`Makefile:16`) is ~17%. If it
   regresses here, surface it now, not at Phase 5.
