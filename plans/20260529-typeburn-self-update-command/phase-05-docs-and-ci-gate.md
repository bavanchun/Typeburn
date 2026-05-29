---
phase: 5
title: "Docs and CI Gate"
status: completed
priority: P2
effort: "0.25d"
dependencies: [4]
---

# Phase 5: Docs and CI Gate

## Overview

Document the new command, record the security trust-boundary note, update the
changelog/roadmap, and run the full CI gate green before the PR.

## Requirements

- Functional:
  - `README.md`: add `typeburn update` to the install/upgrade section — what it
    does, the managed-install refusal behavior, and the trust-boundary caveat
    (unsigned binaries; equivalent to `curl install.sh | sh`).
  - `docs/codebase-summary.md`: document the new `internal/update` files
    (download/verify/archive/selfpath/replace/apply) + the `cmd_update.go` wiring.
  - `docs/project-roadmap.md`: add the shipped/feature entry.
  - `CHANGELOG.md`: `### Added` entry under `[Unreleased]` for `typeburn update`.
  - `SECURITY.md`: note that self-update inherits the existing checksum-only
    (unsigned) trust model; signing remains deferred.
  - Command `--help` text states the trust caveat succinctly.
- Non-functional: docs are source of truth; keep concise.

## Architecture

Docs-only phase plus the final verification gate. No code beyond help text /
comments already added in Phase 4.

## Related Code Files

- Modify: `README.md`, `docs/codebase-summary.md`, `docs/project-roadmap.md`,
  `CHANGELOG.md`, `SECURITY.md`
- Read for context: existing install section in `README.md`, the
  `cmd_version.go` upgrade-instructions block (consistency of wording)

## Implementation Steps

1. Update README install/upgrade section + the four docs files + SECURITY note.
2. Add CHANGELOG `[Unreleased] → Added` entry.
3. Run the full CI gate locally:
   - `gofmt -l .` (must be empty)
   - `go vet ./...`
   - `go test ./... -race -count=1`
   - `make size-check` (binary-size cap — verify the updater additions don't
     blow the cap)
4. `make build && ./bin/typeburn update --help` smoke check.

## Success Criteria

- [ ] All five docs reflect the new command + trust caveat.
- [ ] `gofmt -l .` empty; `go vet ./...` clean; `go test ./... -race` green.
- [ ] `make size-check` passes (size cap respected).
- [ ] `typeburn update --help` renders the caveat.

## Risk Assessment

- **Binary-size cap:** archive/zip/tar + crypto are mostly already linked
  (stdlib); net growth should be small, but `size-check` is the gate — if it
  fails, that's a real finding, not a number to bump silently.
- **Doc drift:** keep README upgrade wording consistent with
  `cmd_version.go`'s existing `--check-update` instructions.
