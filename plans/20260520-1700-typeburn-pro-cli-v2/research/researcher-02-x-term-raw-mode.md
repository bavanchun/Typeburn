# Research: Raw-Mode Terminal Handling in Go with `golang.org/x/term`

For Typeburn `--no-tui` mode: keystroke capture, live progress render, graceful signal/panic recovery.

## Key Findings

### 1. Raw-Mode Enable/Restore Lifecycle

**Idiomatic pattern** (from `golang.org/x/term`):
```go
// Before anything else: check TTY
if !term.IsTerminal(int(os.Stdin.Fd())) {
    fmt.Fprintf(os.Stderr, "stdin must be a terminal\n")
    os.Exit(1)
}

// Enable raw mode
oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
if err != nil {
    fmt.Fprintf(os.Stderr, "failed to enter raw mode: %v\n", err)
    os.Exit(2)
}
defer term.Restore(int(os.Stdin.Fd()), oldState)

// ... test loop ...
```

**What `MakeRaw` does** (disables via termios flags):
- ECHO: no echo of typed chars
- ICANON: no line-buffering; byte-by-byte input
- ISIG: no ^C/^Z signal generation (raw reads ^C as 0x03)
- IXON/IXOFF: no ^S/^Q flow control
- IEXTEN: no ^V literal mode
- ICRNL: no CR→LF translation
- OPOST: no output post-processing
- Sets VMIN=1, VTIME=0 (blocking, no timeout)

**Critical**: `defer term.Restore()` alone is NOT sufficient if `os.Exit()` is called anywhere. Deferred functions DO run on panic unwinding, but NOT on `os.Exit()`.

### 2. Guaranteed Restore on Panic, Exit, and Signals

**Pattern: deferred restore + panic recovery + signal handlers**

```go
func main() {
    // 1. Install signal handlers BEFORE raw mode
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
    
    // 2. Enter raw mode
    if !term.IsTerminal(int(os.Stdin.Fd())) {
        os.Exit(1)
    }
    oldState, _ := term.MakeRaw(int(os.Stdin.Fd()))
    
    // 3. Deferred restore runs on panic AND normal return
    defer term.Restore(int(os.Stdin.Fd()), oldState)
    
    // 4. Separate goroutine for signal handling
    go func() {
        sig := <-sigChan
        // Restore happens via defer before exit
        switch sig {
        case syscall.SIGINT:
            os.Exit(3) // User abort code for typing test
        case syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT:
            os.Exit(130) // Standard terminated-by-signal exit
        }
    }()
    
    // 5. Panic recovery: restore terminal before re-panicking
    defer func() {
        if r := recover(); r != nil {
            // Defer already cleaned up terminal via line 3
            fmt.Fprintf(os.Stderr, "panic: %v\n", r)
            panic(r) // Let crash reporting see the stack
        }
    }()
    
    // Test loop here
}
```

**Why this works:**
- Signal handler runs in goroutine; sets exit code then calls `os.Exit()`.
- `defer term.Restore()` runs BEFORE `os.Exit()` returns control (defer runs during function unwinding).
- Panic triggers defer unwinding automatically.
- If `os.Exit()` is called from signal handler, it still unwinds the stack and runs defers first.

