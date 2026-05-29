# Brainstorm Summary — `typeburn update` Self-Update Command

**Date:** 2026-05-29
**Status:** Design approved; ready for `/ck:plan --tdd`

## Problem

No `update` subcommand exists. Typing `typeburn update` hits cobra's
arbitrary-args root, treats `update` as unknown, and **silently launches the
TUI** (`root.go` RunE) instead of updating or erroring. User wants a one-command
in-place upgrade of the installed binary to the latest release.

## Existing infra (reuse, don't rebuild)

- `internal/update`: `Check()` (24h cache, dev-version skip, prerelease guard),
  `FetchLatest()` (GitHub Releases API, 1.5s timeout, redirect-blocked, 64KB cap),
  `Compare()` semver, `Result`. Detection only — never downloads/installs.
- `typeburn version --check-update` already detects + prints upgrade instructions.
- `releaseURLPrefix` official-repo URL guard already exists.
- GoReleaser publishes 7 assets: `typeburn_<ver-no-v>_<os>_<arch>.{tar.gz|zip}`
  (windows=zip) across linux/darwin/windows × amd64/arm64, + `checksums.txt`.

## Decisions (user-confirmed)

| Axis | Decision |
|------|----------|
| Mechanism | **Self-replacing binary** (download → verify → atomic swap) |
| Platforms | **Linux + macOS + Windows** |
| Managed installs (brew / `go install`) | **Refuse + instruct** (don't overwrite) |
| Dependencies | **Hand-roll with stdlib** (no third-party self-update lib) |

Net effect = hybrid: self-replace where the binary is self-managed
(install.sh `~/.local/bin`, manual); refuse + print the right command where a
package manager owns it.

## Approach (chosen)

**Command surface (YAGNI):**
- `typeburn update` — check → confirm → download → verify → swap
- `typeburn update --yes/-y` — skip confirm (CI/scripts)
- `typeburn update --check` — detect only (thin alias over `update.Check`)
- No `--version` pin this round (install.sh covers pinned installs).

**Flow:**
1. `version.Resolve()`; dev/unknown/pseudo → refuse with install instructions.
2. `update.Check(force=true)`; not newer → "already latest", exit 0.
3. Managed-install guard: `os.Executable()` → resolve symlinks → classify
   (brew prefix / `/Cellar/`; `go env GOBIN`/`GOPATH/bin`). Managed → refuse +
   channel-correct command, distinct exit code.
4. Self-managed → confirm (unless `--yes`) → download deterministic asset +
   `checksums.txt`.
5. sha256 verify against the asset's `checksums.txt` line — abort on mismatch.
6. Extract only the expected `typeburn`/`typeburn.exe` member (tar.gz/zip) with
   path-traversal guard.
7. Atomic replace: temp in same dir, preserve +x, `os.Rename` over target.
   Windows: rename running `.exe` aside → rename new in → best-effort `.old`
   cleanup next run.

**Layering / files (<200 LOC each):**
- `internal/update/`: `download.go` (asset URL + size-capped fetch + single
  allowed redirect to `objects.githubusercontent.com`), `verify.go` (checksums
  parse + sha256), `archive.go` (tar.gz/zip single-member + traversal guard),
  `selfpath.go` (managed-install classification),
  `replace_unix.go` / `replace_windows.go` (build-tagged atomic swap).
- `internal/cli/`: `cmd_update.go` (cobra cmd, confirm prompt, progress, exit
  codes) + register in `root.go`. Reuse the `checkFn` test seam.

## Risks

1. **Windows running-exe replacement** — rename-self-aside dance; most edge cases.
2. **Asset-download redirect** to `objects.githubusercontent.com` — must allow
   that one host without reopening the redirect hole `FetchLatest` closed (SSRF).
3. Partial-download cleanup; read-only/permission-denied dirs → clean refuse.
4. Managed-path false-negatives (symlinked brew installs).

## Security caveat (must document)

Trust boundary is **identical to `curl install.sh | sh`**: binaries are unsigned
and `checksums.txt` ships from the same release, so checksum verification defends
a corrupted/truncated *download*, not a compromised *release*. TLS is the real
integrity anchor. Consistent with documented v1 posture (`SECURITY.md`); note it
in `--help` + docs. Code signing stays a deferred v2+ item.

## Out of scope

Auto-update-on-launch, delta updates, rollback command, `--version` pin, signing.

## Suggested phases (for `/ck:plan --tdd`)

1. Asset URL + download + sha256/checksums verify (httptest, table-driven).
2. Archive extraction (tar.gz + zip), traversal + single-member guards.
3. `selfpath` classification + atomic replace (unix + windows, build-tagged).
4. `cmd_update.go` CLI wiring, confirm prompt, `--check`/`--yes`, exit codes,
   register in root, `Decide()` interplay.
5. Docs (README install, codebase-summary, roadmap), CHANGELOG, full CI gate.

## Success criteria

- `typeburn update` on a self-managed install upgrades in place; re-run reports
  "already latest".
- On brew/go-managed install, refuses + prints the correct command, exits non-zero.
- sha256 mismatch aborts without touching the installed binary.
- dev/unknown build refuses cleanly.
- Works on linux/darwin/windows × amd64/arm64.
- No new deps; every Go file <200 LOC; layering preserved; CI gate green.

## Unresolved questions

- Exit-code convention for the managed-install refusal (distinct code vs 0+message)
  — defer to plan phase 4.
- Whether `--check` should fully alias `version --check-update` or stay a thin
  detect-only path — minor; resolve in plan.
