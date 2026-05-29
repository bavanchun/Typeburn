---
phase: 3
title: Selfpath and Atomic Replace
status: completed
priority: P1
effort: 1d
dependencies:
  - 2
---

# Phase 3: Selfpath and Atomic Replace

## Overview

Two pieces: (a) classify the running binary's install channel so managed
installs can be refused, and (b) atomically replace the running binary with the
extracted one — `os.Rename` over the target on unix, the rename-self-aside dance
on windows.

## Requirements

- Functional:
  - `classifyInstall(execPath string) Install` → `InstallSelfManaged` |
    `InstallHomebrew` | `InstallGo`. Resolve symlinks first (`filepath.EvalSymlinks`).
    Homebrew: path under brew prefix or contains `/Cellar/`. Go: under `go env
    GOBIN` or `<GOPATH>/bin`. Else self-managed.
  - `instructionFor(Install) string` → the channel-correct upgrade command
    (`brew upgrade typeburn` / `go install github.com/bavanchun/Typeburn@latest`).
  - `replaceBinary(target, newBin string) error` — atomic swap:
    - unix (`replace_unix.go`): `O_EXCL` temp in target's dir → `chmod 0o755`
      (reapply the original file mode; do NOT chown — see RT fix #1) → `lstat`
      target and refuse if symlink/non-regular → `os.Rename(newTmp, target)`.
    - windows (`replace_windows.go`): `os.Rename(target, target+".old")` →
      `os.Rename(newBin, target)`; best-effort remove `.old` (may be locked).
  - Must work when target dir is the same filesystem as temp (rename atomicity);
    on cross-device, fall back to copy+fsync+rename within the target dir.
- Non-functional: build-tagged platform files; stdlib only; files <200 LOC.

## Architecture

- `internal/update/selfpath.go` (+ test) — pure path classification, table-driven
  testable by injecting candidate paths + a fake env lookup (`gobin`, `gopath`).
- `internal/update/replace_unix.go` (`//go:build !windows`) and
  `replace_windows.go` (`//go:build windows`) — same `replaceBinary` signature.
- `replace_unix_test.go` runs on unix CI; the windows path is covered by a
  shared `replace_common.go` helper (temp-write + verify) plus a `//go:build
  windows` test where feasible. Keep the platform-specific surface minimal.

## Related Code Files

- Create: `internal/update/selfpath.go`, `internal/update/selfpath_test.go`
- Create: `internal/update/replace_unix.go`, `internal/update/replace_windows.go`
- Create: `internal/update/replace_unix_test.go`
- Read for context: `internal/update/check.go` (`releaseURLPrefix`, repo consts)

## Implementation Steps (TDD — tests first)

1. **Write `selfpath_test.go`:** table tests — `/home/u/.local/bin/typeburn`
   → SelfManaged; `/opt/homebrew/Cellar/typeburn/2.2.0/bin/typeburn` and
   `/usr/local/Cellar/...` → Homebrew; `<gobin>/Typeburn` and
   `<gopath>/bin/Typeburn` → Go. Inject env lookups; assert `instructionFor`.
2. Implement `selfpath.go` to green.
3. **Write `replace_unix_test.go`:** create a temp "installed" file, an extracted
   replacement, call `replaceBinary`, assert contents swapped + mode `0o755` +
   no leftover temp; mismatched-dir (cross-device simulated) falls back cleanly;
   read-only dir → permission error surfaced.
4. Implement `replace_unix.go` and `replace_windows.go` to green (unix tested in
   CI; windows compiled + logic-reviewed, exercised via the windows CI build).
5. `gofmt`, `go vet`, `go test ./internal/update/ -race -count=1`.

## Success Criteria

- [ ] Homebrew + Go installs classified correctly via symlink-resolved paths.
- [ ] `replaceBinary` swaps contents atomically; preserves exec mode; cleans temps.
- [ ] Read-only/permission-denied target → clear error, original untouched.
- [ ] Windows path compiles and follows the rename-aside contract.
- [ ] Tests pass under `-race`; files <200 LOC; no UI deps.

## Risk Assessment

- **Windows running-exe lock:** can't delete a running exe; rename-aside is the
  standard workaround. `.old` cleanup is best-effort on next run.
- **Symlinked brew installs (false negative):** `EvalSymlinks` before classify;
  also check the resolved real path, not the symlink in `~/.local/bin`.
- **Cross-device rename (EXDEV):** Phase 2 now writes into the target dir, so the
  unix path is a pure rename; EXDEV becomes genuinely rare. If it still occurs,
  copy to a *temp name in the target dir* then atomic-rename — never copy over
  `target` directly.
- **Mode loss:** stat the original first; reapply mode after rename. Do NOT chown.

## Red Team Adjustments (applied 2026-05-29)

1. **[Critical] TOCTOU / symlink-race (S3):** the swap temp MUST be created with
   `os.OpenFile(..., O_CREATE|O_EXCL|O_WRONLY, 0o600)` + a PID/random suffix in
   the target dir (mirror `cache.go:108-114`). Before replacing, `lstat` the
   target and REFUSE if it is a symlink or non-regular file (mirror
   `install.sh:143`). **Drop the "preserve owner"/chown idea entirely** — it is a
   root-path escalation vector and impossible without root anyway.
2. **[Critical] Windows rollback (F1):** on second-rename failure, attempt
   `os.Rename(target+".old", target)` to restore the working exe before returning
   the error; use a randomized `.old` suffix; on next-run startup, if `target` is
   missing but a `.old` is present, restore it. Add a post-swap sha256 re-check of
   the installed file.
3. **[High] Pre-flight writability (F4/A7):** export a `canWrite(dir string) bool`
   (create+remove an `O_EXCL` probe file in the target dir). Phase 4 calls it
   right after classify and BEFORE download, so a root-owned `/usr/local/bin`
   self-managed install fails fast instead of after a 5 MB download.
4. **[High] Classification completeness (A3/F6):**
   - Handle `GOBIN` unset → default to `<GOPATH>/bin` (or `go env GOPATH`); test
     the empty-`GOBIN` case explicitly.
   - Resolve symlinks with `EvalSymlinks` before classifying.
   - The replaced binary MUST keep the *running* exe's basename from
     `os.Executable()` (e.g. capital `Typeburn` for `go install`), NOT the
     lowercase archive member name; compare case-insensitively on case-insensitive
     filesystems (macOS APFS). (Module path is capital-`T`; releases ship
     lowercase `typeburn`.)
   - scoop/chocolatey paths aren't specially classified → treated self-managed,
     but the writability probe + lstat guard keep that safe; document as such.
5. **[Medium] Concurrent-update lock (F5):** acquire an advisory lock (an
   `O_EXCL` lockfile in the target dir) for the duration of replace; if held,
   exit "another update is in progress". Lightweight — the Windows rename-aside
   path is non-idempotent under concurrency, so this is correctness, not gold-plating.
