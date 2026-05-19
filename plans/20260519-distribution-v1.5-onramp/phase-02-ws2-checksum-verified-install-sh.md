---
phase: 2
title: "WS2 Hardened install.sh + GoReleaser determinism"
status: completed
priority: P1
effort: "5-7h"
dependencies: []
---

# Phase 2: WS2 Hardened install.sh + GoReleaser determinism

## Overview

Two coupled deliverables: (a) pin `.goreleaser.yaml` so artifact names +
binary name are **deterministic** (prereq for both install.sh and the P3
cask); (b) a hardened POSIX `install.sh` that resolves the correct release,
verifies sha256, rejects prereleases, and installs atomically. Independent of
P1 (red-team M1) — runs in parallel.

## Requirements
- Functional: `curl -fsSL <raw>/install.sh | sh` installs a runnable lowercase
  `typeburn` on darwin/linux × amd64/arm64, checksum-verified, from the correct
  versioned asset, refusing prerelease/`-rc`/`-test` tags.
- Non-functional: POSIX `sh`; explicit failure on every download (no reliance
  on `set -e` through pipes); atomic install; idempotent + crash-safe re-run.
- Out: Windows (sh-only; manual zip), `/usr/local/bin`, sudo.

## Key Insight (red-team C3, H2 — verified)
`.goreleaser.yaml` has **no `archives.name_template`, no `project_name`, no
`builds.binary`** (`grep -n 'name_template\|project_name\|binary:'` → only the
checksum block at `:50`). GoReleaser v2 default archive name includes a
`_{{ .Version }}` segment and the binary name is the derived (often lowercased)
project name. Any hardcoded `Typeburn_<os>_<arch>.tar.gz` guess 404s; any
`bin` rename guess may break. **Fix the root cause: pin determinism in
`.goreleaser.yaml` first**, then install.sh + cask reference exact known strings.

## Architecture
install.sh trust boundary stated honestly (red-team M2): checksum verification
defends **MITM / corrupted/truncated download**, NOT a compromised GitHub
release (`checksums.txt` is unsigned and co-hosted — same trust root). Document
the real boundary; do not claim it makes `curl|sh` "safe". TDD: offline harness
+ `shellcheck` authored first (RED→GREEN), fixtures for every failure mode.

## Related Code Files
- Modify: `.goreleaser.yaml` — add `builds.binary: typeburn`,
  `project_name: typeburn`, explicit `archives.name_template`
  (e.g. `typeburn_{{ .Version }}_{{ .Os }}_{{ .Arch }}`)
- Create: `install.sh` (repo root)
- Create: `scripts/test-install-sh.sh` — offline harness
- Modify: `README.md` (Installation §: one-liner + non-piped audit path +
  honest trust-boundary note) — coordinate numeral with P1/P4
- Modify: `.github/workflows/ci.yml` (`shellcheck install.sh` + harness;
  add `goreleaser check` step)

## Implementation Steps (TDD)
1. **Pin GoReleaser determinism:** add `project_name: typeburn`,
   `builds.binary: typeburn`, `archives.name_template:
   "typeburn_{{ .Version }}_{{ .Os }}_{{ .Arch }}"`. Run `make snapshot`,
   `tar tzf dist/<archive>` → record EXACT archive names + that the binary
   inside is `typeburn`. `goreleaser check` passes.
2. **Tests first** (`scripts/test-install-sh.sh`), all RED — fixtures:
   (a) os/arch → exact asset-name table (from step 1 real names),
   (b) checksum mismatch → non-zero exit, nothing written,
   (c) unsupported platform (windows/freebsd) → exit 1 + clear msg,
   (d) truncated/empty download → abort (no partial install),
   (e) `$BIN_DIR` absent → still succeeds (`mkdir -p`),
   (f) resolved tag matches `-rc`/`-test`/prerelease → refuse,
   (g) archive member is symlink/`..`/non-regular → reject,
   (h) interrupted before final move → prior binary intact.
3. Implement `install.sh`: `set -eu`; explicit exit-status check after every
   `curl` (no pipe-only reliance — POSIX has no `pipefail`); `tmp=$(mktemp -d)`
   `chmod 700`; cleanup trap; resolve latest **non-prerelease** tag (parse
   GitHub API, reject `-rc`/`-test`/prerelease; allow `VERSION=` override);
   download archive + `checksums.txt` to tmp; verify size>0 then sha256
   (`sha256sum` or `shasum -a 256`); extract to tmp, assert expected member is
   a single regular file (reject symlink/dir/`..`); `mkdir -p "$BIN_DIR"`
   (default `~/.local/bin`); `mv -f tmp/typeburn "$BIN_DIR/typeburn"` (atomic,
   same fs) as the FINAL step; PATH membership AND precedence check (warn if a
   different `typeburn` shadows / is shadowed; print exact `export PATH`).
4. Harness GREEN; `shellcheck install.sh` zero warnings.
5. README: one-liner, non-piped audit path (download→inspect→run), honest
   trust-boundary note (defends MITM/corruption, not release compromise).
6. CI: `shellcheck install.sh` + harness + `goreleaser check` (no network).

## Todo List
- [x] `.goreleaser.yaml` determinism pinned; real names recorded; `goreleaser check` green
- [x] Offline harness written, all fixture cases RED first (9 real fails)
- [x] `install.sh` implemented, harness GREEN (14/14)
- [x] `shellcheck` zero warnings (verified across 0.10.0 + 0.11.0 + CI)
- [x] README Installation § updated (honest trust boundary)
- [x] CI: shellcheck + harness + goreleaser check (`installer` job)

## Success Criteria
- [x] Pipe-install yields working `typeburn --version` (clean-container, P4)
- [x] Tampered/truncated/symlink fixtures all abort non-zero, nothing written
- [x] Prerelease/`-rc`/`-test` tag refused (harness + real-release test)
- [x] `$BIN_DIR` absent → install succeeds; prior binary byte-identical on fail
- [x] CI green incl. new steps; `assets==7` untouched this phase

## Risk Assessment
- **Wrong asset name (C3)** → root-caused by pinning names in step 1; fixtures
  use the REAL recorded names, not guesses.
- **Pipe failure silent (H8)** → explicit per-download exit checks + size guard.
- **Symlink/tmp/atomicity (H8)** → mktemp 700, member validation, final `mv`.
- **`releases/latest` poisoned by dry-run (H3)** → install.sh prerelease-reject
  guard is defense-in-depth alongside P3's `release.prerelease: auto`.
- **GitHub API rate-limit** → `VERSION=` override documented.
- **Trust-boundary overclaim (M2)** → docs state the real boundary explicitly.
