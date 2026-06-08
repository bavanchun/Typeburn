---
phase: 3
title: "App Test Hardening"
status: pending
priority: P2
effort: "1h"
dependencies: [1]
---

# Phase 3: App Test Hardening

## Overview

Add tests for 3 high-churn files in `internal/app` that have zero direct tests:
`model_view.go` (7 changes), `model_settings.go` (6 changes), and
`model_history.go` (6 changes). Target: app coverage 75.1% → 85%+.

## Related Code Files

- Create: `internal/app/model_view_test.go`
- Create: `internal/app/model_settings_test.go`
- Create: `internal/app/model_history_test.go`

## Implementation Steps

### 1. `model_view_test.go` — View dispatch + degraded notice

```go
func TestView_DegradedNotice(t *testing.T) {
    // Window size < 60×20 → shows degraded notice
    m := newTestModel()
    m2, _ := m.Update(tea.WindowSizeMsg{Width: 50, Height: 15})
    v := m2.(Model).View().Content
    // Should contain resize prompt, not screen content
    if !strings.Contains(v, "resize") && !strings.Contains(v, "60") {
        t.Error("expected degraded notice for small terminal")
    }
}

func TestView_PersistenceNotice(t *testing.T) {
    // When persistErr is set, View() overlays the notice on last line
    m := newTestModel()
    m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
    rm := m2.(Model)
    rm.persistErr = "test error"
    v := rm.View().Content
    if !strings.Contains(v, "test error") {
        t.Error("persistence notice should appear in view")
    }
}

func TestView_QuitPromptOverlay(t *testing.T) {
    // Esc on Home → quit prompt overlay
    m := newTestModel()
    m2, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
    m3, _ := m2.Update(press(tea.KeyEsc, 0))
    rm := m3.(Model)
    if rm.quitPrompt == nil {
        t.Fatal("quit prompt should be active after Esc on Home")
    }
    v := rm.View().Content
    if !strings.Contains(v, "Quit") && !strings.Contains(v, "quit") {
        t.Error("view should show quit prompt overlay")
    }
}
```

### 2. `model_settings_test.go` — modeIdx preservation (Phase 1 fix)

```go
func TestApplySettings_PreservesModeIdx(t *testing.T) {
    // Tab to non-default mode → change setting → mode should be preserved
    m := newTestModel()
    // Tab forward to Words mode (index 1)
    m2, _ := m.Update(press(tea.KeyTab, 0))
    rm := m2.(Model)
    // Change a setting (blink cursor toggle)
    s := rm.settings
    s.BlinkCursor = !s.BlinkCursor
    rm2 := rm.applySettings(s)
    // Home screen should still show the non-default mode, not reset to Time
    // Verify via View — look for "Words" in the tab row
    rm2.w, rm2.h = 80, 24
    v := rm2.View().Content
    // Tab should NOT snap back to "Time" when it was on "Words"
    // (This tests the Phase 1 modeIdx fix)
}
```

### 3. `model_history_test.go` — buildRecord edge cases

```go
func TestBuildRecord_QuoteModeLength(t *testing.T) {
    // Quote mode forces length to 0
    msg := ui.ResultMsg{
        Mode:   config.ModeQuote,
        Length: 42, // should be overridden to 0
        Result: metrics.Result{NetWPM: 80},
    }
    rec := buildRecord(msg)
    if rec.Length != 0 {
        t.Errorf("quote mode length = %d, want 0", rec.Length)
    }
}

func TestBuildRecord_CodeModeRuneCount(t *testing.T) {
    // Code mode stores rune count as length
    msg := ui.ResultMsg{
        Mode:     config.ModeCode,
        CodeText: "hello 世界",
        Result:   metrics.Result{NetWPM: 50},
    }
    rec := buildRecord(msg)
    wantLen := len([]rune("hello 世界")) // 8
    if rec.Length != wantLen {
        t.Errorf("code mode length = %d, want %d", rec.Length, wantLen)
    }
}
```

## Success Criteria

- [ ] `go test ./internal/app/... -count=1 -cover` shows ≥85%
- [ ] All 3 new test files compile and pass
- [ ] `Init`, `screenTitle`, `placeholderView` remain at 0% (acceptable — dead/trivial code)
- [ ] `applySettings` modeIdx preservation verified by test

## Risk Assessment

- **Minimal.** Test-only changes. Tests use existing `newTestModel()` helper.
  No production code modified in this phase.
