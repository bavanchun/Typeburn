---
phase: 4
title: "Typing + UI Test Hardening"
status: pending
priority: P2
effort: "1.5h"
dependencies: [1]
---

# Phase 4: Typing + UI Test Hardening

## Overview

Add tests for 2 critical untested high-churn files:
- `internal/typing/completion.go` (5 changes, zero tests, core business logic)
- `internal/ui/screen_typing_actions.go` (7 changes, zero tests, core input handler)

These are the #1 and #4 hardening targets from the git retro file hotspot analysis.

## Related Code Files

- Create: `internal/typing/completion_test.go`
- Create: `internal/ui/screen_typing_actions_test.go`

## Implementation Steps

### 1. `completion_test.go` — comprehensive completion logic tests

```go
func TestComplete_TimeMode(t *testing.T) {
    // Time mode: complete when nowMs >= wordTarget (stored as ms)
    cases := []struct {
        name      string
        nowMs     int64
        targetMs  int
        want      bool
    }{
        {"before", 29999, 30000, false},
        {"exact",  30000, 30000, true},
        {"after",  30001, 30000, true},
        {"zero",   0,     30000, false},
    }
    // ... run engine with ModeTime, set wordTarget, check Complete
}

func TestComplete_WordsMode(t *testing.T) {
    // Words mode: complete when all target words are typed
    // Test: 3-word target, type all 3 → true; type 2 → false
}

func TestComplete_QuoteMode(t *testing.T) {
    // Quote mode: complete when typed == target exactly
    // Test: exact match → true; partial → false; extra char → false
}

func TestCountCompletedWords(t *testing.T) {
    cases := []struct {
        name   string
        typed  string
        target string
        want   int
    }{
        {"empty typed",    "",           "hello world", 0},
        {"empty target",   "hello",      "",            0},
        {"both empty",     "",           "",            0},
        {"one word done",  "hello ",     "hello world", 1},
        {"mid word",       "hell",       "hello world", 0},
        {"all done",       "hello world","hello world", 2},
        {"trailing space", "hello world ","hello world",2},
        {"single word",    "hi",         "hi",          1},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            got := countCompletedWords([]rune(tc.typed), []rune(tc.target))
            if got != tc.want {
                t.Errorf("countCompletedWords(%q, %q) = %d, want %d",
                    tc.typed, tc.target, got, tc.want)
            }
        })
    }
}

func TestRunesEqual(t *testing.T) {
    cases := []struct {
        a, b string
        want bool
    }{
        {"abc", "abc", true},
        {"abc", "abd", false},
        {"abc", "ab",  false},
        {"",    "",    true},
        {"a",   "",    false},
    }
    // ...
}
```

### 2. `screen_typing_actions_test.go` — action function tests

```go
func TestRestartSame_ResetsTimersAndWPM(t *testing.T) {
    // Create a TypingModel with some progress, restartSame should zero out
    // startMs, nowMs, lastPaintMs, headerWPM while keeping target
    m := NewTyping(config.ModeWords, 10, 0, theme.Default(),
        config.DefaultKeymap(), false)
    // Simulate some typing progress...
    m2 := m.restartSame()
    if m2.startMs != 0 || m2.nowMs != 0 || m2.headerWPM != 0 {
        t.Error("restartSame should zero timing fields")
    }
    // Target text should be preserved
    if m2.TargetText() != m.TargetText() {
        t.Error("restartSame should preserve target text")
    }
}

func TestNewTest_RegeneratesTarget(t *testing.T) {
    // newTest should produce a fresh TypingModel (possibly different target)
    m := NewTyping(config.ModeWords, 10, 0, theme.Default(),
        config.DefaultKeymap(), false)
    m2 := m.newTest()
    // Size should be preserved
    m.w, m.h = 80, 24
    m3 := m.newTest()
    if m3.w != 80 || m3.h != 24 {
        t.Error("newTest should preserve terminal size")
    }
}

func TestNewTest_CodeMode_PreservesTarget(t *testing.T) {
    // Code mode: newTest should reuse the same code text
    snippet := "func main() {}"
    m := NewTypingCode(snippet, theme.Default(), config.DefaultKeymap(), false)
    m2 := m.newTest()
    if m2.TargetText() != snippet {
        t.Errorf("code mode newTest should preserve target, got %q", m2.TargetText())
    }
}

func TestApplySettings_UpdatesThemeAndBlink(t *testing.T) {
    m := NewTyping(config.ModeWords, 10, 0, theme.Default(),
        config.DefaultKeymap(), false)
    s := config.Defaults()
    s.BlinkCursor = true
    th := theme.Default()
    m2 := m.ApplySettings(s, th)
    // blink should be updated
    // (white-box: check m2.blink is true)
}
```

## Success Criteria

- [ ] `go test ./internal/typing/... -count=1 -cover` shows `completion.go` ≥90%
- [ ] `go test ./internal/ui/... -count=1` passes with new test file
- [ ] `countCompletedWords` edge cases all covered (empty, single word, trailing space)
- [ ] `restartSame` and `newTest` behavior verified
- [ ] Code mode target preservation verified

## Risk Assessment

- **Low.** Test-only changes. `completion.go` functions are pure (no side effects).
  `screen_typing_actions.go` functions are value-receiver methods on value types.
  Note: some `TypingModel` fields are unexported — tests in `package ui` have
  direct access; if creating in a separate package, use exported methods only.
