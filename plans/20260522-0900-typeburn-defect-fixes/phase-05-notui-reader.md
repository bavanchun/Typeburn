---
phase: 5
title: "notui-reader"
status: pending
priority: P2
effort: 30m
dependencies: []
---

# Phase 5: notui-reader (LOW-3)

## Overview

`discardEscape` (`internal/cli/notui/reader.go:62-72`) is called by `ReadEvent` after consuming an `ESC` byte (`reader.go:33-35`) to swallow the rest of an escape sequence in `--no-tui` raw input. It only loops while `r.Buffered() > 0`:

```go
func discardEscape(r *bufio.Reader) {
    for r.Buffered() > 0 {     // ← exits immediately if buffer empty
        b, err := r.ReadByte()
        ...
        if (A-Z|a-z|~) { return }
    }
}
```

**Bug:** when `ESC` arrives at the end of a read buffer (e.g. a slow terminal or piped input split across reads), `r.Buffered()` is 0, so `discardEscape` returns immediately having consumed nothing. The CSI introducer `[` and final byte (e.g. `A`) arrive on the **next** read and are processed as ordinary input — `[` gets emitted as a typed rune. Stray `[` corrupts the typing test.

The current test `{"escape", "\x1b[A", EventNone, 0}` (reader_test.go:20) passes only because the whole sequence is in one buffer. Split reads aren't covered.

**Fix:** always do a blocking `ReadByte` for the introducer, then loop reading final bytes. Blocking is correct here — after `ESC` we *must* consume the introducer; an isolated `ESC` (no following bytes) only blocks until the next byte or EOF, which is the desired raw-mode behavior.

## Requirements

### Functional
- After `ESC`, blocking-read one byte (the introducer):
  - If it's a control byte (`0x03` Ctrl-C or `0x04` Ctrl-D) → unread it and return so the outer loop handles abort.
  - If it's not `[` (CSI) or `O` (SS3) → standalone ESC or 2-byte sequence; stop (consume just that one byte).
  - If `[` or `O` → keep blocking-reading until a CSI/SS3 final byte: `A`–`Z`, `a`–`z`, or `~`.
- On `io.EOF`/error mid-sequence → return cleanly (no panic).
- Single-buffer `ESC[A` still yields `EventNone` (no regression on reader_test.go:20).
- SS3 (`ESC O P`, e.g. F1) consumed.
- Extended CSI with tilde terminator (`ESC[3~`, e.g. Delete) consumed.
- Split-buffer ESC (introducer/final arriving in a later read) consumed — no stray `[`.
- `ESC` then `Ctrl-C` (0x03): Ctrl-C is unread and re-emitted to the outer loop — session remains abortable.

### Non-functional
- Raw-mode robustness; no change to interactive TUI (this is notui-only).
- No new deps.

## Architecture

### Current (broken)
```
discardEscape(r):
  while r.Buffered() > 0:        // empty buffer at ESC boundary → no-op
    read b; if final → return
```

### Target (per task spec, verified against bufio semantics)
```go
func discardEscape(r *bufio.Reader) {
    b, err := r.ReadByte()       // ALWAYS read (blocking) — handles split reads
    if err != nil {
        return
    }
    // Preserve control bytes so the outer loop can handle them (e.g. Ctrl-C/Ctrl-D abort).
    // Without this, ESC then Ctrl-C would silently eat 0x03 and make the session un-abortable.
    if b == 0x03 || b == 0x04 {
        _ = r.UnreadByte()
        return
    }
    if b != '[' && b != 'O' {    // standalone ESC or 2-byte seq, not CSI/SS3
        return
    }
    for {
        b, err := r.ReadByte()
        if err != nil {
            return
        }
        if (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z') || b == '~' {
            return
        }
    }
}
```