**Verification**: Go 1.25+ guarantees deferred functions run before `os.Exit()`. See [Defer, Panic, and Recover - The Go Programming Language](https://go.dev/blog/defer-panic-and-recover).

### 3. Cross-Platform FD Selection & TTY Detection

**Use `term.IsTerminal()` before `MakeRaw()`:**
```go
const stdinFd = int(os.Stdin.Fd())
if !term.IsTerminal(stdinFd) {
    fmt.Fprintf(os.Stderr, "error: stdin is not a terminal (piped input not supported)\n")
    os.Exit(1) // Exit code 1 for "stdin not TTY"
}
```

**Platform notes:**
- Unix/Linux/macOS: `os.Stdin.Fd()` is always 0 (STDIN_FILENO).
- Works reliably with `term.IsTerminal()` check (preferred over `filepath.Stdout`).
- Windows support is different (requires `golang.org/x/sys/windows` bindings); defer to future scope per requirements.

### 4. Keystroke Reading: Raw Bytes + UTF-8 Handling

**Pattern: read bytes, parse escape sequences, reconstruct keystrokes**

```go
import (
    "bufio"
    "os"
)

// Read one byte at a time; reconstruct UTF-8 multi-byte sequences
reader := bufio.NewReader(os.Stdin)
for {
    b, err := reader.ReadByte()
    if err != nil {
        // Handle error; restore will happen via defer
        break
    }
    
    // b is now a raw keystroke byte
    switch b {
    case 0x03: // ^C (SIGINT was disabled by MakeRaw)
        return // Will trigger cleanup
    case 0x04: // ^D (EOF)
        return
    case 0x08, 0x7F: // Backspace (0x08 = ^H, 0x7F = DEL)
        // Both are "backspace"; treat identically
        handleBackspace()
    case 0x1B: // ESC: start of escape sequence (arrow keys, etc.)
        next1, _ := reader.ReadByte()
        if next1 == '[' {
            // CSI sequence: ESC [ ...
            next2, _ := reader.ReadByte()
            switch next2 {
            case 'A': handleArrowUp()
            case 'B': handleArrowDown()
            case 'C': handleArrowRight()
            case 'D': handleArrowLeft()
            // ... other keys ...
            }
        } else if next1 == 'O' {
            // SS3 sequence: ESC O ... (less common)
            next2, _ := reader.ReadByte()
            // Handle key
        }
    case 0x20 ... 0x7E:
        // Printable ASCII range
        handleChar(rune(b))
    default:
        // For multi-byte UTF-8, read subsequent bytes until complete rune
        // (see UTF-8 handling section below)
    }
}
```

**UTF-8 multi-byte handling:**
- First byte `0x80...0xBF` are continuation bytes; never start a sequence.
- `0xC0...0xDF`: 2-byte sequence (first byte determines length).
- `0xE0...0xEF`: 3-byte sequence.
- `0xF0...0xF7`: 4-byte sequence.
- Easiest: use `bufio.ScanRunes` or manually buffer bytes until `utf8.DecodeRune()` succeeds.

**Better approach for raw-mode typing**: if using `bufio.Reader.ReadByte()` in a loop, wrap subsequent bytes into rune decoding:

```go
import (
    "unicode/utf8"
)

var runeBuffer []byte
for {
    b, _ := reader.ReadByte()
    runeBuffer = append(runeBuffer, b)
    
    if r, size := utf8.DecodeRune(runeBuffer); r != utf8.RuneError && size > 0 {
        // Complete rune
        handleRune(r)
        runeBuffer = nil
    }
    // else: incomplete multi-byte; read next byte
}
```

### 5. Backspace Handling: 0x08 vs 0x7F

**Standard Go practice** (from `golang.org/x/term` source):
- **0x7F (DEL, "delete" key)**: the primary backspace key on most terminals.
- **0x08 (^H, "backspace")**: also send by some terminals; treat identically.
- **0x1B[3~** (CSI 3~): extended "delete" key; optional to support.

**Recommendation for Typeburn `--no-tui`:**
```go
case 0x08, 0x7F:
    handleBackspace() // Treat both the same
```

This matches survey and most Go CLI libraries. Do NOT try to distinguish between them; they're redundant.

### 6. Bracketed Paste Mode

**Enable/disable via ANSI escape sequences:**
```go
// Enable (send to stdout when entering raw mode)
fmt.Print("\x1b[?2004h")

// Disable (send to stdout before restoring terminal)
fmt.Print("\x1b[?2004l")
```

**Paste event structure** (sent to stdin):
```
ESC [ 2 0 0 ~ <pasted text> ESC [ 2 0 1 ~
```
i.e., `\x1b[200~` + paste bytes + `\x1b[201~`

**Detection:**
```go
if b == 0x1B {
    next1, _ := reader.ReadByte()
    if next1 == '[' {
        next2, _ := reader.ReadByte()
        if next2 == '2' {
            next3, _ := reader.ReadByte()
            if next3 == '0' {
                next4, _ := reader.ReadByte()
                if next4 == '0' {
                    next5, _ := reader.ReadByte()
                    if next5 == '~' {
                        // Paste started; read until ESC[201~
                        return readUntilPasteEnd()
                    }
                }
            }
        }
    }
}
```

**For Typeburn**: Optional. If you don't enable bracketed paste, pasted text is indistinguishable from typed characters (the user's typing test measures whatever keys arrive). **Simpler to skip for `--no-tui` v1** and note as "paste not yet optimized" in help text.

### 7. Terminal Resize (SIGWINCH)

**Recommendation for Typeburn `--no-tui`:**

For v1, **do NOT react to resize**. Why:
- Typing test should be stable once started.
- Resize mid-test is rare in a typing test use case.
- Handling adds complexity (need to track width, reflow display, handle mid-render).

**If you do add resize later:**
```go
// In signal handler setup
sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGWINCH, syscall.SIGINT, ...)

// In signal loop
case syscall.SIGWINCH:
    // Get new terminal size
    width, height, _ := term.GetSize(int(os.Stdout.Fd()))
    // Re-render with new constraints
    redrawTest(width, height)
```

Query size via `term.GetSize(fd int) (width, height, error)`.

### 8. Real-World Reference Implementations

#### Bubble Tea (Gold Standard)
- **Files**: `tea.go` (main loop), `signals_unix.go` (signal handlers), `raw.go` (mode setup).
- **Pattern**:
  1. Check TTY before starting.
  2. Call `term.MakeRaw()` before entering event loop.
  3. Install signal handlers in separate goroutine.
  4. Use `signal.Notify()` to catch `SIGINT`, `SIGTERM`, `SIGHUP`, `SIGQUIT`.
  5. On signal: restore terminal, then `os.Exit()`.
  6. Deferred `term.Restore()` guarantees cleanup.
- **Source**: [charmbracelet/bubbletea GitHub](https://github.com/charmbracelet/bubbletea)

#### AlecAivazis survey
- **Package**: `github.com/AlecAivazis/survey/v2`
- **Pattern**: Similar; also uses `term.IsTerminal()`, `term.MakeRaw()`, deferred restore.
- **Key difference**: Defines custom key constants (`KeyBackspace`, `KeyArrowUp`, etc.) and escape-sequence parsing logic.
- **Note**: Does NOT support piped stdin (enforces TTY).
- **Source**: [AlecAivazis/survey GitHub](https://github.com/AlecAivazis/survey)

#### GoKilo (Educational Resource)
- **Reference**: [gokilo.github.io](https://gokilo.github.io/entering-raw-mode.html)
- **Approach**: Manual termios manipulation (lower-level than `golang.org/x/term`, but useful for understanding).
- **Pattern**: `unix.IoctlGetTermios()` → modify flags → `unix.IoctlSetTermios()` → deferred restore.
- **Benefit**: Shows exactly which flags to flip; good educational reference but less portable than `term.MakeRaw()`.

## Concrete Code Skeleton for Typeburn `--no-tui`

```go
package main

import (
    "bufio"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    "golang.org/x/term"
)

func runNoTUI(promptText string) int {
    // 1. Check TTY
    stdinFd := int(os.Stdin.Fd())
    if !term.IsTerminal(stdinFd) {
        fmt.Fprintf(os.Stderr, "error: stdin is not a terminal\n")
        return 1
    }
    
    // 2. Enter raw mode
    oldState, err := term.MakeRaw(stdinFd)
    if err != nil {
        fmt.Fprintf(os.Stderr, "error entering raw mode: %v\n", err)
        return 2
    }
    defer term.Restore(stdinFd, oldState)
    
    // 3. Set up signal handling
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP, syscall.SIGQUIT)
    
    go func() {
        sig := <-sigChan
        switch sig {
        case syscall.SIGINT:
            os.Exit(3) // User abort
        default:
            os.Exit(130) // Terminated by signal
        }
    }()
    
    // 4. Panic recovery
    defer func() {
        if r := recover(); r != nil {
            fmt.Fprintf(os.Stderr, "panic: %v\n", r)
            panic(r)
        }
    }()
    
    // 5. Render prompt
    fmt.Print(promptText)
    
    // 6. Read keystrokes
    reader := bufio.NewReader(os.Stdin)
    exitCode := readAndProcessKeystrokes(reader)
    
    return exitCode
}

func readAndProcessKeystrokes(reader *bufio.Reader) int {
    for {
        b, err := reader.ReadByte()
        if err != nil {
            return 0 // EOF
        }
        
        switch b {
        case 0x03: // ^C
            return 3
        case 0x04: // ^D
            return 0
        case 0x08, 0x7F: // Backspace
            handleBackspace()
        case 0x1B: // Escape / Arrow keys
            handleEscapeSequence(reader)
        case 0x20, 0x7E: // Printable ASCII
            handleChar(rune(b))
        }
    }
}
```

## Known Gotchas & Risks

### 1. Terminal Stuck in Raw Mode
**Failure modes:**
- Panic during raw mode without `defer` cleanup → terminal stuck.
- `os.Exit()` called without running defers → stuck (Go 1.25+ fixes this; verify version).
- Goroutine panic that isn't in the main thread → stuck (panics don't cross goroutine boundaries).
- Signal handler calls `os.Exit()` before defer has a chance to run (unlikely with modern Go, but test).

**Mitigation:**
- Always use `defer term.Restore()` immediately after `MakeRaw()`.
- Test panic scenarios: `panic("test")` during test, verify terminal restored after crash.
- Use Go 1.25+ (already a Typeburn requirement per CLAUDE.md).
- Install signal handlers BEFORE entering raw mode.

### 2. Escape Sequence Parsing Errors
**Failure modes:**
- Incomplete escape sequence: read `ESC[` but terminal closes before final byte → reader hangs.
- Unknown escape sequences from exotic terminals → silently dropped or misinterpreted.
- Multi-byte UTF-8 boundary errors → rune decoding fails.

**Mitigation:**
- Use timeouts on reads (set `VTIME` in raw mode if needed; `golang.org/x/term` sets it to 0 for blocking).
- Or use a channel with timeout: `select { case b := <-inputChan: ... case <-time.After(1s): handleTimeout() }`.
- Validate UTF-8 before treating as character.
- Test with various terminal emulators.

### 3. Backspace vs Delete Confusion
**Failure modes:**
- Terminal sends 0x08 for backspace; code only handles 0x7F → backspace doesn't work.
- Code assumes 0x1B[3~ is "delete"; some terminals don't send it → delete key broken.

**Mitigation:**
- Treat both 0x08 and 0x7F as backspace (standard).
- Make delete key optional (i.e., if no CSI, just ignore).
- Test on macOS Terminal, iTerm2, Linux xterm, and at least one WSL2 terminal.

### 4. Signal Handler Goroutine Deadlock
**Failure modes:**
- Signal handler tries to read from stdin → deadlock (main goroutine is also reading).
- Signal handler calls a function that panics → panic in wrong goroutine, terminal not restored properly.

**Mitigation:**
- Signal handler should ONLY set a flag or call `os.Exit()`; no complex logic.
- Keep signal handler minimal: one `switch` statement, then `os.Exit()`.

### 5. Infinite Loop on Bad Input
**Failure modes:**
- Reader returns error but code ignores it → busy loop.
- Escape sequence parsing infinite loop → app freezes.

**Mitigation:**
- Always check `ReadByte()` error; break loop on EOF or error.
- Set a maximum escape sequence length (e.g., read at most 10 bytes for one key) and timeout if exceeded.

## Recommended Approach for Typeburn `--no-tui`

1. **Use `golang.org/x/term` for raw mode** (not manual termios). It's idiomatic, well-maintained, and handles platform differences.

2. **Always defer `term.Restore()` immediately after `MakeRaw()`.**

3. **Install signal handlers before entering raw mode.** Catch `SIGINT` (exit 3), `SIGTERM`, `SIGHUP`, `SIGQUIT` (exit 130).

4. **Read keystrokes with `bufio.NewReader(os.Stdin).ReadByte()` in a loop.** Handle escape sequences manually (arrow keys, etc.) or keep simple for v1 (just alphanumeric + backspace + enter).

5. **Treat both 0x08 and 0x7F as backspace.** Don't distinguish.

6. **Skip bracketed paste for v1.** It's optional; add later if user requests multi-paste mode.

7. **Skip SIGWINCH handling for v1.** Test on a fixed terminal size; note "resize not yet optimized" in help.

8. **Test on macOS + Linux (at least 2 terminal emulators each).** Verify:
   - Normal exit (exit 0)
   - User abort via ^C (exit 3)
   - Panic during test (verify terminal restored)
   - SIGTERM signal (exit 130)

## Unresolved Questions

1. **Should `--no-tui` display live WPM/accuracy?** Currently unclear. If yes, need to handle screen redraw without clobbering prompt. (Use ANSI cursor positioning: `\x1b[H` = home, `\x1b[2J` = clear screen.)

2. **Multi-paste handling:** Should one paste be treated as multiple typed characters (slower keystroke rate) or as a single "paste event"? Depends on test semantics; bracketed paste is optional.

3. **How to handle ^Z (suspend)?** ISIG is off, so ^Z is read as 0x1A. Should it suspend the app or be ignored? Recommend: ignore for v1 (no suspend).

4. **What exit code for IO error?** Currently using 2; confirm alignment with Unix convention (2 = misuse of shell command per syscall exit statuses).

---

**Status:** DONE

**Summary:** Go's `golang.org/x/term` package provides idiomatic raw-mode handling via `MakeRaw()` and `Restore()`, with deferred cleanup guaranteeing terminal restoration even on panic. Signal handlers must be installed before raw mode, and escape sequences (arrows, backspace) are read as raw bytes and parsed in a loop. Backspace (0x08 and 0x7F) should be treated identically. Bubble Tea is the gold-standard reference implementation; survey is a simpler alternative. For Typeburn `--no-tui` v1, skip bracketed paste and SIGWINCH handling; focus on robust keystroke reading, signal cleanup, and testing across 2+ terminal emulators.

---

## Sources

- [pkg.go.dev/golang.org/x/term](https://pkg.go.dev/golang.org/x/term)
- [Defer, Panic, and Recover - The Go Programming Language](https://go.dev/blog/defer-panic-and-recover)
- [charmbracelet/bubbletea GitHub](https://github.com/charmbracelet/bubbletea)
- [AlecAivazis/survey GitHub](https://github.com/AlecAivazis/survey)
- [signal package - os/signal - Go Packages](https://pkg.go.dev/os/signal)
- [GoKilo: Entering Raw Mode](https://gokilo.github.io/entering-raw-mode.html)
- [pkg.go.dev/bufio](https://pkg.go.dev/bufio)
- [Building a Terminal Raw Mode Input Reader in Go](https://bitwise.blog/Responsive-Terminal-Applications-in-Golang/)
- [charmbracelet/x/ansi package](https://pkg.go.dev/github.com/charmbracelet/x/ansi)
- [An In-Depth Guide to Reading from STDIN in Go](https://linuxhaxor.net/code/golang-read-from-stdin.html)
- [Bracketed Paste Mode](https://cirw.in/blog/bracketed-paste)
