---
phase: 4
title: "CLI Command Wiring"
status: completed
priority: P1
effort: "0.5d"
dependencies: [3]
---

# Phase 4: CLI Command Wiring

## Overview

Wire the orchestration into a cobra `update` subcommand: resolve version, check
for an upgrade, classify install (refuse if managed), confirm, then
download→verify→extract→replace. Register in `root.go` and surface clear exit
codes + human output.

## Requirements

- Functional — `typeburn update`:
  1. `version.Resolve()`. For an *install* attempt, dev/unknown/pseudo → refuse
     with install instructions, non-zero exit. For `--check`, dev/pseudo →
     "no release version, skipped", **exit 0** (mirror `version --check-update`).
  2. `update.Check(ctx, ver, force=true)`; `(nil,nil)` occurs ONLY for dev/pseudo
     (handled in step 1, not here); `!UpgradeAvailable` → "already on latest (vX)",
     exit 0.
  3. `classifyInstall(os.Executable())`; managed → print `instructionFor(...)`,
     exit with a distinct code (e.g. `ExitManagedInstall`).
  4. Self-managed → confirm prompt (default no) unless `--yes/-y`; then run the
     download→verify→extract→replace pipeline; print progress + final
     "updated vX → vY".
  - Flags: `--yes/-y` (skip confirm), `--check` (detect-only; print upgrade
    availability and exit, no install).
  - Errors from any stage → non-zero exit, original binary untouched.
- Non-functional: reuse the `checkFn` test seam; add an injectable `updateFn`
  seam for the download/replace pipeline so the command is unit-testable without
  network or real binary swap; `cmd_update.go` <200 LOC (split helpers if needed).

## Architecture

New `internal/cli/cmd_update.go` defining `newUpdateCmd(e env)`, registered in
`root.go` via `root.AddCommand(newUpdateCmd(e))`. The heavy pipeline lives in
`internal/update` (Phases 1-3) exposed as one entry point, e.g.
`update.Apply(ctx, info, opts) (Outcome, error)`, so the CLI layer only handles
flags, prompt, output, and exit-code mapping. Confirm prompt reads from
`cmd.InOrStdin()` for testability.

Exit codes: extend `internal/cli/exitcodes.go` with `ExitManagedInstall` (and
reuse the generic failure code for pipeline errors). Verify `Decide()` /
root arbitrary-args path is unaffected (the new subcommand is recognized, so it
no longer falls through to the TUI — add a regression test for that).

## Related Code Files

- Create: `internal/cli/cmd_update.go`, `internal/cli/cmd_update_test.go`
- Modify: `internal/cli/root.go` (register subcommand)
- Modify: `internal/cli/exitcodes.go` (add `ExitManagedInstall`)
- Create: `internal/update/apply.go`, `internal/update/apply_test.go`
  (pipeline entry point tying Phases 1-3 together)
- Read for context: `internal/cli/cmd_version.go` (checkFn seam, output style),
  `internal/cli/root.go` (registration + arbitrary-args fallthrough)

## Implementation Steps (TDD — tests first)

1. **Write `apply_test.go`:** drive `update.Apply` end-to-end against an
   httptest release (fake archive + checksums) + a temp "installed" binary;
   assert the temp binary is swapped to the new content; checksum mismatch aborts
   without swapping; managed-install classification short-circuits.
2. Implement `apply.go` (compose download→verify→extract→replace).
3. **Write `cmd_update_test.go`:** inject a fake `updateFn`/`checkFn`; assert:
   `--check` prints availability and does not install; managed install prints the
   right command + `ExitManagedInstall`; dev build refuses; `--yes` skips prompt;
   prompt "n" aborts; success prints "updated". Add a `root_test.go` regression:
   `typeburn update` is parsed as the subcommand (not TUI fallthrough).
4. Implement `cmd_update.go` + register in `root.go` + add exit code.
5. `gofmt`, `go vet`, `go test ./... -race -count=1`.

## Success Criteria

- [ ] `typeburn update` runs the full pipeline on self-managed installs; reports
      "already latest" when current.
- [ ] Managed installs refuse + print the correct command with a distinct exit code.
- [ ] `--check` is detect-only; `--yes` skips the prompt; prompt default is no.
- [ ] dev/unknown build refuses cleanly.
- [ ] `update` is a recognized subcommand (regression test); no TUI fallthrough.
- [ ] Full `./...` suite green under `-race`; files <200 LOC.

## Risk Assessment

- **Testability without network/real swap:** the `updateFn`/`Apply` seam +
  httptest + temp dir keep tests hermetic (mirror existing `checkFn` pattern).
- **Exit-code contract:** document codes in `exitcodes.go`; a managed refusal is
  not a crash — distinct code lets scripts branch.
- **Prompt in non-interactive runs:** default-no + require `--yes` for piped use;
  detect non-tty stdin and refuse rather than hang.

## Red Team Adjustments (applied 2026-05-29)

1. **[High] `--check` dev semantics (F7/A4):** the refuse-first ordering is wrong
   for `--check`. `Check` returns `(nil, nil)` ONLY for dev/pseudo builds
   (`check.go:16-19`). For `--check`, mirror `version --check-update`: dev/pseudo
   → print "no release version, skipped", **exit 0** (`cmd_version.go:150-151`).
   The non-zero refusal applies only to an actual *install* attempt. Fix the
   step-2 description: `!UpgradeAvailable` → "already latest, exit 0"; `(nil,nil)`
   → dev-skip, not "already latest".
2. **[High] Non-tty prompt (A5):** `cmd.InOrStdin()` is an `io.Reader` with no fd,
   so isatty can't read it directly. Type-assert to `*os.File`; if it IS a tty,
   prompt; if it is NOT a tty (pipe/file) and `--yes` was not passed, **refuse**
   with "re-run with --yes for non-interactive use" — never block on a read.
   Tests inject a buffer through the seam and always pass `--yes`.
3. **[High] Live-tag only (F8):** `Apply` derives the download tag from the live
   `Check(force=true)` Result passed in, never re-reads the on-disk cache.
4. **[High] Pre-flight writability (F4):** after `classifyInstall` returns
   self-managed, call `canWrite(filepath.Dir(execPath))` (Phase 3) BEFORE invoking
   the download pipeline; non-writable → fail fast with a sudo/reinstall hint.
5. **Order of operations:** resolve → (install? classify+writability : --check skip)
   → Check(force) → managed-refuse → writability → confirm/--yes → Apply.
