---
phase: 3
title: "stripansi-csi"
status: pending
priority: P2
effort: 30m
dependencies: []
---

# Phase 3: stripansi-csi (LOW-1)

## Overview

`stripANSI` (`internal/ui/result_render_helpers.go:76-92`) strips ANSI escape sequences so `injectBorderTitle` can measure the visual width of a lipgloss-rendered top border (`result_render_helpers.go:52-53`). The state machine only exits escape mode on the SGR final byte `'m'`:

```go
case inEsc:
    if r == 'm' {
        inEsc = false
    }
```

But CSI sequences end on **any** final byte in the range `0x40`–`0x7E` (`'@'` … `'~'`). Examples lipgloss/terminals can emit: cursor moves `ESC[A`, erase `ESC[2J`, scroll region `ESC[r`. If any non-`m`-terminated CSI reaches `stripANSI`, `inEsc` stays `true` for the **rest of the string**, swallowing all subsequent visible runes → measured width collapses → the title is mis-centered or injection is skipped (`midStart` math at `result_render_helpers.go:55-62`).

Today's risk is low (lipgloss panels are mostly SGR), hence LOW — but it's a latent correctness bug with a one-character-class fix.

## Requirements

### Functional
- `stripANSI` terminates an escape on any CSI final byte `0x40`–`0x7E`, not just `'m'`.
- Plain text unchanged. SGR sequences (`ESC[31m`, `ESC[0m`) still fully stripped.
- Non-SGR CSI (`ESC[A`, `ESC[2J`, `ESC[1;31r`) fully consumed; following text preserved.

### Non-functional
- Width measurement remains correct for all CSI inputs → border title centering robust.
- Pure-logic-adjacent: no new deps; function stays in `internal/ui`.

### Scope boundary (KISS / YAGNI)
- Only CSI (`ESC [ ... finalByte`) needs handling for this code path — that's what lipgloss emits and what `injectBorderTitle` measures.
- **OSC** (`ESC ] ... BEL`/`ST`) and **SS2/SS3** are out of scope: lipgloss panel borders do not contain them; adding OSC state would be speculative. Document this explicitly so a future reviewer doesn't "widen" without need. Current state machine treats a lone `ESC` followed by a non-`[` as: `stEsc` → `else` branch (not `[`) → `state = stNorm`, consuming exactly that one byte. For `ESC]`: `]` triggers the `else` branch in `stEsc` (since `]` != `[`), so the machine returns to `stNorm` having consumed one byte — it does NOT enter `stCSI` and the 0x40–0x7E range check never fires. Acceptable for our inputs (we never feed OSC). Note this in a comment.

## Architecture

### Current (broken)
```
for r in s:
  r == ESC          → inEsc = true
  inEsc && r == 'm' → inEsc = false        // ONLY 'm' exits → non-SGR CSI hangs
  inEsc             → drop r
  else              → keep r
```

### Target
```
inEsc && r in 0x40..0x7E → inEsc = false   // any CSI final byte exits
```
The intermediate bytes of a CSI (`0x30`–`0x3F` params, `0x20`–`0x2F` intermediates) are all `< 0x40`, so they stay in `inEsc` and are dropped correctly. The `[` after `ESC` is `0x5B` (in `@`–`~`) — but it appears as the *first* byte after ESC; with the simple flag machine, `ESC` sets `inEsc`, then `[` (0x5B) would immediately satisfy `>= '@' && <= '~'` and exit. **This is wrong** — `ESC[A` would keep `A`.

**Therefore the fix must distinguish the `[` introducer.** Minimal correct approach: only treat bytes as "final" once we're past the `[`. Track a tiny state:

