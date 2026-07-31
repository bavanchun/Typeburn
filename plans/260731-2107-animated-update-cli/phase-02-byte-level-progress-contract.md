---
phase: 2
title: "Byte-Level Progress Contract"
status: pending
priority: P1
effort: "3h"
dependencies: [1]
---

# Phase 2: Byte-Level Progress Contract

## Overview

Replace `internal/update`'s coarse `func(Stage)` reporter with a
`func(Progress)` reporter carrying byte counters, so the UI can render a real
percentage instead of a fabricated one. No UI code in this phase — the existing
plain-text output must stay byte-identical apart from one deliberate addition.

## Requirements

- Functional
  - `update.Progress{Stage, Done, Total}` reported throughout `Apply`.
  - `Done`/`Total` reflect real archive bytes; `Total == 0` means the server
    sent no `Content-Length` (caller must render indeterminate, never divide).
  - New `StageChecksums` for the checksums fetch that `downloadVerified`
    already performs but never reported.
- Non-functional
  - Callback volume bounded — a 4.3 MB download must not produce thousands of
    callbacks.
  - `reportFn == nil` stays legal and silent.
  - No change to control flow, error handling, redirect policy, or integrity
    checks.

## Architecture

### The type

```go
// Progress reports how far an Apply run has advanced. Stage is always
// meaningful. Done/Total are only populated during StageDownloading, and Total
// is 0 when the server sent no Content-Length — render such a run as
// indeterminate rather than computing a ratio.
type Progress struct {
    Stage       Stage
    Done, Total int64
}
```

`Stage` gains `StageChecksums` **first** in the `iota` block, so the natural
ordering `current > s` means "that stage is finished" — the renderer in Phase 3
relies on that ordering for its checklist glyphs. Stage values are ephemeral and
never persisted, so renumbering is safe.

### Counting the bytes

`downloadTo` currently does `io.Copy(f, io.LimitReader(resp.Body, cap+1))`
(`internal/update/download.go:138`). Wrap the **destination writer**, not the
reader, so the returned `n` — which the existing empty-body and cap checks
depend on — keeps its exact meaning.

Throttling is required, not cosmetic: each callback crosses into the Bubble Tea
event loop in Phase 4. Emit when ≥50 ms has passed since the last emission, and
always emit once on completion so the bar reliably lands on 100%.

`internal/update/download.go` is already 193 lines against the repo's 200-line
ceiling, so the writer goes in a **new file**, not into `download.go`.

`downloadTo` gains an `onBytes func(done, total int64)` parameter. The
checksums call passes `nil`; the archive call passes a closure forwarding a
`Progress{Stage: StageDownloading, ...}`. A `resp.ContentLength` of `-1`
normalizes to `0`.

### Signature change

```go
func Apply(ctx context.Context, currentVer, tag, execPath, goos, goarch string,
    reportFn func(Progress)) (Outcome, error)
```

`internal/update` is not importable outside the module, so this is a rename, not
a breaking public API change. Call sites that move: `apply.go`, `download.go`,
and the `applyFn` seam in `internal/cli/cmd_update.go` (plus its test overrides
in `cmd_update_test.go`).

### Keeping plain output stable

`cmd_update.go:127` prints one line per stage. Under the new contract
`StageDownloading` fires repeatedly, so the plain reporter must dedupe on stage
transition:

```go
last := update.Stage(-1)
progress := func(p update.Progress) {
    if p.Stage != last {
        last = p.Stage
        fmt.Fprintf(out, "  %s...\n", p.Stage)
    }
}
```

Net effect on plain output: one **added** line, `  checksums...`, before
`  downloading...`. Everything else unchanged. The existing assertions in
`cmd_update_test.go:213` are `strings.Contains` over `downloading`/`verifying`/
`installing`, so they stay green; the added line gets its own explicit test.

## Related Code Files

- Create: `internal/update/progress_writer.go`, `internal/update/progress_writer_test.go`
- Modify: `internal/update/progress.go` (add `Progress`, `StageChecksums`, retype `report`)
- Modify: `internal/update/download.go` (`downloadTo` gains `onBytes`; `downloadVerified` reports checksums + download)
- Modify: `internal/update/apply.go` (`Apply` signature, `report` call sites)
- Modify: `internal/cli/cmd_update.go` (`applyFn` type, deduping plain reporter)
- Modify: `internal/update/download_test.go`, `internal/update/apply_test.go`, `internal/cli/cmd_update_test.go`

## Implementation Steps

1. `internal/update/progress.go`: add `StageChecksums` as the first `iota`
   constant with its `String()` case (`"checksums"`), add the `Progress` struct,
   retype `report` to `func(fn func(Progress), p Progress)`.
2. `internal/update/progress_writer.go`: add the throttled counting writer.
   Plain `io.Writer` decorator, no goroutines — the caller's `io.Copy` drives it.
3. `internal/update/download.go`: add `onBytes` to `downloadTo`; pass `nil` for
   checksums and a forwarding closure for the archive; normalize
   `ContentLength < 0` to `0`; report `StageChecksums` before the checksums
   fetch in `downloadVerified`.
4. `internal/update/apply.go`: change `Apply`'s `reportFn` type; update the
   `StageInstalling` report call.
5. `internal/cli/cmd_update.go`: update the `applyFn` var/getter/setter types
   and install the deduping plain reporter above.
6. Update the fake `applyFn` implementations in `cmd_update_test.go`.
7. Tests (table-driven, real data, no mocks — repo convention):
   - counting writer: byte total exact; ≥50 ms throttle bounds emissions; a
     final emission always fires; `nil` callback is a no-op.
   - `downloadTo` against an `httptest` server: reported `Total` matches
     `Content-Length`; a chunked response with no `Content-Length` reports
     `Total == 0`; existing empty-body and cap-exceeded paths fail identically.
   - `Apply` end-to-end on the existing fixture archive: reported stage
     sequence is exactly `checksums → downloading → verifying → installing`.
   - `cmd_update.go` plain path: output contains `checksums...`,
     `downloading...`, `verifying...`, `installing...` exactly once each even
     when the reporter fires 200 times during download.

## Success Criteria

- [ ] `go test ./internal/update/ -race -count=1` green
- [ ] `go test ./internal/cli/ -race -count=1` green with no assertion loosened
- [ ] A 4.3 MB simulated download produces ≤ 120 progress callbacks
- [ ] `Total == 0` path proven by a test, not by inspection
- [ ] Plain `typeburn update` output differs from `main` by exactly one added
      `  checksums...` line
- [ ] `make lint` green; every touched file under 200 LOC

## Risk Assessment

**Risk:** wrapping the writer changes the meaning of `n` and silently breaks the
cap or empty-body guards, weakening a security-relevant check.
**Mitigation:** the decorator returns `len(p)` from `Write` unchanged and never
short-writes; the existing cap-exceeded and empty-download tests are kept
verbatim and must pass untouched.

**Risk:** the throttle drops the final update and the bar freezes at 97%.
**Mitigation:** completion always emits regardless of the timer; asserted by a
dedicated test.

**Risk:** stage renumbering breaks a comparison somewhere unnoticed.
**Mitigation:** `grep -rn "Stage" internal/` and confirm no numeric literal or
persisted value depends on the old ordering before renumbering.
