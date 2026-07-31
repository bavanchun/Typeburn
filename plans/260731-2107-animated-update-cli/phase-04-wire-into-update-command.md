---
phase: 4
title: "Wire Into Update Command"
status: pending
priority: P1
effort: "4h"
dependencies: [3]
---

# Phase 4: Wire Into Update Command

## Overview

Connect `updateui` to `typeburn update`: run the animated block when stdout is a
wide-enough TTY, fall back to the existing plain lines otherwise, and handle
cancellation without ever corrupting a binary swap.

## Requirements

- Functional
  - TTY + width ≥ 56 columns → animated block; anything else → today's plain
    output, unchanged.
  - `ctrl+c` during download cancels cleanly, removes temporaries, and exits
    with `ExitAbort` (3) and a cancellation message.
  - Cancellation is **ignored once the install stage begins**.
  - The `[y/N]` confirmation still happens before any animation starts.
- Non-functional
  - No data race between the update goroutine and the render loop.
  - `cmd.OutOrStdout()` remains the only output sink, so tests keep working.
  - `cmd_update.go` stays under 200 LOC.

## Architecture

### Gating

Two conditions, both required:

```go
// animatable reports whether the framed renderer can run: stdout must be a real
// terminal (a bytes.Buffer in tests is not) and wide enough for the fixed
// 50-column box plus its two-space margin.
func animatable(w io.Writer) bool {
    f, ok := w.(*os.File)
    if !ok || !term.IsTerminal(int(f.Fd())) {
        return false
    }
    cols, _, err := term.GetSize(int(f.Fd()))
    return err == nil && cols >= minAnimWidth // 56
}
```

This mirrors the existing `isInteractive` seam (`cmd_update.go:46`) — an
overridable `var` so tests can force either path. The box is fixed-width by
design (Phase 3), so a narrow terminal degrades to plain rather than reflowing
mid-download.

`NO_COLOR` does **not** gate this: per repo policy the animation still runs, it
simply renders attribute-only. That matches how the TUI already treats
`NO_COLOR`.

### Concurrency: snapshot, not event stream

`update.Apply` blocks, so it runs in a goroutine. The obvious wiring — pushing
every `Progress` down a channel — forces an unpleasant choice between blocking
the download when the UI is slow and dropping messages (which can swallow a
stage transition and freeze the checklist).

Use a shared snapshot instead. The reporter writes the latest `Progress` under a
mutex; the model's 40 ms tick reads it. Progress is idempotent state, not a
sequence of events, so the newest value is always the correct one and nothing
can be "missed".

```go
// tracker is the hand-off between the blocking Apply goroutine and the render
// loop. Progress is state, not an event stream, so the renderer reads the
// latest snapshot on its own cadence instead of draining a channel — no
// backpressure on the download, no dropped stage transitions.
type tracker struct {
    mu   sync.Mutex
    cur  update.Progress
}

func (t *tracker) set(p update.Progress) { t.mu.Lock(); t.cur = p; t.mu.Unlock() }
func (t *tracker) get() update.Progress  { t.mu.Lock(); defer t.mu.Unlock(); return t.cur }
```

Completion is signalled separately over a buffered `chan result` of capacity 1,
consumed by a `tea.Cmd`, so the program quits on the real terminal state rather
than by inferring it from progress.

### Cancellation safety

`tea.NewProgram(m, tea.WithContext(ctx))` is the pattern `runtime.go:107`
already uses. `ctrl+c` cancels the context, which aborts the in-flight HTTP
request; `downloadVerified`'s existing `defer os.Remove` and `Apply`'s
`defer cleanup` remove the partial archive and extracted binary.

**The swap must not be interruptible.** Once `StageInstalling` is reported, the
work left is `extractBinary` + `replaceBinary`, and `replaceBinary` is the
atomic rename that the whole design's safety rests on. The model therefore stops
honouring `ctrl+c` at that point and shows `installing — cannot cancel`. The
download context is not consulted by the rename path anyway; this makes the
guarantee explicit in the UI instead of leaving it implicit.

