---
phase: 2
title: Update progress feedback
status: completed
priority: P2
effort: 2h
dependencies:
  - 1
---

# Phase 2: Update progress feedback

## Overview

`typeburn update` prints `updating X → Y ...` then blocks silently inside
`update.Apply` while it downloads ~9 MB, verifies SHA-256, extracts, and swaps —
the terminal looks frozen. Add a nil-safe progress reporter so the CLI prints
`downloading… / verifying… / installing…` step lines.

## Requirements

- Functional: during a real `update`, stdout shows distinct download, verify, and
  install stage lines before the final `updated X → Y` line.
- Non-functional: `internal/update` stays stdlib-only (no UI deps); reporter is a
  plain `func(Stage)`; nil reporter = silent (preserves existing callers/tests).

## Architecture

- **Reporter type** (in `internal/update`, e.g. `apply.go` or a small
  `progress.go`):
  ```go
  type Stage int
  const ( StageDownloading Stage = iota; StageVerifying; StageInstalling )
  func (s Stage) String() string // "downloading"/"verifying"/"installing"
  ```
- **Thread the reporter** as a final nil-safe param:
  - `Apply(ctx, currentVer, tag, execPath, goos, goarch, report func(Stage))`
  - `downloadVerified(ctx, rawTag, goos, goarch, destDir, report func(Stage))`
  - Emit: `report(StageDownloading)` before the archive download, then
    `report(StageVerifying)` before `verifySHA256`, then `report(StageInstalling)`
    in `Apply` before `extractBinary`/`replaceBinary`. Guard every call:
    `if report != nil { report(...) }`.
- **Rejected:** bundling params into an `Options` struct — more churn across
  callers/tests for no behavior gain; the single nil-safe param is KISS.
- **CLI wiring** (`cmd_update.go`): replace the lone `updating X → Y ...` print
  with a reporter that prints `<stage>...` step lines; keep the final
  `updated X → Y. restart…` line. Plain `fmt.Fprintln` — no spinner (non-TUI).
- **File-size watch:** `cmd_update.go` is 161 LOC; if wiring pushes it toward 200,
  extract the reporter/printing into a helper file.

## Related Code Files

- Create: `internal/update/progress.go` (Stage type) — or inline in `apply.go`
- Modify: `internal/update/apply.go`, `internal/update/download.go`
- Modify: `internal/cli/cmd_update.go`
- Modify: `internal/update/apply_test.go`, `internal/cli/cmd_update_test.go`
- Update call sites: any existing `Apply(...)`/`downloadVerified(...)` callers +
  the `applyFn` signature/var in `cmd_update.go` and its test override.

## Implementation Steps (TDD)

1. **Red (unit):** in `apply_test.go`, add a test that passes a recording
   reporter to `Apply` (using existing httptest fixtures) and asserts the stage
   sequence == `[downloading, verifying, installing]` on success. Fails to
   compile/run (no param yet).
2. **Green:** add `Stage`, thread the nil-safe `report` param through `Apply` and
   `downloadVerified`, emit at the three boundaries. Update the `applyFn` type in
   `cmd_update.go` + its `setApplyFn` test override to the new signature.
3. **Red (CLI):** in `cmd_update_test.go`, assert stdout contains the step lines
   (`downloading`, `verifying`, `installing`) on a successful stubbed update.
4. **Green:** wire the CLI reporter to print `<stage>...`. Verify nil-reporter
   path (any other caller / `--check`) unaffected.
5. Run `go test ./internal/update/ ./internal/cli/ -race -count=1` → green.

## Success Criteria

- [ ] `Apply` calls the reporter `downloading → verifying → installing` on success.
- [ ] Nil reporter = silent; pre-existing tests pass unchanged in behavior.
- [ ] `typeburn update` stdout shows the three stage lines + final updated line.
- [ ] No `internal/update` UI imports; all files <200 LOC.
- [ ] `go test ./... -race`, `go vet`, `gofmt -l` clean.

## Risk Assessment

- Signature change ripples to `applyFn`/`setApplyFn` in `cmd_update.go` + tests —
  enumerate all `Apply(`/`downloadVerified(` call sites first (grep) so none is
  missed.
- Reporter must be nil-safe to avoid panicking any caller that passes nil.
- Keep stage emission OUTSIDE the verify/replace integrity logic — reporting must
  not alter control flow or error paths.
