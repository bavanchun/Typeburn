---
phase: 5
title: "raw-terminal --no-tui runner"
status: completed
priority: P1
effort: "8h"
dependencies: [1, 3]
---

# Phase 5: `run --no-tui` raw-terminal runner

## Overview

The riskiest phase. Adds a Bubble-Tea-free runner that reads keystrokes from
stdin in raw mode, prints minimal prompt+progress to stdout, and exits cleanly.
Per researcher-02, gold-standard pattern uses `golang.org/x/term` +
signal handlers installed BEFORE entering raw mode + `defer term.Restore`.

TDD here: write the signal/restore lifecycle test first (panic + signal both
restore terminal), then the read loop.

## Requirements

- Functional:
  - `typeburn run --no-tui --mode words --words 10` works on macOS + Linux
  - ^C mid-test → terminal restored to cooked mode, exit code 3
  - Test completes naturally → metrics printed (plain or `--json`), exit 0
  - stdin is not a TTY → error "stdin is not a terminal", exit 1
  - panic anywhere in runner → recover, restore terminal, re-panic with original
  - `--no-tui --mode code` REQUIRES `--text <file>`; without it exits 1 (validate-3). Bracketed paste is NOT supported in v2 `--no-tui`.
- Non-functional:
  - Zero ANSI raw bytes leaked on exit (terminal restored)
  - Cross-platform: linux/macOS only (Windows out of scope)

## Architecture

```
internal/cli/notui/
  runner.go      # NewRunner(session, mode, length, ql), Run() error
  lifecycle.go   # raw-mode enter/exit, signal handlers, panic recovery
  render.go      # minimal ANSI prompt: target line, cursor, live WPM
  reader.go      # byte-by-byte input → runes + special keys
  runner_test.go
  lifecycle_test.go
```

Lifecycle skeleton (from researcher-02, ordered per F8 fix — `signal.Notify` BEFORE `MakeRaw`):

```go
func runRaw(fn func(rd io.Reader, w io.Writer) error) (err error) {
    if !term.IsTerminal(int(os.Stdin.Fd())) {
        return errNotTTY
    }
    fd := int(os.Stdin.Fd())

    // F8: install signal handlers FIRST. If a signal fires before MakeRaw runs,
    // it goes through our handler (which restores nothing because raw mode never
    // engaged) and exits cleanly. The opposite order has a tiny race window
    // where SIGINT default-kills the process AFTER raw-mode is on but BEFORE
    // Notify is wired, leaving the terminal stuck in raw mode.
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
    defer signal.Stop(sigCh)

    old, err := term.MakeRaw(fd)
    if err != nil { return err }
    restored := false
    restore := func() {
        if !restored { _ = term.Restore(fd, old); restored = true }
    }
    defer restore()

    done := make(chan error, 1)
    go func() {
        defer func() {
            if r := recover(); r != nil {
                restore()
                panic(r) // re-raise after restoring
            }
        }()
        done <- fn(os.Stdin, os.Stdout)
    }()

    select {
    case err = <-done:
        return err
    case s := <-sigCh:
        restore()
        if s == syscall.SIGINT { return errAbort }   // exit code 3
        return fmt.Errorf("received signal: %v", s)
    }
}
```

Read loop reads bytes via `bufio.Reader.ReadByte`. Backspace = 0x08 OR 0x7F.
ESC sequences (arrow keys, etc.) consumed and discarded. UTF-8 multi-byte: peek
high bit; collect continuation bytes; decode via `utf8.DecodeRune`.

Render is line-based:
- Print target text once
- After each keystroke, repaint a status line (`\r` + clear-to-eol) with progress + live WPM
- On completion, print a 2-line summary (or JSON if `--json`)

## Related Code Files

- Create: `internal/cli/notui/{runner,lifecycle,render,reader}.go` + tests
- Modify: `internal/cli/cmd_run.go` (wire `--no-tui` flag → notui.Runner)
- Modify: `go.mod`/`go.sum` (`golang.org/x/term`)

## Implementation Steps

1. Add dep with pinned tag: `go get golang.org/x/term@<concrete-tag>` (record tag in CONTRIBUTING.md "Dep upgrade gate"). NOT `@latest`.
2. Write `lifecycle_test.go` first:
   - Case: callback returns nil → no signal → exit ok
   - Case: callback panics → `recover()` fires → restore called → panic propagated (use `defer/recover` in the test)
   - Case: simulated SIGINT → restore called → errAbort returned
3. Implement `lifecycle.go` per skeleton above.
4. Write `reader_test.go`:
   - ASCII rune → emit rune
   - UTF-8 2-byte rune → decoded
   - 0x08 and 0x7F → backspace event
   - ESC + `[A` → discarded (arrow up)
5. Implement `reader.go`.
6. Implement `runner.go`: build `runner.Session` via Phase 1 driver; loop read → engine.Apply → check complete → break.
7. Implement `render.go`: prompt, status line. No theming (monochrome).
8. Wire `--no-tui` in `cmd_run.go` to dispatch to `notui.Run(...)`. Reject `--no-tui --mode code` when `--text` is empty (validate-3).
9. Manual smoke on macOS Terminal + Linux xterm + tmux.
10. F9: add `make notui-noexit-check` target that greps `internal/cli/notui/` for `os.Exit` and fails CI if found. Wire into `make lint` and `ci.yml`.

## Success Criteria

- [ ] `typeburn run --no-tui --mode words --words 10` completes with stdout summary
- [ ] ^C mid-test exits with code 3; terminal cooked-mode confirmed by `stty -a` (script-test)
- [ ] panic in runner restores terminal (test asserts via mock `restore` spy)
- [ ] `echo "" | typeburn run --no-tui ...` exits 1 "stdin is not a terminal"
- [ ] `--json` with `--no-tui` emits valid JSON on completion
- [ ] No regression on existing TUI tests
- [ ] Tests on darwin + linux via CI matrix (Phase 7 sets that up)

## Risk Assessment

- **Risk:** Terminal stuck in raw mode after a failure path is missed.
  **Mitigation:** Single `restore()` closure with idempotent `restored` guard, called from BOTH defer AND signal handler AND panic recovery. Plus an integration smoke that `stty -a` post-exit.
- **Risk:** `os.Exit` called somewhere inside the runner bypasses `defer term.Restore`.
  **Mitigation:** Runner returns error; only `main.go` calls `os.Exit`. Lint rule check: grep `os.Exit` in `internal/cli/notui/` must be empty.
- **Risk:** Reader hangs on EOF (stdin closed).
  **Mitigation:** `ReadByte` returns `io.EOF` → exit cleanly with code 2 + restore.
- **Risk:** SIGWINCH (resize) corrupts the rendered prompt.
  **Mitigation:** v2 deliberately ignores resize; document and accept. Add to roadmap.
- **Risk:** Bracketed paste fragment escapes leak into typed buffer.
  **Mitigation:** v2 does NOT enable bracketed paste in `--no-tui`. Document.
- **Risk:** Some terminal emulators send Backspace as 0x08 vs 0x7F inconsistently.
  **Mitigation:** Treat both as backspace (researcher-02 confirms standard practice).
