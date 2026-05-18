---
phase: 8
title: History & persistence
status: completed
priority: P1
effort: ~5h
dependencies:
  - 6
---

# Phase 8: History & persistence

## Overview

Persist every completed test to a capped JSON history (atomic, XDG data dir), detect new-best, badge it on Result + History, and add the scrollable History screen with trend sparkline. Completes the new-best wiring stubbed in Phase 6.

Refs: researcher-02 §7,8 (TestResult shape); design §5.5 (rows), §8.6 (keys), §2 (success badge); mockups §5.

## Requirements

### Functional
- `internal/storage` history: `$XDG_DATA_HOME/monkeytype-tui/history.json` (fallback `~/.local/share/monkeytype-tui/`). Atomic write. Cap 200 — append, rotate oldest beyond cap. Corrupt/missing → empty history, never crash.
- Record a `TestResult` after every completed test (from `metrics.Result` + mode/length/timestamp).
- New-best detection: highest NetWPM for same mode (+ length for Time/Words) → set `ResultModel.isBest` + `★` in history row.
- History screen: top trend sparkline (`last N` NetWPM), scrollable table `DATE | MODE | WPM | ACC | CONS | ★`, header rules, windowed scroll, `g`/`G` jump top/bottom, empty-state message.
- Keys (§8.6): `↑↓`/`j k` scroll; `g`/`G` top/bottom; `enter` no-op (reserved); `esc`/`1` Home.

### Non-functional
- Atomic, capped, corrupt-safe. Files <200 lines. Unit-tested (rotation, corrupt, XDG fallback, new-best).

## Architecture

Data flow: Phase 6 emits `ResultMsg` → `app` builds `TestResult`, `storage.AppendHistory` (load → append → cap 200 → atomic save), compute `isBest` from loaded history, set on `ResultModel`. History screen loads list on entry.

```go
// internal/storage
type TestResult struct {
    Time       time.Time
    Mode       string   // time|words|quote
    Length     int      // seconds or word count; 0 for quote
    NetWPM, RawWPM, Accuracy, Consistency float64
    Correct, Incorrect, Extra, Missed, Errors int
    DurationMs int64
}
func HistoryPath() (string, error)                 // DataDir()+"/history.json"
func LoadHistory() []TestResult                    // missing/corrupt → []
func AppendHistory(r TestResult) ([]TestResult, error) // load→append→cap200(drop oldest)→atomicWrite
func IsNewBest(hist []TestResult, r TestResult) bool   // max NetWPM same mode(+length)

// internal/ui
type HistoryModel struct {
    rows   []storage.TestResult  // newest first
    top    int                   // scroll window start
    sel    int
    w, h   int
    th     theme.Theme
    km     config.Keymap
}
func NewHistory(rows []storage.TestResult, th theme.Theme, km config.Keymap) HistoryModel
func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd)
func (m HistoryModel) View() string  // sparkline + table + meta; empty-state if len==0
```

Reuses `Sparkline` (Phase 6). Cap rotation: keep last 200 by `Time` (drop oldest front after sort).

## Related Code Files

Create:
- `internal/storage/history-store.go`
- `internal/storage/history-store_test.go` (append/rotation/corrupt/XDG/new-best)
- `internal/ui/screen-history.go`

Modify:
- `internal/app/model.go`, `internal/app/routing.go` (on ResultMsg: append history + compute isBest; `3`/route → real History)
- `internal/ui/screen-result.go` (consume real `isBest` → `★ new best` badge)
- `internal/storage/atomic-write.go` (reuse; no change expected)

Delete: none.

## Implementation Steps

1. `history-store.go`: `LoadHistory` (missing/corrupt → `[]TestResult{}`); `AppendHistory` (load, append, sort by Time, if len>200 drop oldest, atomicWrite via Phase 7 atomic helper); `IsNewBest` (compare NetWPM vs prior same mode (+length for time/words)).
2. `app`: on `ResultMsg` build `TestResult` (now timestamp); `hist := LoadHistory()`; `isBest := IsNewBest(hist, tr)`; `AppendHistory(tr)`; pass `isBest` into `NewResult`.
3. `screen-result.go`: render `★ new best` (`success`) next to WPM when `isBest`.
4. `screen-history.go`: load rows newest-first; trend sparkline of NetWPM (last min(N,len)); table with header rules (§ mockups 5), windowed rows sized to height, selected-row styling §5.5, ACC `success` if ≥95, `★` `success`; meta line `showing a–b of N`; empty-state centered message when len==0. Keys §8.6 (`g`/`G`, scroll, esc/1).
5. Tests: append grows file; 201st entry drops oldest (cap holds); corrupt JSON → empty + app ok; XDG_DATA_HOME vs HOME fallback; `IsNewBest` true on higher WPM same mode, false on lower / different mode.
6. Build, run: complete tests, see history grow, new-best badge appears once then not on lower scores, scroll/g/G work, empty-state on fresh profile. vet/gofmt.

## Success Criteria

- [ ] Every completed test appended atomically to history.json.
- [ ] Cap 200 enforced — oldest rotated out; verified by test.
- [ ] Corrupt/missing history → empty list, app runs; XDG_DATA fallback tested.
- [ ] New-best correctly detected (same mode/length scope); `★ new best` shows on Result + `★` in History.
- [ ] History screen: trend sparkline, scroll, `g`/`G`, windowing, empty-state all work per §8.6/mockups §5.
- [ ] `go test ./internal/storage/... -race` passes; build/vet/gofmt clean.

## Risk Assessment

| Risk | L×I | Mitigation |
|---|---|---|
| Rotation drops wrong (newest) entries | L×H | Sort by Time, drop front (oldest); explicit cap test at 201 |
| New-best scope wrong (cross-mode false positive) | M×M | Scope key = mode (+length for time/words); negative tests |
| Large history slow to load each result | L×M | Cap 200 keeps file small; single read per completion |
| Concurrent write (none in v1 single-process) | L×L | Single-threaded UI loop; atomic rename still safe |