**Behavioral note on standalone ESC:** if the user presses bare `ESC`, the first blocking `ReadByte` waits for the *next* keystroke. That next byte is then consumed as the "introducer" and dropped (unless it's a control byte, which is unread). For a typing test in raw mode this is acceptable: bare ESC is not a typing input, and the next char being eaten on a deliberate ESC press is a negligible edge. Document this trade-off in a comment. (If undesired later, a select/timeout could distinguish, but that is YAGNI for v2.1.)

## Related Code Files

**Modify**
- `internal/cli/notui/reader.go` — `discardEscape` (lines 62-72) + comment on blocking/standalone-ESC trade-off.

**Modify (tests)**
- `internal/cli/notui/reader_test.go` — add SS3, extended-tilde, and split-buffer cases.

**Delete** — none.

## Implementation Steps

1. Replace `discardEscape` (reader.go:62-72) with the blocking version above. Keep the call site (`reader.go:34`) unchanged.

2. Extend `reader_test.go`. The existing `TestReadEvent` table (test:9-34) covers single-buffer cases — add to it:
   ```go
   {"ss3 f1", "\x1bOP", EventNone, 0},        // ESC O P
   {"csi tilde", "\x1b[3~", EventNone, 0},    // ESC [ 3 ~ (Delete)
   ```
   These flow through `ReadEvent`→`discardEscape` and must yield `EventNone`.

3. Add a dedicated split-read test (single-buffer table can't express it). Build a reader whose first `Read` returns only `"\x1b"` and a later `Read` returns `"[A"`. Two clean ways:
   - **Simple/sufficient:** since `bufio.Reader` over a `strings.Reader` does one underlying read per `Fill`, and `ReadByte` blocks for more, feed an `io.MultiReader(strings.NewReader("\x1b"), strings.NewReader("[Ax"))`. After `ReadEvent` consumes ESC + `[A`, the **next** `ReadEvent` must return the trailing `x` as `EventRune`:
     ```go
     func TestReadEvent_SplitEscape(t *testing.T) {
         r := bufio.NewReader(io.MultiReader(
             strings.NewReader("\x1b"),
             strings.NewReader("[Ax"),
         ))
         ev, err := ReadEvent(r)   // consumes ESC + [A across the split
         if err != nil || ev.Kind != EventNone {
             t.Fatalf("escape: want EventNone, got %v (err %v)", ev.Kind, err)
         }
         ev, err = ReadEvent(r)    // must now read the real char, not a stray '['
         if err != nil {
             t.Fatal(err)
         }
         if ev.Kind != EventRune || ev.Rune != 'x' {
             t.Fatalf("after split escape: want rune 'x', got %v/%q", ev.Kind, ev.Rune)
         }
     }
     ```
   - This is the key regression test: under the old code `discardEscape` would return at the split, and the next `ReadEvent` would emit `[` (`EventRune '['`) — the test would catch the stray `[`.

   Imports needed in `reader_test.go`: add `"io"` (already has `bufio`, `strings`, `testing`).

4. Add `TestReadEvent_StandaloneESC` to document the known behavior that the byte following a bare ESC is consumed:
   ```go
   func TestReadEvent_StandaloneESC(t *testing.T) {
       // After bare ESC, discardEscape reads and drops the next byte ('x').
       // This is the accepted trade-off: bare ESC is not a typing input.
       r := bufio.NewReader(strings.NewReader("\x1bx"))
       ev, err := ReadEvent(r)
       if err != nil || ev.Kind != EventNone {
           t.Fatalf("standalone ESC: want EventNone, got %v (err %v)", ev.Kind, err)
       }
       // 'x' was consumed as the introducer — stream is now empty.
       _, err = ReadEvent(r)
       if err == nil {
           t.Fatal("expected EOF after standalone ESC consumed the next byte")
       }
   }
   ```

5. Add `TestReadEvent_EscThenCtrlC` to verify Ctrl-C is preserved after ESC:
   ```go
   func TestReadEvent_EscThenCtrlC(t *testing.T) {
       r := bufio.NewReader(strings.NewReader("\x1b\x03"))
       ev, err := ReadEvent(r) // ESC triggers discardEscape, which unreads 0x03
       if err != nil || ev.Kind != EventNone {
           t.Fatalf("ESC: want EventNone, got %v (err %v)", ev.Kind, err)
       }
       ev, err = ReadEvent(r) // 0x03 was unread by discardEscape — must come back as EventAbort
       if err != nil || ev.Kind != EventAbort {
           t.Fatalf("Ctrl-C after ESC: want EventAbort, got %v (err %v)", ev.Kind, err)
       }
   }
   ```

6. `gofmt -w`, `go vet ./...`, `go test ./internal/cli/notui/ -race -count=1`, then `make test-race`.

## Success Criteria

- [ ] `discardEscape` always blocking-reads the introducer; handles `[`/`O`; loops to a final byte.
- [ ] Ctrl-C (0x03) and Ctrl-D (0x04) after ESC are unread and re-emitted — session remains abortable.
- [ ] Single-buffer `ESC[A` still → `EventNone` (no regression).
- [ ] SS3 `ESC O P` and extended `ESC[3~` → `EventNone`.
- [ ] `TestReadEvent_SplitEscape`: after a split ESC the next rune is the real char, not `[`.
- [ ] `TestReadEvent_StandaloneESC`: documents that the byte after bare ESC is consumed.
- [ ] `TestReadEvent_EscThenCtrlC`: after `ESC`+`0x03`, the second `ReadEvent` returns `EventAbort` (not lost).
- [ ] EOF mid-sequence does not panic.
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Bare ESC now blocks/eats the next keystroke | Medium | Low | Documented trade-off; bare ESC is not typing input. Acceptable for v2.1; revisit only if reported. |
| Blocking read hangs on a truly idle ESC at stream end | Low | Low | On stream end `ReadByte` returns `io.EOF` → clean return (covered by error guards). |
| Non-CSI/SS3 escapes (e.g. `ESC M`) over-consumed | Low | Low | Introducer check returns immediately for non-`[`/`O`, consuming exactly one byte — matches a 2-byte ESC sequence. |

### Rollback
`git revert`; `discardEscape` returns to buffer-gated loop. Isolated to `reader.go`. No other phase touches notui reader.