```go
func stripANSI(s string) string {
    var out strings.Builder
    const (
        stNorm = iota
        stEsc  // saw ESC, expecting introducer
        stCSI  // inside CSI params/intermediates, waiting final byte
    )
    state := stNorm
    for _, r := range s {
        switch state {
        case stNorm:
            if r == '\x1b' {
                state = stEsc
            } else {
                out.WriteRune(r)
            }
        case stEsc:
            // ESC [ → CSI; any other introducer (e.g. ESC M) is a 2-byte seq → done.
            if r == '[' {
                state = stCSI
            } else {
                state = stNorm // consume the single introducer byte, drop it
            }
        case stCSI:
            if r >= '@' && r <= '~' { // CSI final byte 0x40..0x7E
                state = stNorm
            }
            // else: param/intermediate byte → keep dropping
        }
    }
    return out.String()
}
```

This is the smallest change that correctly handles both `ESC[31m` (final `m`) and `ESC[A`/`ESC[2J` while not eating the `[`. It also gracefully drops 2-byte `ESC <X>` sequences without hanging.

## Related Code Files

**Modify**
- `internal/ui/result_render_helpers.go` — `stripANSI` body + doc comment (mention CSI range + OSC out-of-scope).

**Create**
- `internal/ui/result_render_helpers_test.go` — does not exist yet; add table-driven `TestStripANSI_*`.

**Delete** — none.

## Implementation Steps

1. Replace `stripANSI` (lines 76-92) with the 3-state machine above. Update the doc comment from "ANSI SGR escape sequences (ESC [ ... m)" to "ANSI CSI escape sequences (ESC [ ... final-byte 0x40-0x7E)"; note OSC/SS3 are not expected from lipgloss borders and not handled.

2. Create `internal/ui/result_render_helpers_test.go`:
   ```go
   package ui

   import "testing"

   func TestStripANSI(t *testing.T) {
       tests := []struct{ name, in, want string }{
           {"plain", "hello", "hello"},
           {"sgr_color", "\x1b[31mred\x1b[0m", "red"},
           {"sgr_multi", "\x1b[1;31mX\x1b[0m", "X"},
           {"cursor_up", "a\x1b[Ab", "ab"},          // ESC[A non-SGR CSI
           {"erase_screen", "a\x1b[2Jb", "ab"},      // ESC[2J
           {"scroll_region", "a\x1b[1;5rb", "ab"},   // ESC[1;5r
           {"trailing_csi", "x\x1b[A", "x"},             // CSI at end
           {"trailing_incomplete_csi", "x\x1b[", "x"}, // ESC[ with no final byte (EOF in CSI)
           {"two_byte_esc", "a\x1bMb", "ab"},        // ESC M (RI) 2-byte
           {"empty", "", ""},
       }
       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               if got := stripANSI(tt.in); got != tt.want {
                   t.Errorf("stripANSI(%q) = %q, want %q", tt.in, got, tt.want)
               }
           })
       }
   }
   ```

3. (Regression guard) Add a test that `injectBorderTitle` produces a stable-width top line when the input panel contains a non-SGR CSI — optional but recommended; can assert `len([]rune(stripANSI(top)))` equals the visible char count.

4. `gofmt -w`, `go vet ./...`, `go test ./internal/ui/ -race -count=1`, then `make test-race`.

## Success Criteria

- [ ] `stripANSI` exits escape on any CSI final byte; `[` introducer not mistaken for a final byte.
- [ ] All `TestStripANSI` cases pass (SGR, cursor-up, erase, scroll-region, trailing, 2-byte ESC, empty).
- [ ] Existing result-screen golden tests unaffected.
- [ ] `make test-race` GREEN, `go vet` clean, `gofmt -l .` empty.

## Risk Assessment

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| Naive `r >= '@' && r <= '~'` eats the `[` introducer | (avoided) | High | 3-state machine distinguishes `stEsc`→`stCSI`; `[` never treated as final. Covered by `cursor_up` test. |
| OSC sequences mis-handled | Low | Low | Out of scope (lipgloss borders don't emit OSC); documented in comment. |
| Width change shifts existing golden files | Low | Low | Plain/SGR behavior identical to before; only non-SGR CSI changes (previously broken). Run UI golden tests. |

### Rollback
`git revert`; `stripANSI` returns to SGR-only. Self-contained; no other phase touches this file.