### Theme resolution

`th := theme.Load(storage.LoadSettings().Theme, theme.EnvNoColor())`, so the
block honours the user's configured theme. `storage.LoadSettings` returns safe
defaults on a missing or corrupt file and never panics
(`CLAUDE.md`, "Storage is defensive"), so no error path is needed.

### File split

`cmd_update.go` is 177 LOC and this adds gating, the tracker, the goroutine, and
the program run. Move the install-and-render orchestration into a new
`internal/cli/cmd_update_run.go`, leaving `cmd_update.go` owning the cobra
command, the check/preflight flow, and the confirmation prompt.

## Related Code Files

- Create: `internal/cli/cmd_update_run.go`, `internal/cli/cmd_update_run_test.go`
- Modify: `internal/cli/cmd_update.go` (extract the apply block, add `animatable`)
- Modify: `internal/cli/cmd_update_test.go` (assert the plain path is unchanged; add a forced-animatable case)

## Implementation Steps

1. Add `animatable` as an overridable `var` next to `isInteractive`, plus the
   `minAnimWidth = 56` constant with a comment tying it to the Phase 3 box
   width.
2. Create `cmd_update_run.go` with `tracker`, the `result` channel, and
   `runApply(cmd, ver, latest, execPath) error` that branches on `animatable`.
3. Plain branch: today's code, with the Phase 2 deduping reporter. Byte-identical
   to `main` apart from the `checksums...` line.
4. Animated branch: resolve the theme, start the `Apply` goroutine with the
   tracker's `set` as reporter, build `updateui.New(...)` with `tracker.get` as
   its snapshot function, run the program with `tea.WithContext`.
5. Handle the terminal states: success prints the existing
   `updated %s → %s. restart typeburn to use the new version.` line **after**
   the program exits, so the final message survives in scrollback; failure
   returns the existing `ioError` (exit 2); cancellation returns the existing
   `abortError` (`ExitAbort` = 3, `internal/cli/exitcodes.go:9`) — no new exit
   code is needed.
6. Tests:
   - `animatable` forced false → output identical to the current golden plain
     output (whole-string comparison, not `Contains`).
   - `animatable` forced true with a fake `applyFn` → program runs and exits;
     the final success line is present on stdout.
   - a `-race` test driving `tracker.set` from one goroutine and `get` from
     another proves the hand-off is race-free.
   - cancellation before `StageInstalling` returns the cancellation error and
     leaves no file behind in the temp dir.
   - narrow-width case: `animatable` returns false at 55 columns, true at 56.

## Success Criteria

- [ ] `go test ./internal/cli/ -race -count=1` green
- [ ] Plain-path output matches a full-string golden, not just substrings
- [ ] `go test ./... -race -count=1` reports no race in the tracker test
- [ ] `ctrl+c` during download leaves no leftover file in the binary's directory
- [ ] Cancellation is refused once `StageInstalling` is reported
- [ ] `cmd_update.go` and `cmd_update_run.go` both under 200 LOC
- [ ] Manual run against a real release confirms the block renders and settles

## Risk Assessment

**Risk:** Bubble Tea takes over the terminal and the confirmation prompt or an
error message is swallowed.
**Mitigation:** the prompt runs to completion *before* the program starts, and
the success/failure lines print *after* it exits. The program only owns the
terminal for the download window.

**Risk:** cancelling mid-swap leaves a half-written binary — the one genuinely
destructive failure mode in this feature.
**Mitigation:** cancellation is refused from `StageInstalling` onward, and the
underlying swap was already atomic (`replaceBinary`, temp + rename, with the
Windows move-aside rollback). Covered by an explicit test.

**Risk:** a terminal that reports as a TTY but mangles inline rendering (some CI
shells, `script`, certain multiplexer configs).
**Mitigation:** the width probe already rejects the common broken cases, and the
plain path remains a complete, supported fallback. Document `--yes` on a
non-TTY as the scripting-safe invocation.
